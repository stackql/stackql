package parserutil

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/stackql/go-openapistackql/openapistackql"

	log "github.com/sirupsen/logrus"
	"vitess.io/vitess/go/vt/sqlparser"
)

const (
	FloatBitSize int = 64
)

func GetTableNameFromTableExpr(node sqlparser.TableExpr) (sqlparser.TableName, error) {
	switch tt := node.(type) {
	case *sqlparser.AliasedTableExpr:
		tn, ok := tt.Expr.(sqlparser.TableName)
		if ok {
			return tn, nil
		}
	}
	return sqlparser.TableName{}, fmt.Errorf("table expression too colmplex")
}

type ColumnHandle struct {
	Alias           string
	Expr            sqlparser.Expr
	Name            string
	DecoratedColumn string
	IsColumn        bool
	Type            sqlparser.ValType
	Val             *sqlparser.SQLVal
}

func NewUnaliasedColumnHandle(name string) ColumnHandle {
	return ColumnHandle{Name: name}
}

func ExtractSelectColumnNames(selStmt *sqlparser.Select) ([]ColumnHandle, error) {
	var colNames []ColumnHandle
	var err error
	for _, node := range selStmt.SelectExprs {
		switch node := node.(type) {
		case *sqlparser.AliasedExpr:
			colNames = append(colNames, inferColNameFromExpr(node))
		case *sqlparser.StarExpr:

		}
	}
	return colNames, err
}

func ExtractInsertColumnNames(insertStmt *sqlparser.Insert) ([]string, error) {
	var colNames []string
	var err error
	for _, node := range insertStmt.Columns {
		colNames = append(colNames, node.GetRawVal())
	}
	return colNames, err
}

func ExtractAliasedValColumnData(aliasedExpr *sqlparser.AliasedExpr) (map[string]interface{}, error) {
	alias := aliasedExpr.As.GetRawVal()
	switch expr := aliasedExpr.Expr.(type) {
	case *sqlparser.SQLVal:
		switch expr.Type {
		case sqlparser.StrVal:
			return map[string]interface{}{alias: string(expr.Val)}, nil
		case sqlparser.IntVal:
			rv, err := strconv.Atoi(string(expr.Val))
			return map[string]interface{}{alias: rv}, err
		case sqlparser.FloatVal:
			rv, err := strconv.ParseFloat(string(expr.Val), FloatBitSize)
			return map[string]interface{}{alias: rv}, err
		}
	}
	return nil, fmt.Errorf("unextractable val only col")
}

func ExtractStringRepresentationOfValueColumn(expr *sqlparser.SQLVal) string {
	if expr == nil {
		return ""
	}
	switch expr.Type {
	case sqlparser.StrVal:
		return fmt.Sprintf(`'%s'`, string(expr.Val))
	case sqlparser.IntVal, sqlparser.FloatVal:
		return string(expr.Val)
	default:
		return string(expr.Val)
	}
}

func ExtractValuesColumnData(values sqlparser.Values) (map[int]map[int]interface{}, int, error) {
	retVal := make(map[int]map[int]interface{})
	var nonValCount int
	var err error
	for outerIdx, valTuple := range values {
		row := make(map[int]interface{})
		for innerIdx, expr := range valTuple {
			switch expr := expr.(type) {
			case *sqlparser.SQLVal:
				switch expr.Type {
				case sqlparser.StrVal:
					row[innerIdx] = string(expr.Val)
				case sqlparser.IntVal:
					rv, err := strconv.Atoi(string(expr.Val))
					if err != nil {
						return nil, nonValCount, fmt.Errorf("error extracting Values integer: %s", err.Error())
					}
					row[innerIdx] = rv
				case sqlparser.FloatVal:
					rv, err := strconv.ParseFloat(string(expr.Val), FloatBitSize)
					if err != nil {
						return nil, nonValCount, fmt.Errorf("error extracting Values float: %s", err.Error())
					}
					row[innerIdx] = rv
				default:
					return nil, nonValCount, fmt.Errorf("unextractable val only col of type %v", expr.Type)
				}
			}
		}
		retVal[outerIdx] = row
	}
	return retVal, 0, err
}

