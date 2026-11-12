package typing //nolint:testpackage // tests unexported behaviour in same package for simplicity

import (
	"testing"

	"github.com/lib/pq/oid"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

func TestRelationalColumnAccessors(t *testing.T) {
	col := NewRelationalColumn("name", "text")
	if col.GetName() != "name" || col.GetType() != "text" {
		t.Fatalf("unexpected name/type: %q/%q", col.GetName(), col.GetType())
	}
	if col.GetAlias() != "" || col.GetQualifier() != "" || col.GetDecorated() != "" || col.GetWidth() != 0 {
		t.Error("expected zero values for optional fields on a new column")
	}
	if _, ok := col.GetOID(); ok {
		t.Error("expected no OID on a new column")
	}

	col = col.WithAlias("al").WithQualifier("q").WithDecorated("dec").WithWidth(42).
		WithUnquote(true).WithParserNode(&sqlparser.AliasedExpr{})
	if col.GetAlias() != "al" || col.GetQualifier() != "q" || col.GetDecorated() != "dec" || col.GetWidth() != 42 {
		t.Errorf("builder methods did not set fields: %q %q %q %d",
			col.GetAlias(), col.GetQualifier(), col.GetDecorated(), col.GetWidth())
	}
}

func TestRelationalColumnOID(t *testing.T) {
	col := NewRelationalColumn("name", "text").WithOID(oid.T_numeric)
	got, ok := col.GetOID()
	if !ok || got != oid.T_numeric {
		t.Errorf("GetOID() = (%d, %t), want (%d, true)", got, ok, oid.T_numeric)
	}
	// zero is a legitimate, explicitly set OID
	col = NewRelationalColumn("name", "text").WithOID(0)
	if got, ok = col.GetOID(); !ok || got != 0 {
		t.Errorf("GetOID() = (%d, %t), want (0, true)", got, ok)
	}
}

func TestRelationalColumnGetIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		col      RelationalColumn
		expected string
	}{
		{"plain", NewRelationalColumn("c", "text"), "c"},
		{"qualified", NewRelationalColumn("c", "text").WithQualifier("q"), "q.c"},
		{"alias wins", NewRelationalColumn("c", "text").WithQualifier("q").WithAlias("a"), "a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.col.GetIdentifier(); got != tt.expected {
				t.Errorf("GetIdentifier() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestRelationalColumnCanonicalSelectionString(t *testing.T) {
	tests := []struct {
		name     string
		col      RelationalColumn
		expected string
	}{
		{"plain", NewRelationalColumn("c", "text"), `"c" `},
		{"qualified", NewRelationalColumn("c", "text").WithQualifier("q"), `"q"."c" `},
		{"aliased", NewRelationalColumn("c", "text").WithAlias("a"), `"c"  AS "a"`},
		{"qualified and aliased", NewRelationalColumn("c", "text").WithQualifier("q").WithAlias("a"), `"q"."c"  AS "a"`},
		{"decorated wins", NewRelationalColumn("c", "text").WithQualifier("q").WithAlias("a").WithDecorated("COUNT(*)"), "COUNT(*) "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.col.CanonicalSelectionString(); got != tt.expected {
				t.Errorf("CanonicalSelectionString() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestRelationalColumnDelimitedSelectionString(t *testing.T) {
	const delim = `"`
	tests := []struct {
		name     string
		col      RelationalColumn
		expected string
	}{
		{"plain", NewRelationalColumn("c", "text"), `"c" `},
		{"qualified", NewRelationalColumn("c", "text").WithQualifier("q"), `"q"."c" `},
		{"aliased", NewRelationalColumn("c", "text").WithAlias("a"), `"c"  AS "a"`},
		{"unquoted keeps alias delimiters", NewRelationalColumn("c", "text").WithAlias("a").WithUnquote(true), `c  AS "a"`},
		{"decorated equal to name", NewRelationalColumn("c", "text").WithDecorated("c"), `"c" `},
		{
			"decorated equal to qualified name",
			NewRelationalColumn("c", "text").WithQualifier("q").WithDecorated("q.c"),
			`"q"."c" `,
		},
		{"decorated other", NewRelationalColumn("c", "text").WithDecorated("COUNT(*)"), "COUNT(*) "},
		{
			"function expression drops delimiters",
			NewRelationalColumn("c", "text").WithAlias("a").
				WithParserNode(&sqlparser.AliasedExpr{Expr: &sqlparser.FuncExpr{}}),
			`c  AS "a"`,
		},
		{
			"non-function expression keeps delimiters",
			NewRelationalColumn("c", "text").WithParserNode(&sqlparser.AliasedExpr{}),
			`"c" `,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.col.DelimitedSelectionString(delim); got != tt.expected {
				t.Errorf("DelimitedSelectionString(%q) = %q, want %q", delim, got, tt.expected)
			}
		})
	}
}
