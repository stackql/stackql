package util

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/stackql/stackql/internal/stackql/dto"

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

func ExtractSQLNodeParams(statement sqlparser.SQLNode, insertValOnlyRows map[int]map[int]interface{}) (map[int]map[string]interface{}, error) {
	switch stmt := statement.(type) {
	case *sqlparser.Exec:
		val, err := extractExecParams(stmt)
		return map[int]map[string]interface{}{0: val}, err
	case *sqlparser.Insert:
		return extractInsertParams(stmt, insertValOnlyRows)
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
					default:
						err = fmt.Errorf("failed to analyse left node of comparison")
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

func InterfaceToBytes(subject interface{}, isErrorCol bool) []byte {
	switch sub := subject.(type) {
	case bool, sqlparser.BoolVal:
		if sub == true {
			return []byte("true")
		}
		return []byte("false")
	case string:
		return []byte(sub)
	case int:
		return []byte(strconv.Itoa(sub))
	case float32:
		return []byte(fmt.Sprintf("%f", sub))
	case float64:
		return []byte(fmt.Sprintf("%f", sub))
	case []interface{}:
		str, err := json.Marshal(subject)
		if err == nil {
			return []byte(str)
		}
		return []byte(fmt.Sprintf(`{ "marshallingError": {"type": "array", "error": "%s"}}`, err.Error()))
	case map[string]interface{}:
		str, err := json.Marshal(subject)
		if err == nil {
			return []byte(str)
		}
		return []byte(fmt.Sprintf(`{ "marshallingError": {"type": "array", "error": "%s"}}`, err.Error()))
	case nil:
		return []byte("null")
	default:
		return []byte(fmt.Sprintf(`{ "displayError": {"type": "%T", "error": "currently unable to represent object of type %T"}}`, subject, subject))
	}
}

func arrangeOrderedColumnRow(row map[string]interface{}, columnOrder []string, colNumber int) []sqltypes.Value {
	rowVals := make([]sqltypes.Value, colNumber)
	for j := range columnOrder {
		rvj, _ := sqltypes.NewValue(querypb.Type_TEXT, InterfaceToBytes(row[columnOrder[j]], strings.ToLower(columnOrder[j]) == "error"))
		rowVals[j] = rvj
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

	res := &sqltypes.Result{
		Fields: make([]*querypb.Field, colNumber),
		Rows:   make([][]sqltypes.Value, len(payload.RowMap)),
	}

	rowsVisited := make(map[string]bool, len(payload.RowMap))
	if payload.ColumnOrder != nil && len(payload.ColumnOrder) > 0 {
		for f := range res.Fields {
			res.Fields[f] = &querypb.Field{
				Name: payload.ColumnOrder[f],
			}
		}
		i := 0
		for _, key := range payload.RowSort(payload.RowMap) {
			if !rowsVisited[key] && payload.RowMap[key] != nil {
				rowVals := arrangeOrderedColumnRow(payload.RowMap[key], payload.ColumnOrder, colNumber)
				res.Rows[i] = rowVals
				rowsVisited[key] = true
				i++
			}
		}
		for key, row := range payload.RowMap {
			if !rowsVisited[key] {
				rowVals := arrangeOrderedColumnRow(row, payload.ColumnOrder, colNumber)
				res.Rows[i] = rowVals
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
				res.Fields[colIdx] = &querypb.Field{
					Name: k,
				}
				payload.ColumnOrder[colIdx] = k
				colIdx++
				colSet[k] = true
			}
		}
		for k := range sampleRow {
			if !colSet[k] {
				res.Fields[colIdx] = &querypb.Field{
					Name: k,
				}
				payload.ColumnOrder[colIdx] = k
				colIdx++
				colSet[k] = true
			}
		}
		i := 0
		for _, key := range payload.RowSort(payload.RowMap) {
			if !rowsVisited[key] && payload.RowMap[key] != nil {
				rowVals := arrangeOrderedColumnRow(payload.RowMap[key], payload.ColumnOrder, colIdx)
				res.Rows[i] = rowVals
				rowsVisited[key] = true
				i++
			}
		}
		for key, row := range payload.RowMap {
			if !rowsVisited[key] {
				rowVals := arrangeOrderedColumnRow(row, payload.ColumnOrder, colIdx)
				res.Rows[i] = rowVals
				rowsVisited[key] = true
				i++
			}
		}
	}
	return dto.NewExecutorOutput(
		res,
		payload.OutputBody,
		payload.RawRows,
		payload.Msg,
		payload.Err,
	)
}

func EmptyProtectResultSet(rv dto.ExecutorOutput, columns []string) dto.ExecutorOutput {
	if len(rv.GetRawResult()) == 0 {
		resVal := &sqltypes.Result{
			Fields: make([]*querypb.Field, len(columns)),
		}
		for f := range resVal.Fields {
			resVal.Fields[f] = &querypb.Field{
				Name: columns[f],
			}
		}
		rv.GetSQLResult = func() *sqltypes.Result { return resVal }
	}
	return rv
}

func DescribeRowSort(rows map[string]map[string]interface{}) []string {
	return describeRowSortArr
}