func ExtractSelectValColumns(selStmt *sqlparser.Select) (map[int]map[string]interface{}, int) {
	cols := make(map[int]map[string]interface{})
	var nonValCount int
	for idx, node := range selStmt.SelectExprs {
		switch node := node.(type) {
		case *sqlparser.AliasedExpr:
			switch expr := node.Expr.(type) {
			case *sqlparser.SQLVal:
				col, err := ExtractAliasedValColumnData(node)
				if err == nil {
					cols[idx] = col
				} else {
					cols[idx] = nil
					nonValCount++
				}
			case *sqlparser.OrExpr:
				nonValCount++
			case *sqlparser.FuncExpr:
				nonValCount++
			case *sqlparser.ColName:
				nonValCount++
			case sqlparser.BoolVal:
				cols[idx] = map[string]interface{}{fmt.Sprintf("$$unaliased_col_%d", idx): expr}
			default:
				log.Infoln(fmt.Sprintf("cannot use AliasedExpr of type '%T' as a raw value", expr))
				cols[idx] = nil
				nonValCount++
			}
		default:
			log.Infoln(fmt.Sprintf("cannot use SelectExpr of type '%T' as a raw value", node))
			cols[idx] = nil
			nonValCount++
		}
	}
	return cols, nonValCount
}

func ExtractInsertValColumns(insStmt *sqlparser.Insert) (map[int]map[int]interface{}, int, error) {
	return extractInsertValColumns(insStmt, false)
}

func ExtractInsertValColumnsPlusPlaceHolders(insStmt *sqlparser.Insert) (map[int]map[int]interface{}, int, error) {
	return extractInsertValColumns(insStmt, false)
}

func extractInsertValColumns(insStmt *sqlparser.Insert, includePlaceholders bool) (map[int]map[int]interface{}, int, error) {
	var nonValCount int
	var err error
	switch node := insStmt.Rows.(type) {
	case *sqlparser.Select:
		row, nvc := ExtractSelectValColumns(node)
		transformedRow := make(map[int]interface{})
		for k, v := range row {
			if v != nil {
				for _, c := range v {
					transformedRow[k] = c
					break
				}
			} else {
				if includePlaceholders {
					nvc = 0
					transformedRow[k] = nil
				}
			}
		}
		return map[int]map[int]interface{}{
			0: transformedRow,
		}, nvc, err
	case sqlparser.Values:
		return ExtractValuesColumnData(node)
	default:
		err = fmt.Errorf("cannot use an insert Rows value column of type '%T' as a raw value", node)
	}
	return nil, nonValCount, err
}

func ExtractWhereColNames(statement *sqlparser.Where) ([]string, error) {
	var whereNames []string
	var err error
	sqlparser.Walk(func(node sqlparser.SQLNode) (bool, error) {
		switch node := node.(type) {
		case *sqlparser.ColName:
			whereNames = append(whereNames, node.Name.String())
		}
		return true, err
	}, statement)
	return whereNames, err
}

func ExtractShowColNames(statement *sqlparser.ShowTablesOpt) ([]string, error) {
	var whereNames []string
	var err error
	if statement == nil || statement.Filter == nil {
		return whereNames, err
	}
	sqlparser.Walk(func(node sqlparser.SQLNode) (bool, error) {
		switch node := node.(type) {
		case *sqlparser.ColName:
			whereNames = append(whereNames, node.Name.String())
		}
		return true, err
	}, statement.Filter)
	return whereNames, err
}

func ExtractShowColUsage(statement *sqlparser.ShowTablesOpt) ([]ColumnUsageMetadata, error) {
	var colUsageSlice []ColumnUsageMetadata
	var err error
	if statement == nil || statement.Filter == nil {
		return colUsageSlice, err
	}
	return GetColumnUsageTypes(statement.Filter.Filter)
}

