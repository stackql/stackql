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

// Issue #661 fix 2: nullable wrappers (and pointers to them) must render as
// scalars, not as Go default-format struct text like "&{ok true}".
func TestRenderTable_UnwrapsNullableWrappers(t *testing.T) {
	rows := []map[string]any{{
		"s": &sql.NullString{String: "ok", Valid: true},
		"b": &sql.NullBool{Bool: true, Valid: true},
	}}
	got := render.RenderTable(rows)
	if strings.Contains(got, "&{") {
		t.Errorf("table should not contain Go wrapper text: %q", got)
	}
	if !strings.Contains(got, "| ok |") {
		t.Errorf("expected unwrapped string value, got %q", got)
	}
	if !strings.Contains(got, "| true |") {
		t.Errorf("expected unwrapped bool value, got %q", got)
	}
}

func TestRenderKV_UnwrapsNullableWrappers(t *testing.T) {
	rec := []map[string]any{{
		"s": sql.NullString{String: "ok", Valid: true},
		"b": &sql.NullBool{Bool: false, Valid: true},
	}}
	got := render.RenderKV("Sample", rec)
	if strings.Contains(got, "&{") || strings.Contains(got, "{ok") {
		t.Errorf("kv should not contain Go wrapper text: %q", got)
	}
	if !strings.Contains(got, "s: ok") {
		t.Errorf("expected unwrapped string line, got %q", got)
	}
	if !strings.Contains(got, "b: false") {
		t.Errorf("expected unwrapped bool line, got %q", got)
	}
}

func TestRender_InvalidNullableRendersAsEmpty(t *testing.T) {
	rows := []map[string]any{{
		"s": sql.NullString{String: "ignored", Valid: false},
	}}
	got := render.RenderTable(rows)
	if strings.Contains(got, "ignored") {
		t.Errorf("invalid Nullable should not surface payload, got %q", got)
	}
	if !strings.Contains(got, "|  |") {
		t.Errorf("expected empty cell for invalid Nullable, got %q", got)
	}
}
