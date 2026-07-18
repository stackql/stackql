package earlyanalysis

import (
	"fmt"

	"github.com/stackql/any-sdk/public/sqlengine"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

// Constant folding for WHERE-clause function expressions (issue #686).
// Parameter routing consumes only literal comparison values, so a required
// parameter constrained by a pure-literal function expression (eg
// `strftime('%s', date('now'))`) was dropped and the query failed.  Foldable
// operands are evaluated against the session's SQL backend during early
// analysis and the typed literal is substituted into the AST.  Column refs,
// subqueries, tuples and backend-unevaluable expressions are left untouched.

// foldWhereConstantFuncExprs walks every WHERE clause in the statement and
// folds foldable comparison operands in place.
func foldWhereConstantFuncExprs(statement sqlparser.SQLNode, sqlEngine sqlengine.SQLEngine) {
	if sqlEngine == nil {
		return
	}
	//nolint:errcheck // the visitor never errs; non-foldable nodes are skipped
	sqlparser.Walk(func(node sqlparser.SQLNode) (bool, error) {
		where, isWhere := node.(*sqlparser.Where)
		if !isWhere || where == nil || where.Expr == nil {
			return true, nil
		}
		foldComparisonOperands(where.Expr, sqlEngine)
		return true, nil
	}, statement)
}

// foldComparisonOperands descends AND/OR/NOT conjunctions and folds each
// comparison's foldable operand(s) in place.
func foldComparisonOperands(expr sqlparser.Expr, sqlEngine sqlengine.SQLEngine) {
	switch e := expr.(type) {
	case *sqlparser.AndExpr:
		foldComparisonOperands(e.Left, sqlEngine)
		foldComparisonOperands(e.Right, sqlEngine)
	case *sqlparser.OrExpr:
		foldComparisonOperands(e.Left, sqlEngine)
		foldComparisonOperands(e.Right, sqlEngine)
	case *sqlparser.NotExpr:
		foldComparisonOperands(e.Expr, sqlEngine)
	case *sqlparser.ComparisonExpr:
		if folded, ok := foldScalarExpr(e.Right, sqlEngine); ok {
			e.Right = folded
		}
		if folded, ok := foldScalarExpr(e.Left, sqlEngine); ok {
			e.Left = folded
		}
	}
}

// foldScalarExpr evaluates a pure-literal function expression to a typed
// literal. The bool reports whether folding occurred.
func foldScalarExpr(expr sqlparser.Expr, sqlEngine sqlengine.SQLEngine) (sqlparser.Expr, bool) {
	if _, isFunc := expr.(*sqlparser.FuncExpr); !isFunc {
		return nil, false
	}
	if !isFoldableScalar(expr) {
		return nil, false
	}
	renderedQuery := fmt.Sprintf("SELECT %s", sqlparser.String(expr))
	row := sqlEngine.QueryRow(renderedQuery)
	if row == nil {
		return nil, false
	}
	var result interface{}
	if scanErr := row.Scan(&result); scanErr != nil {
		return nil, false
	}
	return literalFromGoValue(result)
}

// isFoldableScalar reports whether the expression tree is side-effect-free
// and fully literal: no column references, subqueries or tuple values.
func isFoldableScalar(expr sqlparser.Expr) bool {
	foldable := true
	//nolint:errcheck // the visitor never errs
	sqlparser.Walk(func(node sqlparser.SQLNode) (bool, error) {
		switch node.(type) {
		case *sqlparser.ColName, *sqlparser.Subquery, sqlparser.ValTuple:
			foldable = false
			return false, nil
		}
		return true, nil
	}, expr)
	return foldable
}

// literalFromGoValue renders a database/sql scan result as a typed SQL literal.
// A NULL result is not foldable: the pre-fold semantics are preserved instead
// of comparing against an unintended empty literal.
func literalFromGoValue(v interface{}) (sqlparser.Expr, bool) {
	switch typed := v.(type) {
	case int64:
		return sqlparser.NewIntVal([]byte(fmt.Sprintf("%d", typed))), true
	case float64:
		return sqlparser.NewFloatVal([]byte(fmt.Sprintf("%g", typed))), true
	case []byte:
		return sqlparser.NewStrVal(typed), true
	case string:
		return sqlparser.NewStrVal([]byte(typed)), true
	default:
		return nil, false
	}
}