func ExtractSleepDuration(statement *sqlparser.Sleep) (int, error) {
	var retVal int
	if statement == nil || statement.Duration == nil {
		return retVal, fmt.Errorf("no sleep duration provided")
	}
	switch statement.Duration.Type {
	case sqlparser.IntVal:
		return strconv.Atoi(string(statement.Duration.Val))
	}
	return retVal, fmt.Errorf("sleep definition inadequate")
}

type ColumnUsageMetadata struct {
	ColName *sqlparser.ColName
	ColVal  *sqlparser.SQLVal
}

func CheckColUsagesAgainstTable(colUsages []ColumnUsageMetadata, table *openapistackql.OperationStore) error {
	for _, colUsage := range colUsages {
		param, ok := table.GetParameter(colUsage.ColName.Name.GetRawVal())
		if ok {
			usageErr := CheckSqlParserTypeVsColumn(colUsage, param.ConditionIsValid)
			if usageErr != nil {
				return usageErr
			}
		}
		log.Debugln(fmt.Sprintf("colname = %v", colUsage.ColName))
	}
	return nil
}

func GetColumnUsageTypes(statement sqlparser.Expr) ([]ColumnUsageMetadata, error) {
	var colMetaSlice []ColumnUsageMetadata
	var err error
	sqlparser.Walk(func(node sqlparser.SQLNode) (bool, error) {
		switch node := node.(type) {
		case *sqlparser.ComparisonExpr:
			colMeta := ColumnUsageMetadata{}
			switch lhs := node.Left.(type) {
			case *sqlparser.ColName:
				colMeta.ColName = lhs
			}
			switch rhs := node.Right.(type) {
			case *sqlparser.SQLVal:
				colMeta.ColVal = rhs
			}
			if colMeta.ColName != nil && colMeta.ColVal != nil {
				colMetaSlice = append(colMetaSlice, colMeta)
			}
		}
		return true, nil
	}, statement)
	return colMetaSlice, err
}

func GetColumnUsageTypesForExec(exec *sqlparser.Exec) ([]ColumnUsageMetadata, error) {
	var colMetaSlice []ColumnUsageMetadata
	for _, execVarDef := range exec.ExecVarDefs {
		colMeta := ColumnUsageMetadata{}
		colMeta.ColName = &sqlparser.ColName{Name: execVarDef.ColIdent}
		switch rhs := execVarDef.Val.(type) {
		case *sqlparser.SQLVal:
			colMeta.ColVal = rhs
		default:
			return nil, fmt.Errorf("EXEC param not supplied as valid SQLVal")
		}
		colMetaSlice = append(colMetaSlice, colMeta)
	}
	return colMetaSlice, nil
}

func InferColNameFromExpr(node *sqlparser.AliasedExpr) ColumnHandle {
	return inferColNameFromExpr(node)
}

