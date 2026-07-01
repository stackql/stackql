// Package pushdown computes, during the analysis/planning phase, the neutral query-option
// intent (projection / predicates / order-by / limit / offset / count) that an acquire can
// push to the upstream API. It is protocol-agnostic: stackql extracts the intent from the
// SELECT and any-sdk owns the dialect-specific translation and request application (via
// HTTPPreparator.WithPushdownIntent). Push-down is purely an optimisation - stackql's
// client-side WHERE / projection / LIMIT remain authoritative, so a partial or absent
// translation can never change results - and a no-op unless the method carries a
// queryParamPushdown config.
package pushdown

import (
	"strconv"
	"strings"

	"github.com/stackql/any-sdk/public/formulation"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

// ComputeIntent extracts the neutral PushdownIntent from a SELECT for the supplied method.
// It returns (nil, false) when the method carries no queryParamPushdown config, the node is
// not a simple resource-scoped scan, or nothing translatable is present. The caller hands
// the intent to the HTTP preparator, which performs the dialect translation and applies the
// resulting query params to the request inside any-sdk.
func ComputeIntent(node sqlparser.SQLNode, op formulation.OperationStore) (formulation.PushdownIntent, bool) {
	if op == nil {
		return nil, false
	}
	if _, ok := op.GetQueryParamPushdown(); !ok {
		return nil, false
	}
	return buildPushdownIntent(node)
}

// buildPushdownIntent extracts the neutral PushdownIntent from a SELECT statement. The
// second return reports whether anything pushable was found. It is the pure AST-to-intent
// translation, independent of any method config (that gate lives in ComputeIntent).
func buildPushdownIntent(node sqlparser.SQLNode) (formulation.PushdownIntent, bool) {
	sel, ok := node.(*sqlparser.Select)
	if !ok || !isSimpleResourceScopedSelect(sel) {
		return nil, false
	}

	var predicates []formulation.PushdownPredicate
	if sel.Where != nil {
		predicates = collectPushdownPredicates(sel.Where.Expr)
	}

	projection, count := extractProjectionAndCount(sel.SelectExprs)
	if count {
		// COUNT(*) collapses grain: push the WHERE (applied before counting) and the count
		// itself, but never limit/offset/projection/order-by - they would change or misreport
		// the count. SQL LIMIT here reverts to a client-side primitive.
		return formulation.NewPushdownIntent(nil, predicates, nil, 0, false, 0, false, true), true
	}

	var orderBy []formulation.PushdownOrder
	for _, o := range sel.OrderBy {
		if cn, isCol := o.Expr.(*sqlparser.ColName); isCol {
			orderBy = append(orderBy, formulation.NewPushdownOrder(
				cn.Name.GetRawVal(),
				strings.Contains(strings.ToLower(o.Direction), "desc"),
			))
		}
	}

	limit, limitSet := 0, false
	offset, offsetSet := 0, false
	if sel.Limit != nil {
		if v, limOk := sqlValAsInt(sel.Limit.Rowcount); limOk {
			limit, limitSet = v, true
		}
		if v, offOk := sqlValAsInt(sel.Limit.Offset); offOk {
			offset, offsetSet = v, true
		}
	}

	hasContent := len(projection) > 0 || len(predicates) > 0 || len(orderBy) > 0 || limitSet || offsetSet
	if !hasContent {
		return nil, false
	}
	return formulation.NewPushdownIntent(
		projection, predicates, orderBy, limit, limitSet, offset, offsetSet, false), true
}

// SelectLimit returns the integer LIMIT of a SELECT when it can be pushed to the upstream
// fetch: only for a simple, resource-scoped scan (one table, no join / GROUP BY / DISTINCT
// / HAVING). For a grain-changing or multi-set query the LIMIT stays a client-side primitive.
// It is used by the GraphQL acquire path to bound the page size.
func SelectLimit(node sqlparser.SQLNode) (int, bool) {
	sel, ok := node.(*sqlparser.Select)
	if !ok || sel.Limit == nil || !isSimpleResourceScopedSelect(sel) {
		return 0, false
	}
	return sqlValAsInt(sel.Limit.Rowcount)
}

// isSimpleResourceScopedSelect reports whether a SELECT is a single-resource scan with
// no grain change, i.e. safe to translate LIMIT/OFFSET/projection into server predicates.
func isSimpleResourceScopedSelect(sel *sqlparser.Select) bool {
	if len(sel.From) != 1 {
		return false
	}
	if _, isSimple := sel.From[0].(*sqlparser.AliasedTableExpr); !isSimple {
		return false
	}
	return !sel.Distinct && len(sel.GroupBy) == 0 && sel.Having == nil
}

// extractProjectionAndCount reads the SELECT list, returning the projected column
// names (nil for SELECT *) and whether a COUNT(*) aggregate is present.
func extractProjectionAndCount(exprs sqlparser.SelectExprs) ([]string, bool) {
	var projection []string
	var count bool
	for _, e := range exprs {
		ae, ok := e.(*sqlparser.AliasedExpr)
		if !ok {
			continue
		}
		switch inner := ae.Expr.(type) {
		case *sqlparser.ColName:
			projection = append(projection, inner.Name.GetRawVal())
		case *sqlparser.FuncExpr:
			if strings.EqualFold(inner.Name.GetRawVal(), "count") {
				count = true
			}
		}
	}
	return projection, count
}

// collectPushdownPredicates flattens AND-conjoined simple comparisons into neutral
// predicates. Anything not translatable (OR, non-column LHS, unsupported operator)
// is simply omitted - it stays a client-side filter via the unchanged WHERE clause.
func collectPushdownPredicates(expr sqlparser.Expr) []formulation.PushdownPredicate {
	switch e := expr.(type) {
	case *sqlparser.AndExpr:
		return append(collectPushdownPredicates(e.Left), collectPushdownPredicates(e.Right)...)
	case *sqlparser.ComparisonExpr:
		if p, ok := comparisonToPredicate(e); ok {
			return []formulation.PushdownPredicate{p}
		}
	}
	return nil
}

func comparisonToPredicate(expr *sqlparser.ComparisonExpr) (formulation.PushdownPredicate, bool) {
	col, ok := expr.Left.(*sqlparser.ColName)
	if !ok {
		return nil, false
	}
	val, ok := expr.Right.(*sqlparser.SQLVal)
	if !ok {
		return nil, false
	}
	column := col.Name.GetRawVal()
	rawVal, valOk := sqlValAsComparable(val)
	if !valOk {
		return nil, false
	}
	switch expr.Operator {
	case sqlparser.EqualStr, sqlparser.NotEqualStr, sqlparser.GreaterThanStr,
		sqlparser.GreaterEqualStr, sqlparser.LessThanStr, sqlparser.LessEqualStr:
		return formulation.NewPushdownPredicate(column, expr.Operator, rawVal), true
	case sqlparser.LikeStr:
		// Only a prefix pattern ('A%' with no other wildcard) maps to a startswith predicate.
		s, isStr := rawVal.(string)
		if isStr && strings.HasSuffix(s, "%") {
			prefix := strings.TrimSuffix(s, "%")
			if !strings.ContainsAny(prefix, "%_") {
				return formulation.NewPushdownPredicate(column, "startswith", prefix), true
			}
		}
	}
	return nil, false
}

// sqlValAsComparable renders a SQLVal as a Go value ApplyPushdown can format:
// integers as int, everything else as the raw string (quoted by any-sdk).
func sqlValAsComparable(v *sqlparser.SQLVal) (interface{}, bool) {
	if v == nil {
		return nil, false
	}
	switch v.Type { //nolint:exhaustive // only int/string are translated; others use the raw string
	case sqlparser.IntVal:
		if i, err := strconv.Atoi(string(v.Val)); err == nil {
			return i, true
		}
		return string(v.Val), true
	case sqlparser.StrVal:
		return string(v.Val), true
	default:
		return string(v.Val), true
	}
}

func sqlValAsInt(expr sqlparser.Expr) (int, bool) {
	v, ok := expr.(*sqlparser.SQLVal)
	if !ok || v.Type != sqlparser.IntVal {
		return 0, false
	}
	i, err := strconv.Atoi(string(v.Val))
	if err != nil {
		return 0, false
	}
	return i, true
}
