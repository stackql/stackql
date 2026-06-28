package execution //nolint:testpackage // tests unexported buildPushdownIntent

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
	if len(intent.Projection) != 2 || intent.Projection[0] != "a" || intent.Projection[1] != "b" {
		t.Errorf("projection = %v", intent.Projection)
	}
	if len(intent.Predicates) != 2 {
		t.Fatalf("predicates = %v", intent.Predicates)
	}
	if intent.Predicates[0].Column != "x" || intent.Predicates[0].Operator != "=" || intent.Predicates[0].Value != "v" {
		t.Errorf("predicate[0] = %+v", intent.Predicates[0])
	}
	if intent.Predicates[1].Column != "y" || intent.Predicates[1].Operator != ">" || intent.Predicates[1].Value != 5 {
		t.Errorf("predicate[1] = %+v", intent.Predicates[1])
	}
	if len(intent.OrderBy) != 1 || intent.OrderBy[0].Column != "z" || !intent.OrderBy[0].Descending {
		t.Errorf("orderBy = %v", intent.OrderBy)
	}
	if !intent.LimitSet || intent.Limit != 10 {
		t.Errorf("limit = %d set=%v", intent.Limit, intent.LimitSet)
	}
	if !intent.OffsetSet || intent.Offset != 2 {
		t.Errorf("offset = %d set=%v", intent.Offset, intent.OffsetSet)
	}
}

func TestBuildPushdownIntent_LikePrefixBecomesStartswith(t *testing.T) {
	intent, ok := buildPushdownIntent(mustSelect(t, "select a from t where n like 'A%'"))
	if !ok || len(intent.Predicates) != 1 {
		t.Fatalf("intent ok=%v preds=%v", ok, intent.Predicates)
	}
	p := intent.Predicates[0]
	if p.Column != "n" || p.Operator != "startswith" || p.Value != "A" {
		t.Errorf("predicate = %+v", p)
	}
}

func TestBuildPushdownIntent_LikeNonPrefixIsResidual(t *testing.T) {
	// A non-prefix LIKE is not translatable; it must not appear in predicates.
	intent, _ := buildPushdownIntent(mustSelect(t, "select a from t where n like '%A%'"))
	if len(intent.Predicates) != 0 {
		t.Errorf("expected no pushable predicates, got %v", intent.Predicates)
	}
}

func TestBuildPushdownIntent_CountStar(t *testing.T) {
	intent, ok := buildPushdownIntent(mustSelect(t, "select count(*) as cnt from t"))
	if !ok || !intent.Count {
		t.Errorf("expected Count=true, got ok=%v intent=%+v", ok, intent)
	}
}

func TestBuildPushdownIntent_StarHasNoProjection(t *testing.T) {
	intent, _ := buildPushdownIntent(mustSelect(t, "select * from t where x = 'v'"))
	if len(intent.Projection) != 0 {
		t.Errorf("expected no projection for SELECT *, got %v", intent.Projection)
	}
}

func TestBuildPushdownIntent_JoinIsNotPushable(t *testing.T) {
	// A join could under-fetch if $top/$skip were pushed to one leg.
	_, ok := buildPushdownIntent(mustSelect(t,
		"select a from t1 join t2 on t1.id = t2.id where t1.x = 'v' limit 5"))
	if ok {
		t.Errorf("expected join to be non-pushable")
	}
}

func TestBuildPushdownIntent_OrIsResidual(t *testing.T) {
	// OR is not flattened into pushable AND-conjoined predicates.
	intent, _ := buildPushdownIntent(mustSelect(t, "select a from t where x = 'v' or y = 'w'"))
	if len(intent.Predicates) != 0 {
		t.Errorf("expected OR to be residual, got %v", intent.Predicates)
	}
}
