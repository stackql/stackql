package execution

import (
	"strconv"
	"strings"

	"github.com/stackql/any-sdk/public/formulation"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

// odata_pushdown wires the any-sdk query-option push-down (ApplyPushdown) into the
// REST acquire path. It is purely an optimisation: the translated query options
// ($filter/$select/$orderby/$top/$skip/$count) are added to the outgoing request so
// the upstream API can pre-filter, but stackql's existing client-side WHERE /
// projection / LIMIT remain authoritative, so a partial or absent push-down can
// never change results. It is a no-op unless the method carries a queryParamPushdown
// config, so non-pushdown providers are unaffected.

// pushdownArmouryGenerator decorates a BaseArmouryGenerator, merging the push-down
// query parameters into every request param's query string.
type pushdownArmouryGenerator struct {
	prior       formulation.BaseArmouryGenerator
	queryParams map[string]string
}

func (g *pushdownArmouryGenerator) GetHTTPArmoury() (formulation.HTTPArmoury, error) {
	armoury, err := g.prior.GetHTTPArmoury()
	if err != nil || len(g.queryParams) == 0 {
		return armoury, err
	}
	params := armoury.GetRequestParams()
	for i, p := range params {
		param := p
		q := param.GetQuery()
		for k, v := range g.queryParams {
			q.Set(k, v)
		}
		param.SetRawQuery(q.Encode())
		params[i] = param
	}
	armoury.SetRequestParams(params)
	return armoury, nil
}

// maybeDecorateWithODataPushdown returns a generator that injects push-down query
// params when the method opts in and the statement yields a non-empty translation;
// otherwise it returns the prior generator unchanged.
func maybeDecorateWithODataPushdown(
	prior formulation.BaseArmouryGenerator,
	node sqlparser.SQLNode,
	method formulation.OperationStore,
) formulation.BaseArmouryGenerator {
	if method == nil || node == nil {
		return prior
	}
	if _, ok := method.GetQueryParamPushdown(); !ok {
		return prior
	}
	intent, ok := buildPushdownIntent(node)
	if !ok {
		return prior
	}
	result := formulation.ApplyPushdown(method, intent)
	queryParams := result.QueryParams()
	if len(queryParams) == 0 {
		return prior
	}
	return &pushdownArmouryGenerator{prior: prior, queryParams: queryParams}
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
		// count itself, but never $top/$skip/$select/$orderby - they would change or
		// misreport the count. SQL LIMIT here reverts to a client-side primitive.
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
		// Only a prefix pattern ('A%' with no other wildcard) maps to startswith.
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
// integers as int, everything else as the raw string (single-quoted by any-sdk).
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