func inferColNameFromExpr(node *sqlparser.AliasedExpr) ColumnHandle {
	alias := node.As.GetRawVal()
	retVal := ColumnHandle{
		Alias: alias,
		Expr:  node.Expr,
	}
	switch expr := node.Expr.(type) {
	case *sqlparser.ColName:
		retVal.Name = expr.Name.String()
		retVal.DecoratedColumn = sqlparser.String(expr)
		retVal.IsColumn = true
	case *sqlparser.FuncExpr:
		// As a shortcut, functions are integral types
		funcNameLowered := expr.Name.Lowered()
		retVal.Name = sqlparser.String(expr)
		if len(funcNameLowered) >= 4 && funcNameLowered[0:4] == "json" {
			retVal.DecoratedColumn = strings.ReplaceAll(retVal.Name, `\"`, `"`)
			return retVal
		}
		if len(expr.Exprs) == 1 {
			switch ex := expr.Exprs[0].(type) {
			case *sqlparser.AliasedExpr:
				rv := inferColNameFromExpr(ex)
				rv.DecoratedColumn = sqlparser.String(expr)
				rv.Alias = alias
				return rv
			}
		} else {
			var exprsDecorated []string
			for _, exp := range expr.Exprs {
				switch ex := exp.(type) {
				case *sqlparser.AliasedExpr:
					rv := inferColNameFromExpr(ex)
					exprsDecorated = append(exprsDecorated, rv.DecoratedColumn)
				}
			}
			retVal.DecoratedColumn = fmt.Sprintf("%s(%s)", funcNameLowered, strings.Join(exprsDecorated, ", "))
			return retVal
		}
		switch funcNameLowered {
		case "substr":
			switch ex := expr.Exprs[0].(type) {
			case *sqlparser.AliasedExpr:
				rv := inferColNameFromExpr(ex)
				rv.DecoratedColumn = sqlparser.String(expr)
				rv.Alias = alias
				return rv
			}
		default:
			retVal.DecoratedColumn = sqlparser.String(expr)
		}
	case *sqlparser.ConvertExpr:
		switch ex := expr.Expr.(type) {
		case *sqlparser.ColName:
			rv := ColumnHandle{
				Alias: "",
				Expr:  ex,
			}
			rv.DecoratedColumn = fmt.Sprintf("CAST(%s AS %s)", sqlparser.String(ex), sqlparser.String(expr.Type))
			rv.Alias = alias
			return rv
		}
	case *sqlparser.SQLVal:
		// As a shortcut, functions are integral types
		retVal.Name = sqlparser.String(expr)
		retVal.Type = expr.Type
		retVal.Val = expr
		retVal.DecoratedColumn = ExtractStringRepresentationOfValueColumn(expr)
	default:
		retVal.DecoratedColumn = sqlparser.String(expr)
	}
	retVal.DecoratedColumn = strings.ReplaceAll(retVal.DecoratedColumn, `\"`, `"`)
	return retVal
}

func CheckSqlParserTypeVsServiceColumn(colUsage ColumnUsageMetadata) error {
	return CheckSqlParserTypeVsColumn(colUsage, openapistackql.ServiceConditionIsValid)
}

func CheckSqlParserTypeVsResourceColumn(colUsage ColumnUsageMetadata) error {
	return CheckSqlParserTypeVsColumn(colUsage, openapistackql.ResourceConditionIsValid)
}

func CheckSqlParserTypeVsColumn(colUsage ColumnUsageMetadata, verifyCallback func(string, interface{}) bool) error {
	switch colUsage.ColVal.Type {
	case sqlparser.StrVal:
		if !verifyCallback(colUsage.ColName.Name.String(), "") {
			return fmt.Errorf("SHOW key = '%s' does NOT match SQL type '%s'", colUsage.ColName.Name.String(), "StrVal")
		}
	case sqlparser.IntVal:
		if !verifyCallback(colUsage.ColName.Name.String(), 11) {
			return fmt.Errorf("SHOW key = '%s' does NOT match SQL type '%s'", colUsage.ColName.Name.String(), "IntVal")
		}
	case sqlparser.FloatVal:
		if !verifyCallback(colUsage.ColName.Name.String(), 3.33) {
			return fmt.Errorf("SHOW key = '%s' does NOT match SQL type '%s'", colUsage.ColName.Name.String(), "FloatVal")
		}
	case sqlparser.HexNum:
		if !verifyCallback(colUsage.ColName.Name.String(), 0x11) {
			return fmt.Errorf("SHOW key = '%s' does NOT match SQL type '%s'", colUsage.ColName.Name.String(), "HexNum")
		}
	case sqlparser.HexVal:
		return fmt.Errorf("SHOW key = '%s' does NOT match SQL type '%s'", colUsage.ColName.Name.String(), "HexVal")
	case sqlparser.ValArg:
		return fmt.Errorf("SHOW key = '%s' does NOT match SQL type '%s'", colUsage.ColName.Name.String(), "ValArg")
	case sqlparser.BitVal:
		return fmt.Errorf("SHOW key = '%s' does NOT match SQL type '%s'", colUsage.ColName.Name.String(), "BitVal")
	}
	return nil
}

