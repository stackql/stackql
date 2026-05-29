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

// TestRenderTable_UnwrapsNullableWrappers asserts that database/sql nullable
// wrappers (and pointers to them) render as the underlying scalar rather than
// the Go struct form (eg `&{ok true}`).  Repro from the bug report: SELECT 1
// as n, 'ok' as status used to render `| &{ok true} | &{1 true} |`.
func TestRenderTable_UnwrapsNullableWrappers(t *testing.T) {
	rows := []map[string]any{{
		"n":      &sql.NullInt64{Int64: 1, Valid: true},
		"status": &sql.NullString{String: "ok", Valid: true},
	}}
	got := render.RenderTable(rows)
	if strings.Contains(got, "&{") {
		t.Fatalf("expected wrappers to be unwrapped, got %q", got)
	}
	if !strings.Contains(got, "| ok ") || !strings.Contains(got, "| 1 ") {
		t.Fatalf("expected unwrapped scalars in cells, got %q", got)
	}
}

// TestRenderKV_UnwrapsNullableWrappers exercises the parallel path in
// RenderKV.  ServerInfo uses RenderKV, and other tools route here too.
func TestRenderKV_UnwrapsNullableWrappers(t *testing.T) {
	got := render.RenderKV("Pull Result", []map[string]any{{
		"messages":  &sql.NullString{String: "ok", Valid: true},
		"timestamp": sql.NullString{String: "2026-05-29", Valid: true},
		"flag":      sql.NullBool{Bool: true, Valid: true},
	}})
	if strings.Contains(got, "&{") {
		t.Fatalf("expected wrappers to be unwrapped, got %q", got)
	}
	for _, want := range []string{"messages: ok", "timestamp: 2026-05-29", "flag: true"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got %q", want, got)
		}
	}
}

// TestRender_InvalidNullableRendersAsEmpty asserts an invalid wrapper renders
// as the empty string, matching the reverse-proxy backend's zero-value
// substitution (z.String / z.Bool / z.Int64 on invalid wrappers).  Without
// the substitution the cell would render as `false` / `0` / a wrapper struct
// form and surface false information to the client.
func TestRender_InvalidNullableRendersAsEmpty(t *testing.T) {
	got := render.RenderKV("Sparse", []map[string]any{{
		"absent": &sql.NullString{Valid: false},
	}})
	if !strings.Contains(got, "absent: \n") {
		t.Fatalf("expected invalid wrapper to render as empty, got %q", got)
	}
}
