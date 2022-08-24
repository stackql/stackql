package util

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jeroenrinzema/psql-wire/pkg/sqldata"
	"github.com/lib/pq/oid"
	openapistackql_util "github.com/stackql/go-openapistackql/pkg/util"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/parserutil"

	"vitess.io/vitess/go/sqltypes"
	"vitess.io/vitess/go/vt/sqlparser"

	querypb "vitess.io/vitess/go/vt/proto/query"

	log "github.com/sirupsen/logrus"
)

var defaultColSortArr []string = []string{
	"id",
	"name",
	"description",
}

var describeRowSortArr []string = []string{
	"id",
	"name",
	"description",
}

func extractExecParams(node *sqlparser.Exec) (map[string]interface{}, error) {
	paramMap := make(map[string]interface{})
	var err error
	for _, varDef := range node.ExecVarDefs {
		key := varDef.ColIdent.GetRawVal()
		switch right := varDef.Val.(type) {
		case *sqlparser.SQLVal:
			paramMap[key] = string(right.Val)
		default:
			return nil, fmt.Errorf("disallowed expression of type '%T' cannot be used for RHS of EXEC parameter", right)
		}
	}
	return paramMap, err
}

func extractInsertParams(insert *sqlparser.Insert, insertValOnlyRows map[int]map[int]interface{}) (map[int]map[string]interface{}, error) {
	retVal := make(map[int]map[string]interface{})
	var err error
	if len(insertValOnlyRows) < 1 {
		return nil, fmt.Errorf("cannot insert zero data")
	}
	for i, row := range insertValOnlyRows {
		rowVal := make(map[string]interface{})
		if len(insert.Columns) != len(row) {
			log.Infoln(fmt.Sprintf("row = %v", row))
			return nil, fmt.Errorf("disparity in fields to insert and supplied data")
		}
		for idx, col := range insert.Columns {
			key := col.GetRawVal()
			val := insertValOnlyRows[i][idx]
			rowVal[key] = val
		}
		retVal[i] = rowVal
	}
	return retVal, err
}

func extractUpdateParams(update *sqlparser.Update, insertValOnlyRows map[int]map[int]interface{}) (map[int]map[string]interface{}, error) {
	retVal := make(map[int]map[string]interface{})
	var err error
	if len(insertValOnlyRows) < 1 {
		return nil, fmt.Errorf("cannot insert zero data")
	}
	lookupMap := make(map[string]*sqlparser.ColName)
	var columnOrder []string
	for _, row := range update.Exprs {
		lookupMap[row.Name.GetRawVal()] = row.Name
		columnOrder = append(columnOrder, row.Name.GetRawVal())
	}
	sqlparser.Walk(func(node sqlparser.SQLNode) (bool, error) {
		switch node := node.(type) {
		case *sqlparser.ComparisonExpr:
			if node.Operator == sqlparser.EqualStr {
				switch l := node.Left.(type) {
				case *sqlparser.ColName:
					key := l.Name.GetRawVal()
					lookupMap[key] = l
					columnOrder = append(columnOrder, key)
					switch r := node.Right.(type) {
					case *sqlparser.SQLVal:
						// val := string(r.Val)
						// paramMap[key] = val
					case *sqlparser.ColName:
						return true, fmt.Errorf("cannot accomodate LHS and RHS col references in update where clause")
					default:
						err = fmt.Errorf("unsupported type on RHS of comparison '%T', FYI LHS type is '%T'", r, l)
						return true, err
					}
				case *sqlparser.FuncExpr:
				default:
					err = fmt.Errorf("failed to analyse left node of comparison")
					return true, err
				}
			}
		}
		return true, err
	}, update.Where)

	sort.Strings(columnOrder)
	for i, valRow := range insertValOnlyRows {
		rowMap := make(map[string]interface{})
		for idx, k := range columnOrder {
			rowMap[k] = valRow[idx]
		}
		retVal[i] = rowMap
	}
	return retVal, err
}

