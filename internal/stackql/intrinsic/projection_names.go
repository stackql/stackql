package intrinsic

import (
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/parserutil"
)

// projectionName retains the host's legacy labels without changing the SDK
// expression or the values transferred between exchanges.
func projectionName(node *sqlparser.AliasedExpr) (string, error) {
	if name := node.As.GetRawVal(); name != "" {
		return name, nil
	}
	if col, ok := node.Expr.(*sqlparser.ColName); ok {
		return col.Name.GetRawVal(), nil
	}
	if _, ok := node.Expr.(*sqlparser.OrExpr); ok {
		return "", nil
	}
	if binary, ok := node.Expr.(*sqlparser.BinaryExpr); ok && binary.Operator == "||" {
		return "", nil
	}
	function, isFunction := node.Expr.(*sqlparser.FuncExpr)
	if isFunction && len(function.Exprs) > 1 && !function.IsAggregate() && function.Over == nil {
		handles, err := parserutil.ExtractSelectColumnNames(&sqlparser.Select{SelectExprs: sqlparser.SelectExprs{node}}, nil)
		if err != nil {
			return "", err
		}
		if len(handles) == 1 && handles[0].Name != "" {
			return handles[0].Name, nil
		}
	}
	return sqlparser.String(node.Expr), nil
}
