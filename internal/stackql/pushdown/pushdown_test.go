package pushdown //nolint:testpackage // tests unexported buildPushdownIntent

import (
	"testing"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

func mustSelect(t *testing.T, sql string) sqlparser.SQLNode {
	t.Helper()
	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		t.Fatalf("parse %q: %v", sql, err)
	}
	return stmt
}

func TestBuildPushdownIntent_FullSelect(t *testing.T) {
	node := mustSelect(t,
		"select a, b from t where x = 'v' and y > 5 order by z desc limit 10 offset 2")
	intent, ok := buildPushdownIntent(node)
	if !ok {
		t.Fatalf("expected pushable intent")
	}
	// The projection is the union of SELECT-list columns and the WHERE / ORDER BY
	// referenced columns, in that order (issue #682).
	proj := intent.GetProjection()
	want := []string{"a", "b", "x", "y", "z"}
	if len(proj) != len(want) {
		t.Fatalf("projection = %v, want %v", proj, want)
	}
	for i := range want {
		if proj[i] != want[i] {
			t.Errorf("projection[%d] = %s, want %s", i, proj[i], want[i])
		}
	}
	preds := intent.GetPredicates()
	if len(preds) != 2 {
		t.Fatalf("predicates = %v", preds)
	}
	if preds[0].GetColumn() != "x" || preds[0].GetOperator() != "=" || preds[0].GetValue() != "v" {
		t.Errorf("predicate[0] = col=%s op=%s val=%v",
			preds[0].GetColumn(), preds[0].GetOperator(), preds[0].GetValue())
	}
	if preds[1].GetColumn() != "y" || preds[1].GetOperator() != ">" || preds[1].GetValue() != 5 {
		t.Errorf("predicate[1] = col=%s op=%s val=%v",
			preds[1].GetColumn(), preds[1].GetOperator(), preds[1].GetValue())
	}
	ob := intent.GetOrderBy()
	if len(ob) != 1 || ob[0].GetColumn() != "z" || !ob[0].IsDescending() {
		t.Errorf("orderBy = %v", ob)
	}
	if l, set := intent.GetLimit(); !set || l != 10 {
		t.Errorf("limit = %d set=%v", l, set)
	}
	if o, set := intent.GetOffset(); !set || o != 2 {
		t.Errorf("offset = %d set=%v", o, set)
	}
}

func TestBuildPushdownIntent_LikePrefixBecomesStartswith(t *testing.T) {
	intent, ok := buildPushdownIntent(mustSelect(t, "select a from t where n like 'A%'"))
	if !ok {
		t.Fatalf("expected pushable intent")
	}
	preds := intent.GetPredicates()
	if len(preds) != 1 {
		t.Fatalf("preds = %v", preds)
	}
	p := preds[0]
	if p.GetColumn() != "n" || p.GetOperator() != "startswith" || p.GetValue() != "A" {
		t.Errorf("predicate = col=%s op=%s val=%v", p.GetColumn(), p.GetOperator(), p.GetValue())
	}
}

func TestBuildPushdownIntent_LikeNonPrefixIsResidual(t *testing.T) {
	// A non-prefix LIKE is not translatable; it must not appear in predicates.
	intent, ok := buildPushdownIntent(mustSelect(t, "select a from t where n like '%A%'"))
	if !ok {
		t.Fatalf("expected an intent (projection present)")
	}
	if len(intent.GetPredicates()) != 0 {
		t.Errorf("expected no pushable predicates, got %v", intent.GetPredicates())
	}
}

func TestBuildPushdownIntent_CountStar(t *testing.T) {
	intent, ok := buildPushdownIntent(mustSelect(t, "select count(*) as cnt from t"))
	if !ok || !intent.IsCount() {
		t.Errorf("expected Count=true, got ok=%v isCount=%v", ok, ok && intent.IsCount())
	}
}

func TestBuildPushdownIntent_StarHasNoProjection(t *testing.T) {
	intent, ok := buildPushdownIntent(mustSelect(t, "select * from t where x = 'v'"))
	if !ok {
		t.Fatalf("expected an intent (predicate present)")
	}
	if len(intent.GetProjection()) != 0 {
		t.Errorf("expected no projection for SELECT *, got %v", intent.GetProjection())
	}
}

