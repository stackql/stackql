package render_test

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/stackql/stackql/pkg/mcp_server/render"
)

func TestRenderTable_Empty(t *testing.T) {
	got := render.RenderTable(nil)
	if !strings.Contains(got, "no results") {
		t.Fatalf("expected 'no results' for empty input, got %q", got)
	}
}

func TestRenderTable_StableColumnOrder(t *testing.T) {
	rows := []map[string]any{
		{"b": 1, "a": "x"},
		{"a": "y", "b": 2, "c": true},
	}
	got := render.RenderTable(rows)
	header := strings.SplitN(got, "\n", 2)[0]
	if !strings.Contains(header, "| a |") || !strings.Contains(header, "| b |") || !strings.Contains(header, "| c |") {
		t.Fatalf("header missing expected columns: %q", header)
	}
	if strings.Index(header, "| a |") > strings.Index(header, "| b |") {
		t.Fatalf("columns not alphabetically ordered: %q", header)
	}
}

func TestRenderKV_TitleAndRecord(t *testing.T) {
	got := render.RenderKV("Server Info", []map[string]any{{"version": "1.2.3", "transport": "http"}})
	if !strings.HasPrefix(got, "# Server Info") {
		t.Fatalf("missing title heading: %q", got)
	}
	if !strings.Contains(got, "## Record 1") {
		t.Fatalf("missing Record 1 heading: %q", got)
	}
	if !strings.Contains(got, "version: 1.2.3") || !strings.Contains(got, "transport: http") {
		t.Fatalf("missing key/value lines: %q", got)
	}
}

func TestRenderKV_Empty(t *testing.T) {
	got := render.RenderKV("Empty", nil)
	if !strings.Contains(got, "no results") {
		t.Fatalf("expected 'no results' message: %q", got)
	}
}

// TestRenderTable_UnwrapsNullableWrappers covers the literal/expression-column
// regression: cells whose value is a pointer-to-nullable wrapper used to
// render as `&{ok true}`.  Unwrap should yield the scalar.
func TestRenderTable_UnwrapsNullableWrappers(t *testing.T) {
	rows := []map[string]any{
		{
			"status": &sql.NullString{String: "ok", Valid: true},
			"n":      &sql.NullInt64{Int64: 1, Valid: true},
		},
	}
	got := render.RenderTable(rows)
	if strings.Contains(got, "&{") {
		t.Fatalf("unwrap failed; got raw Go format %q", got)
	}
	if !strings.Contains(got, "| ok |") || !strings.Contains(got, "| 1 |") {
		t.Fatalf("expected unwrapped scalars; got %q", got)
	}
}

func TestRenderKV_UnwrapsNullableWrappers(t *testing.T) {
	rec := []map[string]any{
		{
			"status":  sql.NullString{String: "ok", Valid: true},
			"enabled": sql.NullBool{Bool: true, Valid: true},
		},
	}
	got := render.RenderKV("Test", rec)
	if strings.Contains(got, "&{") || strings.Contains(got, "{ok true}") {
		t.Fatalf("unwrap failed; got raw Go format %q", got)
	}
	if !strings.Contains(got, "status: ok") {
		t.Fatalf("expected 'status: ok'; got %q", got)
	}
	if !strings.Contains(got, "enabled: true") {
		t.Fatalf("expected 'enabled: true'; got %q", got)
	}
}

// TestRender_InvalidNullableRendersAsEmpty: invalid (Valid==false) wrappers
// collapse to an empty cell, matching the reverse-proxy backend's
// zero-value substitution.
func TestRender_InvalidNullableRendersAsEmpty(t *testing.T) {
	rows := []map[string]any{
		{"col": sql.NullString{Valid: false}},
	}
	got := render.RenderTable(rows)
	if strings.Contains(got, "&{") || strings.Contains(got, "false") {
		t.Fatalf("invalid wrapper should render as empty; got %q", got)
	}
	if !strings.Contains(got, "|  |") {
		t.Fatalf("expected empty cell; got %q", got)
	}
}
