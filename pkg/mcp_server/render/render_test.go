package render_test

import (
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
