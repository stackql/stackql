package input_data_staging //nolint:revive,stylecheck // package name is helpful

import (
	"database/sql"
	"fmt"

	"github.com/stackql/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/typing"
)

type NativeResultSetPreparator interface {
	PrepareNativeResultSet() internaldto.ExecutorOutput
}

type naiveNativeResultSetPreparator struct {
	rows                       *sql.Rows
	insertPreparedStatementCtx drm.PreparedStatementCtx
	drmCfg                     drm.Config
	typCfg                     typing.Config
}

func NewNaiveNativeResultSetPreparator(
	rows *sql.Rows,
	drmCfg drm.Config,
	typCfg typing.Config,
	insertPreparedStatementCtx drm.PreparedStatementCtx,
) NativeResultSetPreparator {
	return &naiveNativeResultSetPreparator{
		rows:                       rows,
		insertPreparedStatementCtx: insertPreparedStatementCtx,
		drmCfg:                     drmCfg,
		typCfg:                     typCfg,
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
			internaldto.NewBackendMessages([]string{"native sql nil result set"}),
			nil,
		)
		return np.nativeProtect(emptyResult, []string{"error"})
	}
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return np.nativeProtect(internaldto.NewErroneousExecutorOutput(err), []string{"error"})
	}

	columns := np.getColumnArr(colTypes)

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
		rowPtr := np.getRowPointers(colTypes)
		err = rows.Scan(rowPtr...)
		if err != nil {
			return np.nativeProtect(internaldto.NewErroneousExecutorOutput(err), []string{"error"})
		}
		dataArr := sqldata.NewSQLRow(rowPtr)
		outRows = append(outRows, dataArr)
		if np.insertPreparedStatementCtx != nil {
			insertInputMap, localErr := getRowDict(colz, dataArr.GetRowDataForPgWire())
			if localErr != nil {
				return np.nativeProtect(internaldto.NewErroneousExecutorOutput(localErr), []string{"error"})
			}
			_, err = np.drmCfg.ExecuteInsertDML(
				np.drmCfg.GetSQLSystem().GetSQLEngine(), np.insertPreparedStatementCtx, insertInputMap, "")
			if err != nil {
				return np.nativeProtect(
					internaldto.NewErroneousExecutorOutput(err), []string{"error"})
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
	resultStream.Write(sqldata.NewSQLResult(columns, 0, 0, outRows)) //nolint:errcheck // output stream
	resultStream.Close()
	if len(outRows) == 0 {
		np.nativeProtect(rv, colz)
	}
	return rv
}

func (np *naiveNativeResultSetPreparator) getRowPointers(colTypes []*sql.ColumnType) []any {
	var rowPtr []any

	for _, col := range colTypes {
		rowPtr = append(rowPtr, np.typCfg.GetScannableObjectForNativeResult(col))
	}
	return rowPtr
}

func (np *naiveNativeResultSetPreparator) getColumnArr(colTypes []*sql.ColumnType) []sqldata.ISQLColumn {
	var columns []sqldata.ISQLColumn

	table := sqldata.NewSQLTable(0, "meta_table")

	for _, col := range colTypes {
		columns = append(columns, np.typCfg.GetPlaceholderColumnForNativeResult(table, col.Name(), col))
	}
	return columns
}

func (np *naiveNativeResultSetPreparator) nativeProtect(
	rv internaldto.ExecutorOutput, columns []string) internaldto.ExecutorOutput {
	if rv.GetSQLResult() == nil {
		table := sqldata.NewSQLTable(0, "meta_table")
		rCols := make([]sqldata.ISQLColumn, len(columns))
		for f := range rCols {
			rCols[f] = np.typCfg.GetPlaceholderColumn(table, columns[f], np.typCfg.GetDefaultOID())
		}
		rv.SetSQLResultFn(func() sqldata.ISQLResultStream {
			return sqldata.NewSimpleSQLResultStream(sqldata.NewSQLResult(rCols, 0, 0, []sqldata.ISQLRow{
				sqldata.NewSQLRow([]interface{}{}),
			}))
		})
	}
	return rv
}
