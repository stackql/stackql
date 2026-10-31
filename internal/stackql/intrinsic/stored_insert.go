package intrinsic

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/sql_system"
)

// A host table target keeps its SQL write in the host while its provider source
// follows the same Omni planning and staging path as an ordinary SELECT.
func storedInsertSelectFunc(ctx queryContext, stmt sqlparser.Statement) (func() internaldto.ExecutorOutput, bool) {
	insert, ok := stmt.(*sqlparser.Insert)
	if !ok {
		return nil, false
	}
	source, ok := insert.Rows.(*sqlparser.Select)
	if !ok || literalSource(source.From) {
		return nil, false
	}
	host, ok := ctx.(interface{ GetSQLSystem() sql_system.SQLSystem })
	if !ok || host.GetSQLSystem() == nil {
		return nil, false
	}
	system := host.GetSQLSystem()
	name := qualifiedName(insert.Table)
	table, present := system.GetPhysicalTableByName(name)
	if !present {
		return nil, false
	}
	if insert.Ignore != "" || len(insert.OnDup) > 0 {
		return refuse(fmt.Errorf("host table INSERT source does not support conflict clauses")), true
	}
	target := system.DelimitFullyQualifiedRelationName(table.GetName())
	prepared, relations, err := prepareMutationRelations(ctx, insert, ctx.GetCurrentProvider())
	if err != nil {
		return refuse(err), true
	}
	preparedInsert, ok := prepared.(*sqlparser.Insert)
	if !ok {
		return refuse(fmt.Errorf("prepared insert has unexpected statement type")), true
	}
	preparedSource, ok := preparedInsert.Rows.(*sqlparser.Select)
	if !ok {
		return refuse(fmt.Errorf("prepared insert requires SELECT source")), true
	}
	dialect, err := backendDialect(ctx)
	if err != nil {
		return refuse(err), true
	}
	translator := newDocTranslator(ctx.GetCurrentProvider(), dialect)
	staged, err := planStagedSelect(preparedSource, translator, ctx.GetASTFormatter())
	if err != nil {
		return refuse(err), true
	}
	staged = withStagedRelations(staged, relations)
	columns := make([]string, len(insert.Columns))
	for i, column := range insert.Columns {
		columns[i] = quoteRelationIdentifier(column.GetRawVal())
	}
	prefix := "INSERT INTO " + target
	if len(columns) > 0 {
		prefix += " (" + strings.Join(columns, ",") + ")"
	}
	return func() internaldto.ExecutorOutput {
		return runStoredInsert(ctx, staged, prefix)
	}, true
}

func runStoredInsert(ctx queryContext, staged stagedSelect, prefix string) internaldto.ExecutorOutput {
	bg := executionContext(ctx)
	manager, err := getStagingManager(ctx)
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	id, err := manager.Begin(bg)
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	defer manager.Release(bg, id) //nolint:errcheck // Query cleanup does not alter the insert outcome.
	rows, _, err := openDocQueryIn(ctx, staged.getSource(), bg)
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	defer rows.Close()
	source := newOmniBatchSource(rows, staged.getColumns(), previewCfg.getBatchSize())
	first, err := source.Next(bg)
	if err != nil && !errors.Is(err, io.EOF) {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	source.pending = first
	if _, err = manager.Stage(bg, id, stagedColumns(staged.getColumns(), first), source); err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	if source.terminalErr != nil {
		return internaldto.NewErroneousExecutorOutput(source.terminalErr)
	}
	db, err := ctx.GetSQLEngine().GetDB()
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	//nolint:gosec // G202: target is registered metadata, source is backend-formatted SQL.
	if _, err = db.ExecContext(bg, prefix+" "+staged.renderOuter(manager.TableName(id))); err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	return internaldto.NewExecutorOutput(nil, nil, nil, nil, nil)
}
