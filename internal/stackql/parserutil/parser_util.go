package parserutil

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/stackql/internal/stackql/astformat"
	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/logging"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

const (
	FloatBitSize int = 64
)

//nolint:gocritic,staticcheck // TODO: clean up this and other tech debt
func ExtractLHSTable(sqlFunc *sqlparser.FuncExpr) (*sqlparser.ColName, bool) {
	if sqlFunc == nil {
		return nil, false
	}
	funcNameLowered := strings.ToLower(sqlFunc.Name.GetRawVal())
	switch funcNameLowered {
	default:
		switch ex := sqlFunc.Exprs[0].(type) {
		case *sqlparser.AliasedExpr:
			switch ex2 := ex.Expr.(type) {
			case *sqlparser.ColName:
				return ex2, true
			}
		}
	}
	return nil, false
}

// These null "dual" tables are some vitess artifact.
func IsNullTable(node sqlparser.TableExpr) bool {
	return isNullTable(node)
}

//nolint:gocritic // TODO: review
func isNullTable(node sqlparser.TableExpr) bool {
	switch node := node.(type) {
	case *sqlparser.AliasedTableExpr:
		switch expr := node.Expr.(type) {
		case sqlparser.TableName:
			if expr.Name.GetRawVal() == "dual" {
				return true
			}
		}
	}
	return false
}

//nolint:gocritic // TODO: review
func GetTableNameFromTableExpr(node sqlparser.TableExpr) (sqlparser.TableName, error) {
	switch tt := node.(type) {
	case *sqlparser.AliasedTableExpr:
		tn, ok := tt.Expr.(sqlparser.TableName)
		if ok {
			return tn, nil
		}
	}
	return sqlparser.TableName{}, fmt.Errorf("table expression too complex")
}

func NewUnaliasedColumnHandle(name string) ColumnHandle {
	return ColumnHandle{Name: name}
}

func ExtractSelectColumnNames(selStmt *sqlparser.Select, formatter sqlparser.NodeFormatter) ([]ColumnHandle, error) {
	var colNames []ColumnHandle
	var err error
	for _, node := range selStmt.SelectExprs {
		switch node := node.(type) {
		case *sqlparser.AliasedExpr:
			cn, cErr := inferColNameFromExpr(node, formatter)
			if cErr != nil {
				return nil, cErr
			}
			colNames = append(colNames, cn)
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

//nolint:gocritic,exhaustive // TODO: review
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
	//nolint:exhaustive // TODO: review
	switch expr.Type {
	case sqlparser.StrVal:
		return fmt.Sprintf(`'%s'`, string(expr.Val))
	case sqlparser.IntVal, sqlparser.FloatVal:
		return string(expr.Val)
	default:
		return string(expr.Val)
	}
}

//nolint:gocritic,exhaustive,govet // TODO: review
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
						return nil, nonValCount, fmt.Errorf("error extracting Values integer: %w", err)
					}
					row[innerIdx] = rv
				case sqlparser.FloatVal:
					rv, err := strconv.ParseFloat(string(expr.Val), FloatBitSize)
					if err != nil {
						return nil, nonValCount, fmt.Errorf("error extracting Values float: %w", err)
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

func isNullFromClause(from sqlparser.TableExprs) bool {
	for _, tb := range from {
		if !isNullTable(tb) {
			return false
		}
	}
	return true
}

func ExtractSelectValColumns(selStmt *sqlparser.Select) (map[int]map[string]interface{}, int) {
	cols := make(map[int]map[string]interface{})
	var nonValCount int
	fromIsNull := isNullFromClause(selStmt.From)
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
				if !fromIsNull {
					nonValCount++
				} else {
					alias := node.As.GetRawVal()
					cols[idx] = map[string]interface{}{
						alias: expr,
					}
				}
			case *sqlparser.ColName:
				nonValCount++
			case sqlparser.BoolVal:
				cols[idx] = map[string]interface{}{fmt.Sprintf("$$unaliased_col_%d", idx): expr}
			default:
				logging.GetLogger().Infoln(fmt.Sprintf("cannot use AliasedExpr of type '%T' as a raw value", expr))
				cols[idx] = nil
				nonValCount++
			}
		default:
			logging.GetLogger().Infoln(fmt.Sprintf("cannot use SelectExpr of type '%T' as a raw value", node))
			cols[idx] = nil
			nonValCount++
		}
	}
	return cols, nonValCount
}

