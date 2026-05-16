package mcp_server //nolint:testpackage,revive // exercise unexported formatter

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestFormatToolResult_PrefersStructuredContent(t *testing.T) {
	res := &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "# rendered markdown\n\nfoo: bar\n"}},
		StructuredContent: map[string]any{
			"version":      "1.2.3",
			"is_read_only": false,
		},
	}
	out, err := formatToolResult("server_info", res)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"version":"1.2.3"`) || !strings.Contains(out, `"is_read_only":false`) {
		t.Errorf("expected JSON serialisation of structured content, got %q", out)
	}
	if strings.Contains(out, "rendered markdown") {
		t.Errorf("rendered text should not leak into stdout: %q", out)
	}
}

func TestFormatToolResult_FallsBackToTextWhenNoStructured(t *testing.T) {
	res := &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "plain output"}},
	}
	out, err := formatToolResult("anything", res)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "plain output" {
		t.Errorf("expected fallback text, got %q", out)
	}
}

func TestFormatToolResult_IsErrorReturnsErrorWithTextPayload(t *testing.T) {
	res := &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: "tool 'run_mutation_query' refused: server is read-only"}},
	}
	_, err := formatToolResult("run_mutation_query", res)
	if err == nil {
		t.Fatal("expected error for IsError result")
	}
	if !strings.Contains(err.Error(), "read-only") {
		t.Errorf("expected refusal text in error, got %q", err.Error())
	}
}

func TestFormatToolResult_NilResult(t *testing.T) {
	_, err := formatToolResult("anything", nil)
	if err == nil {
		t.Fatal("expected error for nil result")
	}
}

func TestFormatToolResult_StructuredContentAsArray(t *testing.T) {
	// SDK may surface structured content as either a typed value (server side)
	// or a generic decoded JSON value (client side). Verify both shape families
	// round-trip cleanly.
	res := &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ignored"}},
		StructuredContent: map[string]any{
			"rows": []any{
				map[string]any{"name": "google"},
				map[string]any{"name": "aws"},
			},
		},
	}
	out, err := formatToolResult("list_providers", res)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{`"rows":`, `"name":"google"`, `"name":"aws"`} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got %q", want, out)
		}
	}
}