func ExtractSQLNodeParams(statement sqlparser.SQLNode, insertValOnlyRows map[int]map[int]interface{}) (map[int]map[string]interface{}, error) {
	switch stmt := statement.(type) {
	case *sqlparser.Exec:
		val, err := extractExecParams(stmt)
		return map[int]map[string]interface{}{0: val}, err
	case *sqlparser.Insert:
		return extractInsertParams(stmt, insertValOnlyRows)
	case *sqlparser.Update:
		return extractUpdateParams(stmt, insertValOnlyRows)
	}
	paramMap := make(map[string]interface{})
	var err error
	sqlparser.Walk(func(node sqlparser.SQLNode) (bool, error) {
		switch node := node.(type) {
		case *sqlparser.ComparisonExpr:
			if node.Operator == sqlparser.EqualStr {
				switch l := node.Left.(type) {
				case *sqlparser.ColName:
					key := l.Name.GetRawVal()
					switch r := node.Right.(type) {
					case *sqlparser.SQLVal:
						val := string(r.Val)
						paramMap[key] = val
					case *sqlparser.ColName:
						kr := r.Name.GetRawVal()
						paramMap[key] = kr
					default:
						err = fmt.Errorf("unsupported type on RHS of comparison '%T', FYI LHS type is '%T'", node.Right, l)
						return true, err
					}
				case *sqlparser.FuncExpr:
				default:
					err = fmt.Errorf("failed to analyse left node of comparison")
					return true, err
				}
			}
		}
		return true, err
	}, statement)
	return map[int]map[string]interface{}{0: paramMap}, err
}

func TransformSQLRawParameters(input map[string]interface{}) (map[string]interface{}, error) {
	rv := make(map[string]interface{})
	for k, v := range input {
		switch v := v.(type) {
		case *sqlparser.FuncExpr:
			log.Infof("%v\n", v)
			continue
		case parserutil.ParameterMetadata:
			switch t := v.GetVal().(type) {
			case *sqlparser.FuncExpr:
				log.Infof("%v\n", t)
				continue
			}
		}
		r, err := extractRaw(v)
		if err != nil {
			return nil, err
		}
		rv[k] = r
	}
	return rv, nil
}

func extractRaw(raw interface{}) (string, error) {
	switch r := raw.(type) {
	case *sqlparser.SQLVal:
		val := string(r.Val)
		return val, nil
	case *sqlparser.ColName:
		kr := r.Name.GetRawVal()
		return kr, nil
	case parserutil.ParameterMetadata:
		return extractRaw(r.GetVal())
	default:
		err := fmt.Errorf("unsupported type on RHS of comparison '%T'", r)
		return "", err
	}
}

func arrangeOrderedColumnRow(row map[string]interface{}, columns []sqldata.ISQLColumn, columnOrder []string, colNumber int) []interface{} {
	rowVals := make([]interface{}, colNumber)
	for j := range columnOrder {
		rowVals[j] = openapistackql_util.InterfaceToBytes(row[columnOrder[j]], strings.ToLower(columnOrder[j]) == "error")
	}
	return rowVals
}

func DefaultRowSort(rowMap map[string]map[string]interface{}) []string {
	var keys []string
	for k := range rowMap {
		keys = append(keys, k)
	}
	if keys != nil {
		sort.Strings(keys)
		return keys
	}
	return []string{}
}

func GenerateSimpleErroneousOutput(err error) dto.ExecutorOutput {
	return PrepareResultSet(
		dto.NewPrepareResultSetDTO(
			nil,
			nil,
			nil,
			nil,
			err,
			nil,
		),
	)
}