func ExtractTableNameFromTableExpr(tableExpr sqlparser.TableExpr) (*sqlparser.TableName, error) {
	switch table := tableExpr.(type) {
	case *sqlparser.AliasedTableExpr:
		switch tableExpr := table.Expr.(type) {
		case sqlparser.TableName:
			return &tableExpr, nil
		default:
			return nil, fmt.Errorf("could not extract table name from AliasedTableExpr of type %T", tableExpr)
		}
	default:
		return nil, fmt.Errorf("could not extract table name from TableExpr of type %T", table)
	}
	return nil, fmt.Errorf("could not extract table name from TableExpr")
}

func ExtractSingleTableFromTableExprs(tableExprs sqlparser.TableExprs) (*sqlparser.TableName, error) {
	for _, t := range tableExprs {
		log.Infoln(fmt.Sprintf("t = %v", t))
		return ExtractTableNameFromTableExpr(t)
	}
	return nil, fmt.Errorf("could not extract table name from TableExprs")
}

func TableFromSelectNode(sel *sqlparser.Select) (sqlparser.TableName, error) {
	if len(sel.From) != 1 {
		return sqlparser.TableName{}, fmt.Errorf("table expression is complex")
	}
	aliased, ok := sel.From[0].(*sqlparser.AliasedTableExpr)
	if !ok {
		return sqlparser.TableName{}, fmt.Errorf("table expression is complex")
	}
	tableName, ok := aliased.Expr.(sqlparser.TableName)
	if !ok {
		return sqlparser.TableName{}, fmt.Errorf("table expression is complex")
	}
	return tableName, nil
}

type TableExprMap map[sqlparser.TableName]sqlparser.TableExpr

type TableAliasMap map[string]sqlparser.TableExpr

func (tem TableExprMap) GetByAlias(alias string) (sqlparser.TableExpr, bool) {
	for k, v := range tem {
		if k.GetRawVal() == alias {
			return v, true
		}
	}
	return nil, false
}

type ParameterMetadata struct {
	Parent *sqlparser.ComparisonExpr
	Val    interface{}
}

type ParameterMap map[*sqlparser.ColName]ParameterMetadata

type ColTableMap map[*sqlparser.ColName]sqlparser.TableExpr

func (tm ParameterMap) ToStringMap() map[string]interface{} {
	rv := make(map[string]interface{})
	for k, v := range tm {
		rv[k.GetRawVal()] = v
	}
	return rv
}

func (tm TableExprMap) SingleTableMap(filterTable sqlparser.TableName) TableExprMap {
	rv := make(TableExprMap)
	for k, v := range tm {
		if k == filterTable {
			rv[k] = v
		}
	}
	return rv
}

func (tm TableExprMap) ToStringMap() map[string]interface{} {
	rv := make(map[string]interface{})
	for k, v := range tm {
		rv[k.GetRawVal()] = v
	}
	return rv
}

type ParameterRouter struct {
	tablesAliasMap    TableAliasMap
	tableMap          TableExprMap
	paramMap          ParameterMap
	colRefs           ColTableMap
	invalidatedParams map[string]interface{}
}

func NewParameterRouter(tablesAliasMap TableAliasMap, tableMap TableExprMap, paramMap ParameterMap, colRefs ColTableMap) *ParameterRouter {
	return &ParameterRouter{
		tablesAliasMap:    tablesAliasMap,
		tableMap:          tableMap,
		paramMap:          paramMap,
		colRefs:           colRefs,
		invalidatedParams: make(map[string]interface{}),
	}
}

