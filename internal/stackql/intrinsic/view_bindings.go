package intrinsic

import (
	"fmt"
	"strings"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

// viewLiteralBindings collects only unconditional caller inputs. An OR branch
// cannot establish an inbox value, and another alias cannot bind this view.
func viewLiteralBindings(where *sqlparser.Where, alias, resource string) map[string]*sqlparser.ComparisonExpr {
	bindings := map[string]*sqlparser.ComparisonExpr{}
	var visit func(sqlparser.Expr)
	visit = func(expr sqlparser.Expr) {
		if and, ok := expr.(*sqlparser.AndExpr); ok {
			visit(and.Left)
			visit(and.Right)
			return
		}
		compare, ok := expr.(*sqlparser.ComparisonExpr)
		if !ok || !literalViewBinding(compare) {
			return
		}
		col, ok := compare.Left.(*sqlparser.ColName)
		if !ok {
			return
		}
		qualifier := col.Qualifier.Name.GetRawVal()
		if qualifier != "" && qualifier != alias && qualifier != resource {
			return
		}
		bindings[col.Name.GetRawVal()] = compare
	}
	if where != nil {
		visit(where.Expr)
	}
	return bindings
}

func literalViewBinding(compare *sqlparser.ComparisonExpr) bool {
	switch compare.Operator {
	case sqlparser.EqualStr:
		_, literal := compare.Right.(*sqlparser.SQLVal)
		return literal
	case sqlparser.InStr:
		tuple, ok := compare.Right.(sqlparser.ValTuple)
		if !ok || len(tuple) == 0 {
			return false
		}
		for _, value := range tuple {
			if _, literal := value.(*sqlparser.SQLVal); !literal {
				return false
			}
		}
		return true
	}
	return false
}

// matchViewPredicate selects declared variants from explicit inputs. It neither
// interprets provider code nor guesses omitted scope values.
func matchViewPredicate(predicate, dialect string, bindings map[string]*sqlparser.ComparisonExpr) (bool, error) {
	condition := strings.Join(strings.Fields(predicate), "")
	if condition == "" || condition == "true" {
		return true, nil
	}
	matches := true
	for _, term := range strings.Split(condition, "&&") {
		switch {
		case term == "false":
			matches = false
		case term == "true":
		case strings.HasPrefix(term, "sqlDialect=="):
			matches = matches && strings.Trim(strings.TrimPrefix(term, "sqlDialect=="), "\"'") == dialect
		case strings.HasPrefix(term, "requiredParams==[") && strings.HasSuffix(term, "]"):
			names := strings.TrimSuffix(strings.TrimPrefix(term, "requiredParams==["), "]")
			if names == "" {
				continue
			}
			for _, name := range strings.Split(names, ",") {
				matches = matches && bindings[strings.Trim(name, "\"'")] != nil
			}
		default:
			return false, fmt.Errorf("unsupported view predicate %q", predicate)
		}
	}
	return matches, nil
}
