package intrinsic

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/util"
)

func metadataProviderName(bundle string, filters []*sqlparser.Show) string {
	if len(filters) > 0 && filters[0] != nil {
		name := filters[0].OnTable.Qualifier.GetRawVal()
		if name != "" {
			return name
		}
	}
	return UnstablePrefix + bundle
}

// Metadata filters run in the configured SQL backend, retaining that backend's
// comparison and LIKE semantics. All values are bound parameters.
func prepareDocMetadata(
	ctx queryContext,
	columns []string,
	rows map[string]map[string]interface{},
	filters []*sqlparser.Show,
) internaldto.ExecutorOutput {
	if len(filters) == 0 || filters[0] == nil || filters[0].ShowTablesOpt == nil ||
		filters[0].ShowTablesOpt.Filter == nil {
		return prepare(ctx, columns, rows, util.DefaultRowSort)
	}
	filter := filters[0].ShowTablesOpt.Filter
	var predicate sqlparser.Expr = filter.Filter
	if filter.Like != "" {
		predicate = &sqlparser.ComparisonExpr{
			Operator: sqlparser.LikeStr,
			Left:     &sqlparser.ColName{Name: sqlparser.NewColIdent("name")},
			Right:    sqlparser.NewStrVal([]byte(filter.Like)),
		}
	}
	if predicate == nil {
		return prepare(ctx, columns, rows, util.DefaultRowSort)
	}
	db, err := ctx.GetSQLEngine().GetDB()
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	formatter := sqlparser.NewTrackedBuffer(ctx.GetASTFormatter())
	predicate.Format(formatter)
	for key, row := range rows {
		names := make([]string, 0, len(row))
		for name := range row {
			names = append(names, name)
		}
		sort.Strings(names)
		fields := make([]string, 0, len(names))
		args := make([]any, 0, len(names))
		for i, name := range names {
			fields = append(fields, fmt.Sprintf("$%d AS \"%s\"", i+1, strings.ReplaceAll(name, "\"", "\"\"")))
			args = append(args, row[name])
		}
		statement := "SELECT 1 FROM (SELECT " + strings.Join(fields, ",") + ") AS metadata WHERE " + formatter.String()
		var found int
		err = db.QueryRow(statement, args...).Scan(&found)
		if errors.Is(err, sql.ErrNoRows) {
			delete(rows, key)
		} else if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
	}
	return prepare(ctx, columns, rows, util.DefaultRowSort)
}
