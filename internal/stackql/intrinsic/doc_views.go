package intrinsic

import (
	"fmt"
	"strings"

	"github.com/stackql-labs/omnisdk/pkg/docparse/aot"
	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql-labs/omnisdk/pkg/sqlfn"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

// expandDocumentViews substitutes a single FROM view before the normal SDK
// planner runs. SQL expressions remain AST nodes and are evaluated by the same
// configured backend as ordinary projections. No view is executed while planning.
//
//nolint:errcheck,gocognit,funlen // Parsed SELECT AST shape and bounded recursive expansion are intentional.
func expandDocumentViews(
	ctx queryContext,
	original *sqlparser.Select,
	current string,
) (*sqlparser.Select, error) {
	node := original
	seen := map[string]bool{}
	for depth := 0; depth < 16; depth++ {
		if len(node.From) != 1 {
			return node, nil
		}
		table, ok := node.From[0].(*sqlparser.AliasedTableExpr)
		if !ok {
			return node, nil
		}
		name, ok := table.Expr.(sqlparser.TableName)
		if !ok {
			return node, nil
		}
		bundle, isDoc := docProvider(resolveProvider(name.QualifierSecond.GetRawVal(), current))
		if !isDoc {
			return node, nil
		}
		service, resource := name.Qualifier.GetRawVal(), name.Name.GetRawVal()
		root, err := docRoot(ctx, bundle)
		if err != nil {
			return nil, err
		}
		view, exists, err := omnisdk.DocView(root, bundle, service, resource)
		if err != nil {
			return nil, err
		}
		if !exists {
			return node, nil
		}
		key := bundle + "." + service + "." + resource
		if seen[key] {
			return nil, fmt.Errorf("document view cycle at %s", key)
		}
		seen[key] = true
		dialect, err := backendDialect(ctx)
		if err != nil {
			return nil, err
		}
		ddl, err := viewSQL(view, dialect.catalogue())
		if err != nil {
			return nil, fmt.Errorf("view %s: %w", key, err)
		}
		stmt, err := sqlparser.Parse(ddl)
		if err != nil {
			return nil, fmt.Errorf("view %s: %w", key, err)
		}
		projection, ok := stmt.(*sqlparser.Select)
		if !ok {
			return nil, fmt.Errorf("view %s must declare a SELECT", key)
		}
		if len(projection.GroupBy) > 0 || projection.Having != nil || projection.Distinct ||
			projection.Limit != nil || len(projection.OrderBy) > 0 {
			return nil, fmt.Errorf("view %s requires a separate relational staging boundary", key)
		}
		// Clone the caller before replacing any AST references.
		clone, err := sqlparser.Parse(sqlparser.String(node))
		if err != nil {
			return nil, err
		}
		node = clone.(*sqlparser.Select)
		alias := table.As.GetRawVal()
		if alias == "" {
			alias = resource
		}
		node, err = inlineDocumentView(node, projection, alias)
		if err != nil {
			return nil, fmt.Errorf("view %s: %w", key, err)
		}
	}
	return nil, fmt.Errorf("document views exceed the expansion depth bound")
}

func viewSQL(view aot.View, dialect sqlfn.Dialect) (string, error) {
	target := "sqlite3"
	if dialect == sqlfn.Postgres {
		target = "postgres"
	}
	for depth := 0; depth < 16; depth++ {
		condition := strings.Join(strings.Fields(view.Predicate()), "")
		matches := condition == "" || condition == "true"
		if strings.HasPrefix(condition, "sqlDialect==") {
			value := strings.Trim(strings.TrimPrefix(condition, "sqlDialect=="), "\"'")
			matches = value == target
		} else if condition != "" && condition != "true" && condition != "false" {
			return "", fmt.Errorf("unsupported view predicate %q", view.Predicate())
		}
		if matches {
			return view.DDL(), nil
		}
		next, ok := view.Fallback()
		if !ok {
			return "", fmt.Errorf("no view for SQL dialect %s", target)
		}
		view = next
	}
	return "", fmt.Errorf("view fallback exceeds the depth bound")
}

//nolint:errcheck,govet,gocognit,gocyclo,nestif,cyclop,funlen // AST rewriting keeps these cases together.
func inlineDocumentView(node, view *sqlparser.Select, alias string) (*sqlparser.Select, error) {
	columns := map[string]sqlparser.Expr{}
	for _, selection := range view.SelectExprs {
		selected, ok := selection.(*sqlparser.AliasedExpr)
		if !ok {
			return nil, fmt.Errorf("view must name its projected columns")
		}
		name := selected.As.GetRawVal()
		if name == "" {
			if col, ok := selected.Expr.(*sqlparser.ColName); ok {
				name = col.Name.GetRawVal()
			}
		}
		if name == "" {
			return nil, fmt.Errorf("view expression must declare an alias")
		}
		if _, duplicate := columns[name]; duplicate {
			return nil, fmt.Errorf("view has duplicate column %s", name)
		}
		columns[name] = selected.Expr
	}
	rewrite := func(expr sqlparser.SQLNode) sqlparser.SQLNode {
		return sqlparser.Rewrite(expr, func(c *sqlparser.Cursor) bool {
			col, ok := c.Node().(*sqlparser.ColName)
			if !ok {
				return true
			}
			qualifier := col.Qualifier.Name.GetRawVal()
			if qualifier != "" && qualifier != alias {
				return false
			}
			if replacement, ok := columns[col.Name.GetRawVal()]; ok {
				c.Replace(replacement)
			}
			return false
		}, nil)
	}
	selections := sqlparser.SelectExprs{}
	for _, selection := range node.SelectExprs {
		if star, ok := selection.(*sqlparser.StarExpr); ok {
			if qualifier := star.TableName.Name.GetRawVal(); qualifier != "" && qualifier != alias {
				return nil, fmt.Errorf("unknown view star qualifier %s", qualifier)
			}
			selections = append(selections, view.SelectExprs...)
			continue
		}
		selected, ok := selection.(*sqlparser.AliasedExpr)
		if !ok {
			return nil, fmt.Errorf("unsupported view projection")
		}
		if col, ok := selected.Expr.(*sqlparser.ColName); ok && selected.As.GetRawVal() == "" {
			selected.As = col.Name
		}
		selected.Expr = rewrite(selected.Expr).(sqlparser.Expr)
		selections = append(selections, selected)
	}
	node.SelectExprs = selections
	// Explicit scope/body bindings on the outer query override the view's
	// declared defaults; business predicates remain conjunctive.
	bindings := map[string]bool{}
	if node.Where != nil {
		_ = sqlparser.Walk(func(n sqlparser.SQLNode) (bool, error) {
			if compare, ok := n.(*sqlparser.ComparisonExpr); ok && compare.Operator == sqlparser.EqualStr {
				if col, ok := compare.Left.(*sqlparser.ColName); ok {
					name := col.Name.GetRawVal()
					if name == "region" || strings.HasPrefix(name, "data__") {
						bindings[name] = true
					}
				}
			}
			return true, nil
		}, node.Where.Expr)
		node.Where.Expr = rewrite(node.Where.Expr).(sqlparser.Expr)
	}
	var strip func(sqlparser.Expr) sqlparser.Expr
	strip = func(expr sqlparser.Expr) sqlparser.Expr {
		if and, ok := expr.(*sqlparser.AndExpr); ok {
			lhs, rhs := strip(and.Left), strip(and.Right)
			if lhs == nil {
				return rhs
			}
			if rhs == nil {
				return lhs
			}
			return &sqlparser.AndExpr{Left: lhs, Right: rhs}
		}
		if compare, ok := expr.(*sqlparser.ComparisonExpr); ok && compare.Operator == sqlparser.EqualStr {
			if col, ok := compare.Left.(*sqlparser.ColName); ok && bindings[col.Name.GetRawVal()] {
				return nil
			}
		}
		return expr
	}
	if view.Where != nil {
		condition := strip(view.Where.Expr)
		if condition != nil {
			if node.Where == nil {
				node.Where = &sqlparser.Where{Type: sqlparser.WhereStr, Expr: condition}
			} else {
				node.Where.Expr = &sqlparser.AndExpr{Left: node.Where.Expr, Right: condition}
			}
		}
	}
	for i, group := range node.GroupBy {
		node.GroupBy[i] = rewrite(group).(sqlparser.Expr)
	}
	if node.Having != nil {
		node.Having.Expr = rewrite(node.Having.Expr).(sqlparser.Expr)
	}
	for _, term := range node.OrderBy {
		term.Expr = rewrite(term.Expr).(sqlparser.Expr)
	}
	node.From = view.From
	return node, nil
}
