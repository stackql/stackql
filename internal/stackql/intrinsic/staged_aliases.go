package intrinsic

import (
	"github.com/stackql-labs/omnisdk/pkg/query"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

// Preserve aliases that WHERE references in the SDK source query. The backend
// still evaluates the final projection, while SDK resolution can substitute the
// original expression and retain precedence for actual source columns.
func whereAliasOutputs(node *sqlparser.Select) ([]query.Output, error) {
	if node.Where == nil {
		return nil, nil
	}
	names := map[string]bool{}
	_ = sqlparser.Walk(func(n sqlparser.SQLNode) (bool, error) {
		if col, ok := n.(*sqlparser.ColName); ok && col.Qualifier.Name.IsEmpty() {
			names[col.Name.GetRawVal()] = true
		}
		return true, nil
	}, node.Where.Expr)
	outputs := []query.Output{}
	for _, selection := range node.SelectExprs {
		selected, ok := selection.(*sqlparser.AliasedExpr)
		if !ok {
			continue
		}
		alias := selected.As.GetRawVal()
		if alias == "" || !names[alias] {
			continue
		}
		value, err := expression(selected.Expr)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, query.NewOutput(alias, value))
	}
	return outputs, nil
}
