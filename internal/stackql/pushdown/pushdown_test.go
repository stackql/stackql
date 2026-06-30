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
	proj := intent.GetProjection()
	if len(proj) != 2 || proj[0] != "a" || proj[1] != "b" {
		t.Errorf("projection = %v", proj)
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
