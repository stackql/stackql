package parserutil

import (
	"strings"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

func IsInsertRowsDynamic(node *sqlparser.Insert) bool {
	switch rowsNode := node.Rows.(type) {
	case *sqlparser.Select:
		return IsSelectDynamic(rowsNode)
	case sqlparser.Values:
		return false
	default:
		return true
	}
}

func IsSelectDynamic(node *sqlparser.Select) bool {
	if node.Where != nil {
		return true
	}
	return false
}

func isScalarSQLVal(expr *sqlparser.SQLVal) bool {
	//nolint:exhaustive // this is not exhaustive, but it is sufficient for our use cases
	switch expr.Type {
	case sqlparser.IntVal, sqlparser.FloatVal, sqlparser.HexNum, sqlparser.HexVal, sqlparser.ValArg:
		return true
	case sqlparser.StrVal:
		valStr := strings.ToLower(string(expr.Val))
		if valStr == "true" || valStr == "false" || valStr == "null" {
			return true
		}
		return false
	default:
		return false
	}
}

func IsScalarComparison(expr *sqlparser.ComparisonExpr) bool {
	if expr.Operator != sqlparser.EqualStr {
		return false
	}
	lhs, leftIsVal := expr.Left.(*sqlparser.SQLVal)
	rhs, rightIsVal := expr.Right.(*sqlparser.SQLVal)
	if !leftIsVal || !rightIsVal {
		return false
	}
	return isScalarSQLVal(lhs) && isScalarSQLVal(rhs)
}
