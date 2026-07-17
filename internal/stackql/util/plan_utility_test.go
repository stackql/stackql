package util //nolint:testpackage // exercise unexported helpers

import (
	"testing"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

func mustParse(t *testing.T, sql string) sqlparser.Statement {
	t.Helper()
	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		t.Fatalf("parse %q: %v", sql, err)
	}
	return stmt
}

func TestExtractSQLNodeParams_ScalarOnlyIsSingleSet(t *testing.T) {
	stmt := mustParse(t, "delete from t where project = 'p1' and firewall = 'f1'")
	got, err := ExtractSQLNodeParams(stmt, nil)
	if err != nil {
		t.Fatalf("ExtractSQLNodeParams: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected a single parameter set, got %d", len(got))
	}
	if got[0]["project"] != "p1" || got[0]["firewall"] != "f1" {
		t.Errorf("unexpected parameter set: %v", got[0])
	}
}

func TestExtractSQLNodeParams_InListFansOutPerElement(t *testing.T) {
	// Issue #683: each IN-list element yields its own parameter set, with the
	// scalar parameters replicated into every set.
	stmt := mustParse(t, "delete from t where project = 'p1' and firewall in ('f1', 'f2', 'f3')")
	got, err := ExtractSQLNodeParams(stmt, nil)
	if err != nil {
		t.Fatalf("ExtractSQLNodeParams: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 parameter sets, got %d: %v", len(got), got)
	}
	seen := map[string]bool{}
	for _, row := range got {
		if row["project"] != "p1" {
			t.Errorf("scalar param not replicated: %v", row)
		}
		fw, isStr := row["firewall"].(string)
		if !isStr {
			t.Fatalf("firewall element not a string: %v", row["firewall"])
		}
		seen[fw] = true
	}
	for _, want := range []string{"f1", "f2", "f3"} {
		if !seen[want] {
			t.Errorf("missing parameter set for element %q; got %v", want, seen)
		}
	}
}

func TestExtractSQLNodeParams_MultipleInListsProduceCartesianProduct(t *testing.T) {
	stmt := mustParse(t, "delete from t where a in ('a1', 'a2') and b in ('b1', 'b2')")
	got, err := ExtractSQLNodeParams(stmt, nil)
	if err != nil {
		t.Fatalf("ExtractSQLNodeParams: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("expected 4 parameter sets (2x2 product), got %d: %v", len(got), got)
	}
	combos := map[string]bool{}
	for _, row := range got {
		combos[row["a"].(string)+"/"+row["b"].(string)] = true
	}
	for _, want := range []string{"a1/b1", "a1/b2", "a2/b1", "a2/b2"} {
		if !combos[want] {
			t.Errorf("missing combination %q; got %v", want, combos)
		}
	}
}

func TestExtractSQLNodeParams_NotInIsNotFannedOut(t *testing.T) {
	// NOT IN remains a client-side filter: it must not multiply parameter sets.
	stmt := mustParse(t, "delete from t where project = 'p1' and firewall not in ('f1', 'f2')")
	got, err := ExtractSQLNodeParams(stmt, nil)
	if err != nil {
		t.Fatalf("ExtractSQLNodeParams: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected a single parameter set for NOT IN, got %d", len(got))
	}
	if _, present := got[0]["firewall"]; present {
		t.Errorf("NOT IN must not supply the parameter, got %v", got[0])
	}
}
