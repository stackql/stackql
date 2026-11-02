package handler //nolint:testpackage // Verify the private session store and its snapshots.

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

func TestScalarVariableResolution(t *testing.T) {
	hc := sessionTestHandler()
	for name, value := range map[string]string{"word": `"hello"`, "number": `9007199254740993`, "decimal": `1.25`, "yes": `true`, "empty": `null`} {
		if err := hc.SetConfigAtPath("variables."+name, json.RawMessage(value), ""); err != nil {
			t.Fatal(err)
		}
	}
	statement, err := sqlparser.Parse("select upper($word), $number + 1, $decimal, $yes, $empty")
	if err != nil {
		t.Fatal(err)
	}
	if err = hc.ResolveVariables(statement); err != nil {
		t.Fatal(err)
	}
	rendered := sqlparser.String(statement)
	for _, expected := range []string{"upper('hello')", "9007199254740993", "1.25", "true", "null"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("missing %s in %s", expected, rendered)
		}
	}
	if err = hc.SetConfigAtPath("variables.word", json.RawMessage(`"changed"`), ""); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(sqlparser.String(statement), "changed") {
		t.Fatal("statement snapshot mutated")
	}
	for _, bad := range []json.RawMessage{json.RawMessage(`{}`), json.RawMessage(`[]`)} {
		if err = hc.SetConfigAtPath("variables.bad", bad, ""); err == nil {
			t.Fatal("accepted nonscalar variable")
		}
	}
	if err = hc.SetConfigAtPath("variables.word", nil, ""); err != nil {
		t.Fatal(err)
	}
	statement, err = sqlparser.Parse("select $word")
	if err != nil {
		t.Fatal(err)
	}
	if err = hc.ResolveVariables(statement); err == nil {
		t.Fatal("missing variable resolved as null")
	}
}

func TestSecretVariableOptIn(t *testing.T) {
	hc := sessionTestHandler()
	if err := hc.SetConfigAtPath("auth.widgets.password", "private-value", ""); err != nil {
		t.Fatal(err)
	}
	statement, err := sqlparser.Parse(`select "$auth.widgets.password"`)
	if err != nil {
		t.Fatal(err)
	}
	if err = hc.ResolveVariables(statement); err == nil || strings.Contains(err.Error(), "private-value") {
		t.Fatal("secret reference was not safely refused")
	}
	hc.runtimeContext.AllowSecretDereference = true
	if err = hc.ResolveVariables(statement); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sqlparser.String(statement), "'private-value'") {
		t.Fatal("opt-in did not resolve secret")
	}
}

func TestVariablesRemainClientLocal(t *testing.T) {
	parent := sessionTestHandler()
	if err := parent.SetConfigAtPath("variables.x", json.RawMessage(`1`), ""); err != nil {
		t.Fatal(err)
	}
	first := parent.NewClientContext()
	second := parent.NewClientContext()
	if err := first.SetConfigAtPath("variables.x", json.RawMessage(`2`), ""); err != nil {
		t.Fatal(err)
	}
	statement, err := sqlparser.Parse("select $x")
	if err != nil {
		t.Fatal(err)
	}
	if err = second.ResolveVariables(statement); err != nil {
		t.Fatal(err)
	}
	selected := statement.(*sqlparser.Select).SelectExprs[0].(*sqlparser.AliasedExpr).Expr.(*sqlparser.SQLVal)
	if string(selected.Val) != "1" {
		t.Fatal("client variable state leaked")
	}
}
