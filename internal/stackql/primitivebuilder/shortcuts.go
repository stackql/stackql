package primitivebuilder

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/stackql/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/buildinfo"
	"github.com/stackql/stackql/internal/stackql/contributors"
	"github.com/stackql/stackql/internal/stackql/dependencies"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/intrinsic"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/provider"
	"github.com/stackql/stackql/internal/stackql/util"
)

func NewUpdateableValsPrimitive(
	handlerCtx handler.HandlerContext,
	vals map[*sqlparser.ColName]interface{},
) (primitive.IPrimitive, error) {
	return primitive.NewLocalPrimitive(
		//nolint:revive // no big deal
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			keys := make(map[string]map[string]interface{})
			row := make(map[string]interface{})
			rawRow := make(map[int]interface{})
			var columnOrder []string
			i := 0
			lookupMap := make(map[string]*sqlparser.ColName)
			for k := range vals {
				columnOrder = append(columnOrder, k.Name.GetRawVal())
				lookupMap[k.Name.GetRawVal()] = k
			}
			sort.Strings(columnOrder)
			for _, rk := range columnOrder {
				k := lookupMap[rk]
				v := vals[k]
				row[k.Name.GetRawVal()] = v
				rawRow[i] = v
				i++
			}
			keys["0"] = row
			rawRows := map[int]map[int]interface{}{
				0: rawRow,
			}
			return util.PrepareResultSet(
				internaldto.NewPrepareResultSetPlusRawDTO(
					nil, keys, columnOrder, nil, nil, nil, rawRows,
					handlerCtx.GetTypingConfig()),
			)
		}), nil
}

func NewInsertableValsPrimitive(
	handlerCtx handler.HandlerContext,
	vals map[int]map[int]interface{},
) (primitive.IPrimitive, error) {
	return primitive.NewLocalPrimitive(
		//nolint:revive // acceptable for now
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			keys := make(map[string]map[string]interface{})
			row := make(map[string]interface{})
			var rowKeys []int
			var colKeys []int
			var columnOrder []string
			for k := range vals {
				rowKeys = append(rowKeys, k)
			}
			for _, v := range vals {
				for ck := range v {
					colKeys = append(colKeys, ck)
				}
				break
			}
			sort.Ints(rowKeys)
			sort.Ints(colKeys)
			for _, ck := range colKeys {
				columnOrder = append(columnOrder, "val_"+strconv.Itoa(ck))
			}
			for idx := range colKeys {
				col := vals[0][idx]
				colName := columnOrder[idx]
				row[colName] = col
			}
			keys["0"] = row
			return util.PrepareResultSet(internaldto.NewPrepareResultSetPlusRawDTO(nil, keys, columnOrder, nil, nil, nil, vals,
				handlerCtx.GetTypingConfig()))
		}), nil
}

func buildDependenciesShowOutput(
	node *sqlparser.Show, extended bool, handlerCtx handler.HandlerContext,
) internaldto.ExecutorOutput {
	var pattern *string
	if node.ShowTablesOpt != nil && node.ShowTablesOpt.Filter != nil {
		pattern = &node.ShowTablesOpt.Filter.Like
	}
	keys, err := dependencies.Rows(extended, pattern)
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	columns := dependencies.Columns(extended)
	table := sqldata.NewSQLTable(0, "meta_table")
	sqlColumns := make([]sqldata.ISQLColumn, len(columns))
	for i, column := range columns {
		sqlColumns[i] = handlerCtx.GetTypingConfig().GetPlaceholderColumn(
			table, column, handlerCtx.GetTypingConfig().GetDefaultOID())
	}
	sqlRows := make([]sqldata.ISQLRow, 0, len(keys))
	for _, key := range util.DefaultRowSort(keys) {
		values := make([]interface{}, len(columns))
		for i, column := range columns {
			if value, ok := keys[key][column].(string); ok {
				values[i] = []byte(value)
			}
		}
		sqlRows = append(sqlRows, newDependencySQLRow(values))
	}
	stream := sqldata.NewSimpleSQLResultStream(sqldata.NewSQLResult(sqlColumns, 0, 0, sqlRows))
	return internaldto.NewExecutorOutput(stream, nil, nil, nil, nil)
}

type dependencySQLRow struct {
	sqldata.ISQLRow
}

func newDependencySQLRow(values []interface{}) sqldata.ISQLRow {
	return &dependencySQLRow{ISQLRow: sqldata.NewSQLRow(values)}
}