func TestBuildPushdownIntent_JoinIsNotPushable(t *testing.T) {
	// A join could under-fetch if a limit/offset were pushed to one leg.
	_, ok := buildPushdownIntent(mustSelect(t,
		"select a from t1 join t2 on t1.id = t2.id where t1.x = 'v' limit 5"))
	if ok {
		t.Errorf("expected join to be non-pushable")
	}
}

func TestBuildPushdownIntent_OrIsResidual(t *testing.T) {
	// OR is not flattened into pushable AND-conjoined predicates.
	intent, ok := buildPushdownIntent(mustSelect(t, "select a from t where x = 'v' or y = 'w'"))
	if !ok {
		t.Fatalf("expected an intent (projection present)")
	}
	if len(intent.GetPredicates()) != 0 {
		t.Errorf("expected OR to be residual, got %v", intent.GetPredicates())
	}
}

func TestBuildPushdownIntent_GrainChangeIsNotPushable(t *testing.T) {
	// GROUP BY / DISTINCT / HAVING change grain: LIMIT/projection must stay client-side.
	cases := []string{
		"select city, count(*) as c from t group by city limit 5",
		"select distinct city from t limit 5",
		"select city from t group by city having count(*) > 1",
	}
	for _, sql := range cases {
		if _, ok := buildPushdownIntent(mustSelect(t, sql)); ok {
			t.Errorf("expected non-pushable for %q", sql)
		}
	}
}

func TestBuildPushdownIntent_ProjectionUnionsWhereAndOrderColumns(t *testing.T) {
	// Issue #682: a WHERE-only column must reach the pushed projection or the
	// authoritative client-side filter sees an absent column and drops all rows.
	intent, ok := buildPushdownIntent(mustSelect(t,
		"select id, displayName from users where userType = 'Member'"))
	if !ok {
		t.Fatalf("expected pushable intent")
	}
	proj := intent.GetProjection()
	want := []string{"id", "displayName", "userType"}
	if len(proj) != len(want) {
		t.Fatalf("projection = %v, want %v", proj, want)
	}
	for i := range want {
		if proj[i] != want[i] {
			t.Errorf("projection[%d] = %s, want %s", i, proj[i], want[i])
		}
	}
}

func TestBuildPushdownIntent_ProjectionUnionIncludesResidualPredicateColumns(t *testing.T) {
	// OR branches are not pushable predicates, but their columns are still needed
	// client-side, so they must be fetched.
	intent, ok := buildPushdownIntent(mustSelect(t,
		"select a from t where x = 'v' or y = 'w'"))
	if !ok {
		t.Fatalf("expected an intent (projection present)")
	}
	if len(intent.GetPredicates()) != 0 {
		t.Errorf("expected OR to remain residual, got %v", intent.GetPredicates())
	}
	proj := intent.GetProjection()
	want := []string{"a", "x", "y"}
	if len(proj) != len(want) {
		t.Fatalf("projection = %v, want %v", proj, want)
	}
	for i := range want {
		if proj[i] != want[i] {
			t.Errorf("projection[%d] = %s, want %s", i, proj[i], want[i])
		}
	}
}

func TestBuildPushdownIntent_ProjectionUnionDeduplicates(t *testing.T) {
	intent, ok := buildPushdownIntent(mustSelect(t,
		"select a, b from t where a = 'v' and b > 1 order by a asc"))
	if !ok {
		t.Fatalf("expected pushable intent")
	}
	proj := intent.GetProjection()
	if len(proj) != 2 || proj[0] != "a" || proj[1] != "b" {
		t.Errorf("expected deduplicated projection [a b], got %v", proj)
	}
}

func TestBuildPushdownIntent_CountSuppressesLimitAndProjection(t *testing.T) {
	// A bare COUNT(*) pushes WHERE + count, but never top/skip/select/orderby.
	intent, ok := buildPushdownIntent(mustSelect(t,
		"select count(*) as cnt from t where x = 'v' order by y limit 5 offset 2"))
	if !ok || !intent.IsCount() {
		t.Fatalf("expected Count intent, got ok=%v", ok)
	}
	if len(intent.GetPredicates()) != 1 {
		t.Errorf("expected the WHERE to still push, got %v", intent.GetPredicates())
	}
	_, limSet := intent.GetLimit()
	_, offSet := intent.GetOffset()
	if limSet || offSet || len(intent.GetProjection()) != 0 || len(intent.GetOrderBy()) != 0 {
		t.Errorf("expected top/skip/select/orderby suppressed for COUNT")
	}
}