type TableParameterCoupling struct {
	paramMap    ParameterMap
	colMappings map[string]*sqlparser.ColName
}

func NewTableParameterCoupling() *TableParameterCoupling {
	return &TableParameterCoupling{
		paramMap:    make(ParameterMap),
		colMappings: make(map[string]*sqlparser.ColName),
	}
}

func (tpc *TableParameterCoupling) Add(col *sqlparser.ColName, val ParameterMetadata) error {
	tpc.paramMap[col] = val
	_, ok := tpc.colMappings[col.Name.GetRawVal()]
	if ok {
		return fmt.Errorf("parameter '%s' already present", col.Name.GetRawVal())
	}
	tpc.colMappings[col.Name.GetRawVal()] = col
	return nil
}

func (tpc *TableParameterCoupling) GetStringified() map[string]interface{} {
	rv := make(map[string]interface{})
	for k, v := range tpc.paramMap {
		rv[k.Name.GetRawVal()] = v
	}
	return rv
}

func (tpc *TableParameterCoupling) AbbreviateMap(verboseMap map[string]interface{}) (map[string]interface{}, error) {
	rv := make(map[string]interface{})
	for k, v := range tpc.paramMap {
		_, ok := verboseMap[k.GetRawVal()]
		if !ok || v.Val == nil {
			continue
		}
		rv[k.Name.GetRawVal()] = v
	}
	return rv, nil
}

func (tpc *TableParameterCoupling) ReconstituteConsumedParams(returnedMap map[string]interface{}) (map[string]interface{}, error) {
	rv := make(map[string]interface{})
	for k, v := range tpc.paramMap {
		rv[k.GetRawVal()] = v
	}
	for k, v := range returnedMap {
		key, ok := tpc.colMappings[k]
		if !ok || v == nil {
			return nil, fmt.Errorf("no reconstitution mapping for key = '%s'", k)
		}
		key.Metadata = true
		keyToDelete := key.GetRawVal()
		_, ok = rv[keyToDelete]
		if !ok {
			return nil, fmt.Errorf("cannot process consumed params: attempt to delete non existing key")
		}
		delete(rv, keyToDelete)
	}
	return rv, nil
}

func (pr *ParameterRouter) GetAvailableParameters(tb sqlparser.TableExpr) *TableParameterCoupling {
	rv := NewTableParameterCoupling()
	for k, v := range pr.paramMap {
		key := k.GetRawVal()
		tableAlias := k.Qualifier.GetRawVal()
		foundTable, ok := pr.tablesAliasMap[tableAlias]
		if ok && foundTable != tb {
			continue
		}
		if pr.isInvalidated(key) {
			continue
		}
		ref, ok := pr.colRefs[k]
		if ok && ref != tb {
			continue
		}
		rv.Add(k, v)
	}
	return rv
}

func (pr *ParameterRouter) InvalidateParams(params map[string]interface{}) error {
	for k, v := range params {
		err := pr.invalidate(k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (pr *ParameterRouter) isInvalidated(key string) bool {
	_, ok := pr.invalidatedParams[key]
	return ok
}

func (pr *ParameterRouter) invalidate(key string, val interface{}) error {
	if pr.isInvalidated(key) {
		return fmt.Errorf("parameter '%s' already invalidated", key)
	}
	pr.invalidatedParams[key] = val
	return nil
}

func (pr *ParameterRouter) Route(tb sqlparser.TableExpr) error {
	for k, _ := range pr.paramMap {
		alias := k.Qualifier.GetRawVal()
		if alias == "" {
			continue
		}
		t, ok := pr.tablesAliasMap[alias]
		if !ok {
			return fmt.Errorf("alias '%s' does not map to any table expression", alias)
		}
		if t == tb {
			ref, ok := pr.colRefs[k]
			if ok && ref != t {
				return fmt.Errorf("failed parameter routing, cannot re-assign")
			}
			pr.colRefs[k] = t
		}
	}
	return nil
}
