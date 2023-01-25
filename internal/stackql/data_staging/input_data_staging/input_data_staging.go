package input_data_staging

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/jeroenrinzema/psql-wire/pkg/sqldata"
	"github.com/lib/pq/oid"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
)

type NativeResultSetPreparator interface {
	PrepareNativeResultSet() internaldto.ExecutorOutput
}

type naiveNativeResultSetPreparator struct {
	rows                       *sql.Rows
	insertPreparedStatementCtx drm.PreparedStatementCtx
	drmCfg                     drm.DRMConfig
}

func NewNaiveNativeResultSetPreparator(rows *sql.Rows, drmCfg drm.DRMConfig, insertPreparedStatementCtx drm.PreparedStatementCtx) NativeResultSetPreparator {
	return &naiveNativeResultSetPreparator{
		rows:                       rows,
		insertPreparedStatementCtx: insertPreparedStatementCtx,
		drmCfg:                     drmCfg,
	}
}

func getRowDict(colz []string, rowData []any) (map[string]interface{}, error) {
	rv := make(map[string]interface{})
	if len(colz) != len(rowData) {
		return rv, fmt.Errorf("cannot assemble row dict, len(colz) ((%d)) != len(rowData) ((%d))", len(colz), len(rowData))
	}
	for i, k := range colz {
		datum := rowData[i]
		rv[k] = datum
	}
	return rv, nil
}

func (np *naiveNativeResultSetPreparator) PrepareNativeResultSet() internaldto.ExecutorOutput {
	rows := np.rows
	if rows == nil {
		emptyResult := internaldto.NewExecutorOutput(
			nil,
			nil,
			nil,
			&internaldto.BackendMessages{WorkingMessages: []string{"native sql nil result set"}},
			nil,
		)
		return nativeProtect(emptyResult, []string{"error"})
	}
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nativeProtect(internaldto.NewErroneousExecutorOutput(err), []string{"error"})
	}

	columns := getColumnArr(colTypes)

	var colz []string

	for _, c := range colTypes {
		colz = append(colz, c.Name())
	}

	var outRows []sqldata.ISQLRow

	for {
		hasNext := rows.Next()
		if !hasNext {
			break
		}
		rowPtr := getRowPointers(colTypes)
		err = rows.Scan(rowPtr...)
		if err != nil {
			return nativeProtect(internaldto.NewErroneousExecutorOutput(err), []string{"error"})
		}
		dataArr := sqldata.NewSQLRow(rowPtr)
		outRows = append(outRows, dataArr)
		if np.insertPreparedStatementCtx != nil {
			insertInputMap, err := getRowDict(colz, dataArr.GetRowDataForPgWire())
			if err != nil {
				return nativeProtect(internaldto.NewErroneousExecutorOutput(err), []string{"error"})
			}
			_, err = np.drmCfg.ExecuteInsertDML(np.drmCfg.GetSQLSystem().GetSQLEngine(), np.insertPreparedStatementCtx, insertInputMap, "")
			if err != nil {
				return nativeProtect(internaldto.NewErroneousExecutorOutput(err), []string{"error"})
			}
		}
	}
	resultStream := sqldata.NewChannelSQLResultStream()
	rv := internaldto.NewExecutorOutput(
		resultStream,
		nil,
		nil,
		nil,
		nil,
	)
	if len(outRows) == 0 {
		outRows = append(outRows, sqldata.NewSQLRow([]interface{}{}))
	}
	resultStream.Write(sqldata.NewSQLResult(columns, 0, 0, outRows))
	resultStream.Close()
	if len(outRows) == 0 {
		nativeProtect(rv, colz)
	}
	return rv
}

func getRowPointers(colTypes []*sql.ColumnType) []any {
	var rowPtr []any

	for _, col := range colTypes {
		rowPtr = append(rowPtr, getScannableObjectForNativeResult(col))
	}
	return rowPtr
}

func getScannableObjectForNativeResult(colSchema *sql.ColumnType) any {
	switch strings.ToLower(colSchema.DatabaseTypeName()) {
	case "int", "int32", "smallint", "tinyint":
		return new(sql.NullInt32)
	case "uint", "uint32":
		return new(sql.NullInt64)
	case "int64", "bigint":
		return new(sql.NullInt64)
	case "numeric", "decimal", "float", "float32", "float64":
		return new(sql.NullFloat64)
	case "bool":
		return new(sql.NullBool)
	default:
		return new(sql.NullString)
	}
}

func getColumnArr(colTypes []*sql.ColumnType) []sqldata.ISQLColumn {
	var columns []sqldata.ISQLColumn

	table := sqldata.NewSQLTable(0, "meta_table")

	for _, col := range colTypes {
		columns = append(columns, getPlaceholderColumnForNativeResult(table, col.Name(), col))
	}
	return columns
}

func getDefaultOID() oid.Oid {
	return oid.T_text
}

func getOidForSQLDatabaseTypeName(typeName string) oid.Oid {
	typeNameLowered := strings.ToLower(typeName)
	switch strings.ToLower(typeNameLowered) {
	case "object", "array":
		return oid.T_text
	case "boolean", "bool":
		return oid.T_text
	case "number", "int", "bigint", "smallint", "tinyint":
		return oid.T_numeric
	default:
		return oid.T_text
	}
}

func getPlaceholderColumnForNativeResult(table sqldata.ISQLTable, colName string, colSchema *sql.ColumnType) sqldata.ISQLColumn {
	return sqldata.NewSQLColumn(
		table,
		colName,
		0,
		uint32(getOidForSQLType(colSchema)),
		1024,
		0,
		"TextFormat",
	)
}

func getOidForSQLType(colType *sql.ColumnType) oid.Oid {
	if colType == nil {
		return oid.T_text
	}
	return getOidForSQLDatabaseTypeName(colType.DatabaseTypeName())
}

func nativeProtect(rv internaldto.ExecutorOutput, columns []string) internaldto.ExecutorOutput {
	if rv.GetSQLResult() == nil {
		table := sqldata.NewSQLTable(0, "meta_table")
		rCols := make([]sqldata.ISQLColumn, len(columns))
		for f := range rCols {
			rCols[f] = getPlaceholderColumn(table, columns[f], getDefaultOID())
		}
		rv.GetSQLResult = func() sqldata.ISQLResultStream {
			return sqldata.NewSimpleSQLResultStream(sqldata.NewSQLResult(rCols, 0, 0, []sqldata.ISQLRow{
				sqldata.NewSQLRow([]interface{}{}),
			}))
		}
	}
	return rv
}

func getPlaceholderColumn(table sqldata.ISQLTable, colName string, colOID oid.Oid) sqldata.ISQLColumn {
	return sqldata.NewSQLColumn(
		table,
		colName,
		0,
		uint32(colOID),
		1024,
		0,
		"TextFormat",
	)
}
