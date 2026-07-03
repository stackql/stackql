package mcp_server //nolint:testpackage,revive // exercise unexported formatter

import (
	"encoding/json"
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
	out, err := formatToolResult("server_info", res, false)
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

func TestFormatToolResult_PreferTextReturnsTextContent(t *testing.T) {
	// `"prefer_text": true` in the client config must surface the rendered
	// text content even when a structured payload is present (issue #669).
	res := &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: `{"rows":[{"name":"google"}]}`}},
		StructuredContent: map[string]any{
			"rows": []any{map[string]any{"name": "google", "extra": "structured-only"}},
		},
	}
	out, err := formatToolResult("list_providers", res, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != `{"rows":[{"name":"google"}]}` {
		t.Errorf("expected text content verbatim, got %q", out)
	}
}

func TestFormatToolResult_FallsBackToTextWhenNoStructured(t *testing.T) {
	res := &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "plain output"}},
	}
	out, err := formatToolResult("anything", res, false)
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
	_, err := formatToolResult("run_mutation_query", res, false)
	if err == nil {
		t.Fatal("expected error for IsError result")
	}
	if !strings.Contains(err.Error(), "read-only") {
		t.Errorf("expected refusal text in error, got %q", err.Error())
	}
}

func TestFormatToolResult_NilResult(t *testing.T) {
	_, err := formatToolResult("anything", nil, false)
	if err == nil {
		t.Fatal("expected error for nil result")
	}
}

func TestFormatToolResult_EmbeddedJSONStringInValueRoundtripsCleanly(t *testing.T) {
	// describe_method returns rows whose `shape` field is itself a JSON-encoded
	// string.  Robot scenarios feed our stdout to json.loads, so anything that
	// gets the escaping wrong (eg interpolating into a single-quoted Python
	// source literal) will explode.  Verify formatToolResult's output is
	// itself round-trippable through Go's json.Unmarshal -- the strict-mode
	// equivalent of what Robot does in its Parse MCP JSON Output helper.
	res := &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ignored markdown"}},
		StructuredContent: map[string]any{
			"rows": []any{
				map[string]any{
					"name":  "routingConfig",
					"shape": `{"description":"A routing config.","type":"object"}`,
				},
			},
		},
	}
	out, err := formatToolResult("describe_method", res, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("client output is not valid JSON: %v\noutput: %s", err, out)
	}
	rows, ok := decoded["rows"].([]any)
	if !ok || len(rows) != 1 {
		t.Fatalf("expected one row, got %#v", decoded["rows"])
	}
	first, ok := rows[0].(map[string]any)
	if !ok {
		t.Fatalf("row is not a map: %#v", rows[0])
	}
	if first["shape"] != `{"description":"A routing config.","type":"object"}` {
		t.Errorf("shape did not survive round-trip: %#v", first["shape"])
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
	out, err := formatToolResult("list_providers", res, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{`"rows":`, `"name":"google"`, `"name":"aws"`} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got %q", want, out)
		}
	}
}