func (row *dependencySQLRow) GetRowDataForPgWire() []interface{} {
	values := row.ISQLRow.GetRowDataForPgWire()
	for i, value := range values {
		if value == nil {
			// The wire text encoder distinguishes a nil byte slice from an empty string.
			values[i] = []byte(nil)
		}
	}
	return values
}

// buildVersionShowOutput renders the SHOW VERSION result. Extracted so that
// NewShowInstructionExecutor stays under the gocyclo threshold.
func buildVersionShowOutput(extended bool) ([]string, map[string]map[string]interface{}) {
	bi := buildinfo.Get()
	if extended {
		return []string{"version", "commit", "build_date", "platform"},
			map[string]map[string]interface{}{
				"1": {
					"version":    bi.GetSemVersion(),
					"commit":     bi.GetShortCommitSHA(),
					"build_date": bi.GetDate(),
					"platform":   bi.GetPlatform(),
				},
			}
	}
	return []string{"version"},
		map[string]map[string]interface{}{
			"1": {"version": bi.GetSemVersion()},
		}
}

// buildContributorsShowOutput renders SHOW [EXTENDED] CONTRIBUTORS; keys are
// zero-padded so the sorted result set keeps leaderboard order.
func buildContributorsShowOutput(extended bool) ([]string, map[string]map[string]interface{}) {
	keys := make(map[string]map[string]interface{})
	for i, c := range contributors.List() {
		row := map[string]interface{}{"contributor": c.GetLogin()}
		if extended {
			row["contributions"] = c.GetContributions()
		}
		keys[fmt.Sprintf("%06d", i)] = row
	}
	if extended {
		return []string{"contributor", "contributions"}, keys
	}
	return []string{"contributor"}, keys
}

//nolint:revive // future proofing
func NewLocalSelectExecutor(
	handlerCtx handler.HandlerContext,
	node *sqlparser.Select,
	rowSort func(map[string]map[string]interface{}) []string,
	colz []map[string]interface{},
) (primitive.IPrimitive, error) {
	return primitive.NewLocalPrimitive(
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			var columnOrder []string
			keys := make(map[string]map[string]interface{})
			row := make(map[string]interface{})
			for idx := range colz {
				col := colz[idx]
				if col != nil {
					var alias string
					var val interface{}
					for k, v := range col {
						alias = k
						val = v
						break
					}
					if alias == "" {
						alias = "val_" + strconv.Itoa(idx)
					}
					row[alias] = val
					columnOrder = append(columnOrder, alias)
				}
			}
			keys["0"] = row
			return util.PrepareResultSet(internaldto.NewPrepareResultSetDTO(nil, keys, columnOrder, rowSort, nil, nil,
				handlerCtx.GetTypingConfig(),
			))
		}), nil
}

func NewShowInstructionExecutor(
	node *sqlparser.Show, prov provider.IProvider, handlerCtx handler.HandlerContext,
) internaldto.ExecutorOutput {
	if execute, handled := intrinsic.GeneratePrimitiveFunc(handlerCtx, node); handled {
		return execute()
	}
	extended := strings.EqualFold(strings.TrimSpace(node.Extended), "extended")
	var rows map[string]map[string]interface{}
	var columns []string
	switch strings.ToUpper(node.Type) {
	case "AUTH":
		authCtx, err := handlerCtx.GetAuthContext(prov.GetProviderString())
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		metadata, err := prov.ShowAuth(authCtx)
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		rows, columns = map[string]map[string]interface{}{"1": metadata.ToMap()}, []string{"principal", "type", "source"}
	case "PROVIDERS":
		var err error
		rows, err = handlerCtx.GetSupportedProviders(extended)
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		if len(rows) == 0 {
			columns = []string{"name", "version"}
		}
	case "VERSION":
		columns, rows = buildVersionShowOutput(extended)
	case "CONTRIBUTORS":
		columns, rows = buildContributorsShowOutput(extended)
	case "DEPENDENCIES":
		return buildDependenciesShowOutput(node, extended, handlerCtx)
	default:
		return internaldto.NewErroneousExecutorOutput(fmt.Errorf("unsupported SHOW %s", node.Type))
	}
	result := util.PrepareResultSet(internaldto.NewPrepareResultSetDTO(
		nil, rows, columns, nil, nil, nil, handlerCtx.GetTypingConfig()))
	if len(rows) == 0 {
		return util.EmptyProtectResultSet(result, columns, handlerCtx.GetTypingConfig())
	}
	return result
}
