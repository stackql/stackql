// Package pushdown computes, during the analysis/planning phase, the request query
// parameters that an acquire should push to the upstream API. It is protocol-agnostic:
// stackql extracts a neutral, dialect-free intent (projection / predicates / order-by /
// limit / offset / count) from the SELECT and hands it to any-sdk's ApplyPushdown, which
// owns any dialect-specific translation and returns the params to set. The result is
// attached to the plan; the executor merely carries it out. Push-down is purely an
// optimisation - stackql's client-side WHERE / projection / LIMIT remain authoritative,
// so a partial or absent translation can never change results - and a no-op unless the
// method carries a queryParamPushdown config.
package pushdown

import (
	"strconv"
	"strings"

	"github.com/stackql/any-sdk/public/formulation"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

// ComputeQueryParams translates a SELECT statement into the request query params to push
// down for the supplied config source (typically the acquire's OperationStore). It returns
// nil when there is no pushdown config or nothing translatable.
func ComputeQueryParams(node sqlparser.SQLNode, src formulation.PushdownConfigSource) map[string]string {
	if src == nil || node == nil {
		return nil
	}
	if _, ok := src.GetQueryParamPushdown(); !ok {
		return nil
	}
	intent, ok := buildPushdownIntent(node)
	if !ok {
		return nil
	}
	result := formulation.ApplyPushdown(src, intent)
	queryParams := result.QueryParams()
	if len(queryParams) == 0 {
		return nil
	}
	return queryParams
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

// buildPushdownIntent extracts a neutral PushdownIntent from a SELECT statement.
// The second return reports whether anything pushable was found.
func buildPushdownIntent(node sqlparser.SQLNode) (formulation.PushdownIntent, bool) {
	var intent formulation.PushdownIntent
	sel, ok := node.(*sqlparser.Select)
	if !ok {
		return intent, false
	}
	// Only push down for a simple, resource-scoped scan: one table, no join, and no
	// grain change (GROUP BY / DISTINCT / HAVING). Otherwise LIMIT/OFFSET and the
	// projection apply to the post-aggregation / post-join result, so pushing them to
	// the upstream fetch could under-fetch or mis-shape rows; they must stay as
	// client-side (DB engine) primitives.
	if !isSimpleResourceScopedSelect(sel) {
		return intent, false
	}

	if sel.Where != nil {
		intent.Predicates = collectPushdownPredicates(sel.Where.Expr)
	}

	_, intent.Count = extractProjectionAndCount(sel.SelectExprs)
	if intent.Count {
		// COUNT(*) collapses grain: push the WHERE (applied before counting) and the
		// count itself, but never limit/offset/projection/order-by - they would change
		// or misreport the count. SQL LIMIT here reverts to a client-side primitive.
		return intent, len(intent.Predicates) > 0 || intent.Count
	}

	intent.Projection, _ = extractProjectionAndCount(sel.SelectExprs)

	for _, o := range sel.OrderBy {
		if cn, isCol := o.Expr.(*sqlparser.ColName); isCol {
			intent.OrderBy = append(intent.OrderBy, formulation.PushdownOrder{
				Column:     cn.Name.GetRawVal(),
				Descending: strings.Contains(strings.ToLower(o.Direction), "desc"),
			})
		}
	}

	if sel.Limit != nil {
		if v, limOk := sqlValAsInt(sel.Limit.Rowcount); limOk {
			intent.Limit = v
			intent.LimitSet = true
		}
		if v, offOk := sqlValAsInt(sel.Limit.Offset); offOk {
			intent.Offset = v
			intent.OffsetSet = true
		}
	}

	hasContent := len(intent.Projection) > 0 || len(intent.Predicates) > 0 ||
		len(intent.OrderBy) > 0 || intent.LimitSet || intent.OffsetSet
	return intent, hasContent
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
	var pred formulation.PushdownPredicate
	col, ok := expr.Left.(*sqlparser.ColName)
	if !ok {
		return pred, false
	}
	val, ok := expr.Right.(*sqlparser.SQLVal)
	if !ok {
		return pred, false
	}
	column := col.Name.GetRawVal()
	rawVal, valOk := sqlValAsComparable(val)
	if !valOk {
		return pred, false
	}
	switch expr.Operator {
	case sqlparser.EqualStr, sqlparser.NotEqualStr, sqlparser.GreaterThanStr,
		sqlparser.GreaterEqualStr, sqlparser.LessThanStr, sqlparser.LessEqualStr:
		return formulation.PushdownPredicate{Column: column, Operator: expr.Operator, Value: rawVal}, true
	case sqlparser.LikeStr:
		// Only a prefix pattern ('A%' with no other wildcard) maps to a startswith predicate.
		s, isStr := rawVal.(string)
		if isStr && strings.HasSuffix(s, "%") {
			prefix := strings.TrimSuffix(s, "%")
			if !strings.ContainsAny(prefix, "%_") {
				return formulation.PushdownPredicate{Column: column, Operator: "startswith", Value: prefix}, true
			}
		}
	}
	return pred, false
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
