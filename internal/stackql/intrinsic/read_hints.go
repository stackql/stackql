package intrinsic

import (
	"strings"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

// readHints preserves original SQL controls across the staging boundary. These
// are optimisations only; the backend still applies the original SQL, except
// for offsets the SDK explicitly reports as already consumed by the API.
//
//nolint:gocognit,govet // AST traversal keeps the related hint rules together.
func readHints(node *sqlparser.Select) map[string]omnisdk.ReadHints {
	if len(node.From) != 1 {
		return nil
	}
	table, ok := node.From[0].(*sqlparser.AliasedTableExpr)
	if !ok {
		return nil
	}
	name, ok := table.Expr.(sqlparser.TableName)
	if !ok {
		return nil
	}
	alias := table.As.GetRawVal()
	if alias == "" {
		alias = name.Name.GetRawVal()
	}
	hint := omnisdk.ReadHints{DisableRowControls: len(node.GroupBy) > 0 || node.Having != nil || node.Distinct}
	seen := map[string]bool{}
	add := func(column *sqlparser.ColName) {
		if qualifier := column.Qualifier.Name.GetRawVal(); qualifier != "" && qualifier != alias {
			return
		}
		name := column.Name.GetRawVal()
		if !seen[name] {
			hint.Columns = append(hint.Columns, name)
			seen[name] = true
		}
	}
	selectedAliases := map[string]*sqlparser.ColName{}
	for _, expr := range node.SelectExprs {
		if _, star := expr.(*sqlparser.StarExpr); star {
			hint.AllColumns = true
			continue
		}
		if selected, ok := expr.(*sqlparser.AliasedExpr); ok {
			if col, ok := selected.Expr.(*sqlparser.ColName); ok && selected.As.GetRawVal() != "" {
				selectedAliases[selected.As.GetRawVal()] = col
			}
		}
	}
	_ = sqlparser.Walk(func(n sqlparser.SQLNode) (bool, error) {
		if function, ok := n.(*sqlparser.FuncExpr); ok && function.IsAggregate() {
			hint.DisableRowControls = true
			if strings.EqualFold(function.Name.GetRawVal(), "count") && len(function.Exprs) == 1 {
				if _, star := function.Exprs[0].(*sqlparser.StarExpr); star {
					hint.Count = true
				}
			}
		}
		if col, ok := n.(*sqlparser.ColName); ok {
			add(col)
		}
		return true, nil
	}, node.SelectExprs)
	if node.Where != nil {
		_ = sqlparser.Walk(func(n sqlparser.SQLNode) (bool, error) {
			if col, ok := n.(*sqlparser.ColName); ok {
				add(col)
			}
			return true, nil
		}, node.Where.Expr)
	}
	for _, term := range node.OrderBy {
		column, ok := term.Expr.(*sqlparser.ColName)
		if !ok {
			hint.DisableRowControls = true
			continue
		}
		if selected, ok := selectedAliases[column.Name.GetRawVal()]; ok && column.Qualifier.Name.IsEmpty() {
			column = selected
		}
		add(column)
		hint.OrderBy = append(hint.OrderBy, omnisdk.ReadOrder{
			Column: column.Name.GetRawVal(), Descending: strings.EqualFold(term.Direction, "desc"),
		})
	}
	if node.Limit != nil {
		if node.Limit.Rowcount != nil {
			hint.Limit, _ = rowCount(node.Limit.Rowcount)
		}
		if node.Limit.Offset != nil {
			hint.Offset, _ = rowCount(node.Limit.Offset)
		}
	}
	return map[string]omnisdk.ReadHints{alias: hint}
}