func PrepareResultSet(payload dto.PrepareResultSetDTO) dto.ExecutorOutput {
	if payload.Err != nil {
		return dto.NewExecutorOutput(
			nil,
			payload.OutputBody,
			nil,
			payload.Msg,
			payload.Err,
		)
	}
	if payload.RowMap == nil || len(payload.RowMap) == 0 {
		return dto.NewExecutorOutput(
			nil,
			payload.OutputBody,
			nil,
			payload.Msg,
			nil,
		)
	}
	if payload.RowSort == nil {
		payload.RowSort = DefaultRowSort
	}

	// infer col count
	var colNumber int
	var sampleRow map[string]interface{}
	if payload.ColumnOrder != nil {
		colNumber = len(payload.ColumnOrder)
	} else {
		for k := range payload.RowMap {
			sampleRow = payload.RowMap[k]
			colNumber = len(sampleRow)
			break
		}
	}

	table := sqldata.NewSQLTable(0, "meta_table")
	columns := make([]sqldata.ISQLColumn, colNumber)
	rows := make([]sqldata.ISQLRow, len(payload.RowMap))

	rowsVisited := make(map[string]bool, len(payload.RowMap))
	if payload.ColumnOrder != nil && len(payload.ColumnOrder) > 0 {
		for f := range columns {
			columns[f] = getPlaceholderColumn(table, payload.ColumnOrder[f])
		}
		i := 0
		for _, key := range payload.RowSort(payload.RowMap) {
			if !rowsVisited[key] && payload.RowMap[key] != nil {
				rowVals := arrangeOrderedColumnRow(payload.RowMap[key], columns, payload.ColumnOrder, colNumber)
				rows[i] = sqldata.NewSQLRow(rowVals)
				rowsVisited[key] = true
				i++
			}
		}
		for key, row := range payload.RowMap {
			if !rowsVisited[key] {
				rowVals := arrangeOrderedColumnRow(row, columns, payload.ColumnOrder, colNumber)
				rows[i] = sqldata.NewSQLRow(rowVals)
				rowsVisited[key] = true
				i++
			}
		}
	} else {
		colIdx := 0
		payload.ColumnOrder = make([]string, len(sampleRow))
		colSet := make(map[string]bool, len(sampleRow))
		for k := range sampleRow {
			colSet[k] = false
		}
		for _, k := range defaultColSortArr {
			if _, isPresent := sampleRow[k]; isPresent {
				columns[colIdx] = getPlaceholderColumn(table, k)
				payload.ColumnOrder[colIdx] = k
				colIdx++
				colSet[k] = true
			}
		}
		for k := range sampleRow {
			if !colSet[k] {
				columns[colIdx] = getPlaceholderColumn(table, k)
				payload.ColumnOrder[colIdx] = k
				colIdx++
				colSet[k] = true
			}
		}
		i := 0
		for _, key := range payload.RowSort(payload.RowMap) {
			if !rowsVisited[key] && payload.RowMap[key] != nil {
				rowVals := arrangeOrderedColumnRow(payload.RowMap[key], columns, payload.ColumnOrder, colIdx)
				rows[i] = sqldata.NewSQLRow(rowVals)
				rowsVisited[key] = true
				i++
			}
		}
		for key, row := range payload.RowMap {
			if !rowsVisited[key] {
				rowVals := arrangeOrderedColumnRow(row, columns, payload.ColumnOrder, colIdx)
				rows[i] = sqldata.NewSQLRow(rowVals)
				rowsVisited[key] = true
				i++
			}
		}
	}
	resultStream := sqldata.NewChannelSQLResultStream()
	rv := dto.NewExecutorOutput(
		resultStream,
		payload.OutputBody,
		payload.RawRows,
		payload.Msg,
		payload.Err,
	)
	resultStream.Write(sqldata.NewSQLResult(columns, 0, 0, rows))
	resultStream.Close()
	return rv
}

func EmptyProtectResultSet(rv dto.ExecutorOutput, columns []string) dto.ExecutorOutput {
	if rv.GetRawResult().IsNil() {
		resVal := &sqltypes.Result{
			Fields: make([]*querypb.Field, len(columns)),
		}
		for f := range resVal.Fields {
			resVal.Fields[f] = &querypb.Field{
				Name: columns[f],
			}
		}
		table := sqldata.NewSQLTable(0, "meta_table")
		rCols := make([]sqldata.ISQLColumn, len(columns))
		for f := range rCols {
			rCols[f] = getPlaceholderColumn(table, columns[f])
		}
		rv.GetSQLResult = func() sqldata.ISQLResultStream {
			return sqldata.NewSimpleSQLResultStream(sqldata.NewSQLResult(rCols, 0, 0, nil))
		}
	}
	return rv
}

func DescribeRowSort(rows map[string]map[string]interface{}) []string {
	return describeRowSortArr
}

func GetPlaceholderColumn(table sqldata.ISQLTable, colName string) sqldata.ISQLColumn {
	return getPlaceholderColumn(table, colName)
}

func getPlaceholderColumn(table sqldata.ISQLTable, colName string) sqldata.ISQLColumn {
	return sqldata.NewSQLColumn(
		table,
		colName,
		0,
		uint32(oid.T_text),
		1024,
		0,
		"TextFormat",
	)
}

func GetHeaderOnlyResultStream(colz []string) sqldata.ISQLResultStream {
	table := sqldata.NewSQLTable(0, "table_meta")
	columns := make([]sqldata.ISQLColumn, len(colz))
	for i := range colz {
		columns[i] = getPlaceholderColumn(table, colz[i])
	}
	return sqldata.NewSimpleSQLResultStream(sqldata.NewSQLResult(columns, 0, 0, nil))

}