func ExtractInsertValColumns(insStmt *sqlparser.Insert) (map[int]map[int]interface{}, int, error) {
	return extractInsertValColumns(insStmt, false)
}

func ExtractUpdateValColumns(
	upStmt *sqlparser.Update,
) (map[*sqlparser.ColName]interface{}, []*sqlparser.ColName, error) {
	return extractUpdateValColumns(upStmt, false)
}

func ExtractInsertValColumnsPlusPlaceHolders(insStmt *sqlparser.Insert) (map[int]map[int]interface{}, int, error) {
	return extractInsertValColumns(insStmt, false)
}

func extractInsertValColumns(
	insStmt *sqlparser.Insert,
	includePlaceholders bool,
) (map[int]map[int]interface{}, int, error) {
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
			} else { //nolint:gocritic // TODO: review
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

//nolint:gocognit,gocritic // not overly complex
func extractUpdateValColumns(
	updateStmt *sqlparser.Update,
	includePlaceholders bool, //nolint:unparam,revive // TODO: review
) (map[*sqlparser.ColName]interface{}, []*sqlparser.ColName, error) {
	var nonValCols []*sqlparser.ColName
	retVal := make(map[*sqlparser.ColName]interface{})
	for _, ex := range updateStmt.Exprs {
		switch node := ex.Expr.(type) {
		case *sqlparser.Subquery:
			logging.GetLogger().Infof("subquery provided for update: '%v'", node)
			return nil, nil, fmt.Errorf("subquery in update statement not yet supported")
		case *sqlparser.SQLVal:
			retVal[ex.Name] = string(node.Val)
		case *sqlparser.FuncExpr:
			if strings.ToLower(node.Name.GetRawVal()) == "string" {
				_, err := GetStringFromStringFunc(node)
				if err != nil {
					return nil, nil, fmt.Errorf("could not extract string from func string()")
				}
				retVal[ex.Name] = node
			} else if strings.ToLower(node.Name.GetRawVal()) == "json" {
				retVal[ex.Name] = node
			} else {
				return nil, nil, fmt.Errorf("could not extract string from func string()")
			}
		default:
			return nil, nil, fmt.Errorf("update statement RHS of type '%T' not yet supported", node)
		}
	}
	var err error
	err = sqlparser.Walk(func(node sqlparser.SQLNode) (bool, error) {
		switch node := node.(type) {
		case *sqlparser.ComparisonExpr:
			if node.Operator == sqlparser.EqualStr {
				switch l := node.Left.(type) {
				case *sqlparser.ColName:
					// key := l.Name.GetRawVal()
					// lookupMap[key] = l
					// columnOrder = append(columnOrder, key)
					switch r := node.Right.(type) {
					case *sqlparser.SQLVal:
						retVal[l] = string(r.Val)
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
	}, updateStmt.Where)
	if err != nil {
		return nil, nonValCols, err
	}
	return retVal, nonValCols, nil
}

//nolint:gocritic // TODO: review
func ExtractWhereColNames(statement *sqlparser.Where) ([]string, error) {
	var whereNames []string
	var err error
	sqlparser.Walk(func(node sqlparser.SQLNode) (bool, error) { //nolint:errcheck // TODO: review
		switch node := node.(type) {
		case *sqlparser.ColName:
			whereNames = append(whereNames, node.Name.String())
		}
		return true, err
	}, statement)
	return whereNames, err
}

//nolint:gocritic // TODO: review
func ExtractShowColNames(statement *sqlparser.ShowTablesOpt) ([]string, error) {
	var whereNames []string
	var err error
	if statement == nil || statement.Filter == nil {
		return whereNames, err
	}
	sqlparser.Walk(func(node sqlparser.SQLNode) (bool, error) { //nolint:errcheck // TODO: review
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

//nolint:gocritic,exhaustive // TODO: review
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

func CheckColUsagesAgainstTable(colUsages []ColumnUsageMetadata, table anysdk.OperationStore) error {
	for _, colUsage := range colUsages {
		param, ok := table.GetParameter(colUsage.ColName.Name.GetRawVal())
		if ok {
			usageErr := CheckSQLParserTypeVsColumn(colUsage, param.ConditionIsValid)
			if usageErr != nil {
				return usageErr
			}
		}
		logging.GetLogger().Debugln(fmt.Sprintf("colname = %v", colUsage.ColName))
	}
	return nil
}

//nolint:gocritic // TODO: review
func GetColumnUsageTypes(statement sqlparser.Expr) ([]ColumnUsageMetadata, error) {
	var colMetaSlice []ColumnUsageMetadata
	var err error
	sqlparser.Walk(func(node sqlparser.SQLNode) (bool, error) { //nolint:errcheck // TODO: review
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

func InferColNameFromExpr(node *sqlparser.AliasedExpr, formatter sqlparser.NodeFormatter) (ColumnHandle, error) {
	return inferColNameFromExpr(node, formatter)
}

func GetStringFromStringFunc(fe *sqlparser.FuncExpr) (string, error) {
	if strings.ToLower(fe.Name.GetRawVal()) == "string" && len(fe.Exprs) == 1 {
		//nolint:gocritic // acceptable
		switch et := fe.Exprs[0].(type) {
		case *sqlparser.AliasedExpr:
			switch et2 := et.Expr.(type) {
			case *sqlparser.SQLVal:
				return string(et2.Val), nil
			}
		}
	}
	return "", fmt.Errorf("cannot extract string from func '%s'", fe.Name)
}

type aggregatedCol interface {
	getReturnType() sqlparser.ValType
	getName() string
}

type simpleAggSQLCol struct {
	name       string
	returnType sqlparser.ValType
}

func (s simpleAggSQLCol) getReturnType() sqlparser.ValType {
	return s.returnType
}

func (s simpleAggSQLCol) getName() string {
	return s.name
}

func inferAggregatedCol(funcNameLowered string) (aggregatedCol, bool) {
	switch funcNameLowered {
	case "count":
		return simpleAggSQLCol{
			name:       funcNameLowered,
			returnType: sqlparser.IntVal,
		}, true
	case "sum":
		return simpleAggSQLCol{
			name:       funcNameLowered,
			returnType: sqlparser.IntVal,
		}, true
	default:
		return nil, false
	}
}

//nolint:funlen,gocognit,gocritic // not overly complex
func inferColNameFromExpr(
	node *sqlparser.AliasedExpr,
	formatter sqlparser.NodeFormatter,
) (ColumnHandle, error) {
	alias := node.As.GetRawVal()
	retVal := ColumnHandle{
		Alias: alias,
		Expr:  node.Expr,
	}
	switch expr := node.Expr.(type) {
	case *sqlparser.ColName:
		retVal.Name = expr.Name.String()
		retVal.Qualifier = expr.Qualifier.GetRawVal()
		decoratedCol := astformat.String(expr, formatter)
		// if decoratedCol != retVal.Name {
		retVal.Alias = alias
		retVal.DecoratedColumn = getDecoratedColRendition(decoratedCol, alias)
		//}
		retVal.IsColumn = true
	case *sqlparser.GroupConcatExpr:
		if len(expr.Exprs) != 1 {
			return retVal, fmt.Errorf("group_concat() arg count = %d is NOT permissable", len(expr.Exprs))
		}
		switch ex := expr.Exprs[0].(type) {
		case *sqlparser.AliasedExpr:
			rv, err := inferColNameFromExpr(ex, formatter)
			if err != nil {
				return rv, err
			}
			rv.DecoratedColumn = astformat.String(expr, formatter)
			rv.Alias = alias
			return rv, nil
		}

	case *sqlparser.FuncExpr:
		// As a shortcut, functions are integral types
		funcNameLowered := expr.Name.Lowered()
		unaliasedRendition := astformat.String(expr, formatter)
		retVal.Name = unaliasedRendition
		aggCol, isAggCol := inferAggregatedCol(funcNameLowered)
		if isAggCol {
			retVal.IsAggregateExpr = true
			retVal.Type = aggCol.getReturnType()
		}
		if len(funcNameLowered) >= 4 && funcNameLowered[0:4] == "json" {
			decoratedColumn := strings.ReplaceAll(retVal.Name, `\"`, `"`)
			retVal.DecoratedColumn = getDecoratedColRendition(decoratedColumn, alias)
			if len(funcNameLowered) == 4 { //nolint:mnd // TODO: remove this
				return retVal, nil
			}
		}
		if len(expr.Exprs) == 1 { //nolint:nestif // TODO: review
			switch ex := expr.Exprs[0].(type) {
			case *sqlparser.AliasedExpr:
				rv, err := inferColNameFromExpr(ex, formatter)
				if err != nil {
					return rv, err
				}
				decoratedColumn := astformat.String(expr, formatter)
				rv.DecoratedColumn = getDecoratedColRendition(decoratedColumn, alias)
				rv.Alias = alias
				rv.IsAggregateExpr = retVal.IsAggregateExpr
				return rv, nil
			}
		} else {
			var exprsDecorated []string
			for i, exp := range expr.Exprs {
				switch ex := exp.(type) {
				case *sqlparser.AliasedExpr:
					rv, err := inferColNameFromExpr(ex, formatter)
					if err != nil {
						return rv, err
					}
					if i == 0 {
						retVal.Name = rv.Name
						if funcNameLowered == constants.SQLFuncJSONExtractPostgres {
							rv.DecoratedColumn = fmt.Sprintf(`%s%s`, rv.DecoratedColumn, constants.PostgresJSONCastSuffix)
						}
					}
					exprsDecorated = append(exprsDecorated, rv.DecoratedColumn)
				}
			}
			decoratedColumn := fmt.Sprintf("%s(%s)", funcNameLowered, strings.Join(exprsDecorated, ", "))
			if retVal.Name != constants.SQLFuncJSONExtractPostgres {
				retVal.DecoratedColumn = getDecoratedColRendition(decoratedColumn, alias)
			}
			return retVal, nil
		}
		switch funcNameLowered {
		case "substr":
			switch ex := expr.Exprs[0].(type) {
			case *sqlparser.AliasedExpr:
				rv, err := inferColNameFromExpr(ex, formatter)
				if err != nil {
					return rv, err
				}
				rv.Alias = alias
				return rv, nil
			}
		default:
			retVal.DecoratedColumn = astformat.String(expr, formatter)
		}
	case *sqlparser.ConvertExpr:
		switch ex := expr.Expr.(type) {
		case *sqlparser.ColName:
			rv := ColumnHandle{
				Alias: "",
				Expr:  ex,
			}
			decoratedColumn := fmt.Sprintf(
				"CAST(%s AS %s)",
				astformat.String(ex, formatter), astformat.String(expr.Type, formatter))
			rv.DecoratedColumn = getDecoratedColRendition(decoratedColumn, alias)
			rv.Alias = alias
			return rv, nil
		}
	case *sqlparser.SQLVal:
		// As a shortcut, functions are integral types
		retVal.Name = alias
		retVal.Type = expr.Type
		retVal.Val = expr
		decoratedColumn := ExtractStringRepresentationOfValueColumn(expr)
		retVal.DecoratedColumn = getDecoratedColRendition(decoratedColumn, alias)

	default:
		decoratedColumn := astformat.String(expr, formatter)
		retVal.DecoratedColumn = getDecoratedColRendition(decoratedColumn, alias)
	}
	retVal.DecoratedColumn = strings.ReplaceAll(retVal.DecoratedColumn, `\"`, `"`)
	return retVal, nil
}

func getDecoratedColRendition(baseDecoratedColumn, alias string) string {
	if alias != "" {
		return fmt.Sprintf(`%s AS "%s"`, baseDecoratedColumn, alias)
	}
	return baseDecoratedColumn
}

func CheckSQLParserTypeVsServiceColumn(
	colUsage ColumnUsageMetadata) error {
	return CheckSQLParserTypeVsColumn(colUsage, anysdk.ServiceConditionIsValid)
}

func CheckSQLParserTypeVsResourceColumn(
	colUsage ColumnUsageMetadata) error {
	return CheckSQLParserTypeVsColumn(colUsage, anysdk.ResourceConditionIsValid)
}

//nolint:mnd // TODO: remove this
func CheckSQLParserTypeVsColumn(colUsage ColumnUsageMetadata, verifyCallback func(string, interface{}) bool) error {
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
}

func ExtractSingleTableFromTableExprs(tableExprs sqlparser.TableExprs) (*sqlparser.TableName, error) {
	for _, t := range tableExprs {
		logging.GetLogger().Infoln(fmt.Sprintf("t = %v", t))
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

func IsFromExprSimple(from sqlparser.TableExprs) bool {
	for i, node := range from {
		if i == 0 {
			switch node.(type) {
			case *sqlparser.JoinTableExpr, *sqlparser.AliasedTableExpr:
				continue
			default:
				return false
			}
		} else {
			switch node.(type) {
			case *sqlparser.TableValuedFuncTableExpr:
				continue
			default:
				return false
			}
		}
	}
	return true
}

func IsCreateMaterializedView(stmt sqlparser.Statement) bool {
	switch st := stmt.(type) {
	case *sqlparser.DDL:
		return isCreateMaterializedView(st)
	default:
		return false
	}
}

func IsDropMaterializedView(stmt sqlparser.Statement) bool {
	switch st := stmt.(type) {
	case *sqlparser.DDL:
		return isDropMaterializedView(st)
	default:
		return false
	}
}

func IsDropPhysicalTable(stmt sqlparser.Statement) bool {
	switch st := stmt.(type) {
	case *sqlparser.DDL:
		return isDropPhysicalTable(st)
	default:
		return false
	}
}

func isCreateMaterializedView(ddl *sqlparser.DDL) bool {
	switch ddl.Action {
	case sqlparser.CreateStr:
		switch strings.ToLower(ddl.Modifier) {
		case "materialized":
			return true
		default:
			return false
		}
	default:
		return false
	}
}

func isDropMaterializedView(ddl *sqlparser.DDL) bool {
	switch ddl.Action {
	case sqlparser.DropStr:
		switch strings.ToLower(ddl.Modifier) {
		case "materialized":
			return true
		default:
			return false
		}
	default:
		return false
	}
}

func isDropPhysicalTable(ddl *sqlparser.DDL) bool {
	switch ddl.Action {
	case sqlparser.DropStr:
		switch strings.ToLower(ddl.Modifier) {
		case "table":
			return true
		default:
			return false
		}
	default:
		return false
	}
}

func IsCreatePhysicalTable(stmt sqlparser.Statement) bool {
	switch st := stmt.(type) {
	case *sqlparser.DDL:
		return isCreatePhysicalTable(st)
	default:
		return false
	}
}

func IsCreateTemporaryPhysicalTable(stmt sqlparser.Statement) bool {
	switch st := stmt.(type) {
	case *sqlparser.DDL:
		return isCreatePhysicalTable(st) && isCreateTemp(st)
	default:
		return false
	}
}

func isCreateTemp(ddl *sqlparser.DDL) bool {
	switch ddl.Action {
	case sqlparser.CreateStr:
		switch strings.ToLower(ddl.Modifier) {
		case "temp", "temporary":
			return true
		default:
			return false
		}
	default:
		return false
	}
}

func isCreatePhysicalTable(ddl *sqlparser.DDL) bool {
	// if ddl.OptLike == nil && ddl.TableSpec == nil {
	// 	return false
	// }
	switch ddl.Action {
	case sqlparser.CreateStr:
		return ddl.OptLike != nil || ddl.TableSpec != nil
	default:
		return false
	}
}

func RenderDDLTableSpecStmt(ddl *sqlparser.DDL) (string, error) {
	return renderDDLTableSpecStmt(ddl)
}

func renderDDLTableSpecStmt(ddl *sqlparser.DDL) (string, error) {
	if ddl == nil || ddl.TableSpec == nil {
		return "", fmt.Errorf("cannot render DDL table spec for ddl = '%v'", ddl)
	}
	return strings.ReplaceAll(
			astformat.String(ddl.TableSpec, astformat.DefaultSelectExprsFormatter), `"`, ""),
		nil
}

func RenderDDLSelectStmt(ddl *sqlparser.DDL) string {
	return renderDDLSelectStmt(ddl)
}

func renderDDLSelectStmt(ddl *sqlparser.DDL) string {
	return strings.ReplaceAll(
		astformat.String(ddl.SelectStatement, astformat.DefaultSelectExprsFormatter), `"`, "")
}

func RenderRefreshMaterializedViewSelectStmt(ref *sqlparser.RefreshMaterializedView) string {
	return strings.ReplaceAll(
		astformat.String(ref.ImplicitSelect, astformat.DefaultSelectExprsFormatter), `"`, "")
}

//nolint:gocritic // acceptable
func ExtractSelectStatmentFromDDL(stmt sqlparser.Statement) (sqlparser.SelectStatement, bool) {
	switch st := stmt.(type) {
	case *sqlparser.DDL:
		if st.SelectStatement != nil {
			return st.SelectStatement, true
		}
	}
	return nil, false
}
