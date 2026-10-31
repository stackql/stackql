package intrinsic

import (
	"context"
	"fmt"
	"strings"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/datasource/sql_datasource"
)

type sqlSourceProvider interface {
	GetSQLDataSource(string) (sql_datasource.SQLDataSource, bool)
}

type externalSQLRelation interface {
	omnisdk.Relation
	externalSQLSource() bool
}
type externalRelationHandle struct{ omnisdk.Relation }

func (externalRelationHandle) externalSQLSource() bool { return true }
func newExternalRelationHandle(relation omnisdk.Relation) externalSQLRelation {
	return externalRelationHandle{Relation: relation}
}

func externalSource(ctx queryContext, name sqlparser.TableName) (sql_datasource.SQLDataSource, bool) {
	host, ok := ctx.(sqlSourceProvider)
	if !ok {
		return nil, false
	}
	connection := name.QualifierSecond.GetRawVal()
	if connection == "" {
		connection = ctx.GetCurrentProvider()
	}
	if connection == "" {
		return nil, false
	}
	return host.GetSQLDataSource(connection)
}

func hasExternalRelations(relations map[string]omnisdk.Relation) bool {
	for _, relation := range relations {
		if external, ok := relation.(externalSQLRelation); ok && external.externalSQLSource() {
			return true
		}
	}
	return false
}

func (r *storedResolver) registerExternal(
	table *sqlparser.AliasedTableExpr, name sqlparser.TableName, source sql_datasource.SQLDataSource,
) error {
	if err := checkHost(r.ctx, "host/backend", "read"); err != nil {
		return err
	}
	reader, ok := source.(sql_datasource.ContextQuerier)
	if !ok {
		return fmt.Errorf("external source %s does not support cancellable reads", name.QualifierSecond.GetRawVal())
	}
	if name.Qualifier.IsEmpty() {
		return fmt.Errorf("external relation %s requires an explicit schema", qualifiedName(name))
	}
	// SQL identifiers are parsed names and escaped individually; connection names never enter SQL.
	actual := quoteRelationIdentifier(name.Qualifier.GetRawVal()) + "." + quoteRelationIdentifier(name.Name.GetRawVal())
	discovery, cancel := context.WithTimeout(executionContext(r.ctx), omnisdk.DefaultRequestTimeout)
	defer cancel()
	//nolint:gosec // Only individually quoted SQL identifiers enter this metadata query.
	metadata, err := reader.QueryContext(discovery, "SELECT * FROM "+actual+" WHERE 1=0")
	if err != nil {
		return err
	}
	defer metadata.Close()
	columnTypes, err := metadata.ColumnTypes()
	if err != nil {
		return err
	}
	if metadataErr := metadata.Err(); metadataErr != nil {
		return metadataErr
	}
	if len(columnTypes) == 0 {
		return fmt.Errorf("external relation %s has no declared columns", qualifiedName(name))
	}
	names := make([]string, 0, len(columnTypes))
	quoted := make([]string, 0, len(columnTypes))
	columns := map[string]omnisdk.ColumnType{}
	for _, column := range columnTypes {
		names = append(names, column.Name())
		quoted = append(quoted, quoteRelationIdentifier(column.Name()))
		columns[column.Name()] = storedColumnType(column.DatabaseTypeName())
	}
	statement := "SELECT " + strings.Join(quoted, ",") + " FROM " + actual
	open := func(runCtx context.Context, _ map[string]any) (omnisdk.Rows, error) {
		rows, queryErr := reader.QueryContext(runCtx, statement)
		if queryErr != nil {
			return nil, queryErr
		}
		return materializeSQLRows(runCtx, rows, names, nil)
	}
	local := fmt.Sprintf("external%d", len(r.relations))
	handle := UnstablePrefix + "omni_relations.sources." + local
	relation, err := omnisdk.NewRelation(handle, nil, columns, open)
	if err != nil {
		return err
	}
	r.relations[handle] = newExternalRelationHandle(relation)
	table.Expr = sqlparser.TableName{
		QualifierSecond: sqlparser.NewTableIdent(UnstablePrefix + "omni_relations"),
		Qualifier:       sqlparser.NewTableIdent("sources"), Name: sqlparser.NewTableIdent(local),
	}
	if table.As.IsEmpty() {
		table.As = name.Name
	}
	return nil
}
