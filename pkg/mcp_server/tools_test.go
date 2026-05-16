package mcp_server //nolint:testpackage,revive // exercise internal wiring

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stackql/stackql/pkg/mcp_server/dto"
)

// testBackend is a controllable Backend used to assert tool wiring end-to-end.
type testBackend struct {
	serverInfoOut    dto.ServerInfoOutput
	listProvidersOut []map[string]any
	listServicesOut  []map[string]any
	listResourcesOut []map[string]any
	listMethodsOut   []map[string]any
	describeRsrcOut  []map[string]any
	describeMethOut  []map[string]any
	runJSONOut       []map[string]any
	validateOut      []map[string]any
	validateErr      error
	execOut          map[string]any

	// Capture last inputs for assertions
	lastHierarchy   dto.HierarchyInput
	lastQueryJSON   dto.QueryJSONInput
	lastExecQuery   string
	lastValidateSQL string
}

func (b *testBackend) Ping(_ context.Context) error { return nil }
func (b *testBackend) Close() error                 { return nil }

func (b *testBackend) ServerInfo(_ context.Context, _ any) (dto.ServerInfoOutput, error) {
	return b.serverInfoOut, nil
}
func (b *testBackend) ExecQuery(_ context.Context, q string) (map[string]any, error) {
	b.lastExecQuery = q
	return b.execOut, nil
}
func (b *testBackend) ValidateQuery(_ context.Context, q string) ([]map[string]any, error) {
	b.lastValidateSQL = q
	return b.validateOut, b.validateErr
}
func (b *testBackend) RunQueryJSON(_ context.Context, in dto.QueryJSONInput) ([]map[string]any, error) {
	b.lastQueryJSON = in
	return b.runJSONOut, nil
}
func (b *testBackend) ListProviders(_ context.Context) ([]map[string]any, error) {
	return b.listProvidersOut, nil
}
func (b *testBackend) ListServices(_ context.Context, h dto.HierarchyInput) ([]map[string]any, error) {
	b.lastHierarchy = h
	return b.listServicesOut, nil
}
func (b *testBackend) ListResources(_ context.Context, h dto.HierarchyInput) ([]map[string]any, error) {
	b.lastHierarchy = h
	return b.listResourcesOut, nil
}
func (b *testBackend) ListMethods(_ context.Context, h dto.HierarchyInput) ([]map[string]any, error) {
	b.lastHierarchy = h
	return b.listMethodsOut, nil
}
func (b *testBackend) DescribeResource(_ context.Context, h dto.HierarchyInput) ([]map[string]any, error) {
	b.lastHierarchy = h
	return b.describeRsrcOut, nil
}
func (b *testBackend) DescribeMethod(_ context.Context, h dto.HierarchyInput) ([]map[string]any, error) {
	b.lastHierarchy = h
	return b.describeMethOut, nil
}

// connectInProcess wires a freshly-built server to an in-memory client and returns the client session.
func connectInProcess(t *testing.T, cfg *Config, backend Backend) *mcp.ClientSession {
	t.Helper()
	mcpSrv, err := newMCPServer(cfg, backend, nil)
	if err != nil {
		t.Fatalf("newMCPServer: %v", err)
	}
	rawServer := mcpSrv.(*simpleMCPServer).server

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	t1, t2 := mcp.NewInMemoryTransports()
	if _, err := rawServer.Connect(ctx, t1, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0"}, nil)
	cs, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func callTool(t *testing.T, cs *mcp.ClientSession, name string, args any) *mcp.CallToolResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(%s): %v", name, err)
	}
	return res
}

func firstText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatalf("no content blocks")
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("first content block is not TextContent: %T", res.Content[0])
	}
	return tc.Text
}

func structuredAs[T any](t *testing.T, res *mcp.CallToolResult) T {
	t.Helper()
	var out T
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured: %v", err)
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal structured into %T: %v", out, err)
	}
	return out
}

func TestTool_ServerInfo_RendersKVAndStructured(t *testing.T) {
	be := &testBackend{
		serverInfoOut: dto.ServerInfoOutput{
			Version: "1.2.3", Commit: "abc1234", Platform: "linux/amd64",
			Transport: "http", SQLBackend: "sqlite3",
			ProviderRegistry: "https://registry.example", ReadOnly: true,
		},
	}
	cs := connectInProcess(t, DefaultConfig(), be)

	res := callTool(t, cs, "server_info", map[string]any{})
	text := firstText(t, res)
	for _, want := range []string{"# Server Info", "version: 1.2.3", "sql_backend: sqlite3", "provider_registry: https://registry.example", "is_read_only: true"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected text to contain %q, got %q", want, text)
		}
	}

	out := structuredAs[dto.ServerInfoDTO](t, res)
	if out.Version != "1.2.3" || out.SQLBackend != "sqlite3" || !out.ReadOnly {
		t.Errorf("structured payload mismatch: %+v", out)
	}
}

func TestTool_ListProviders_RendersTable(t *testing.T) {
	be := &testBackend{listProvidersOut: []map[string]any{
		{"name": "google", "version": "v1"},
		{"name": "aws", "version": "v2"},
	}}
	cs := connectInProcess(t, DefaultConfig(), be)

	res := callTool(t, cs, "list_providers", map[string]any{})
	text := firstText(t, res)
	if !strings.Contains(text, "| name |") || !strings.Contains(text, "| version |") {
		t.Errorf("missing table header: %q", text)
	}
	if !strings.Contains(text, "| google |") || !strings.Contains(text, "| aws |") {
		t.Errorf("missing data rows: %q", text)
	}
	out := structuredAs[dto.QueryResultDTO](t, res)
	if len(out.Rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(out.Rows))
	}
}

func TestTool_ListServices_ForwardsHierarchy(t *testing.T) {
	be := &testBackend{listServicesOut: []map[string]any{{"name": "compute"}}}
	cs := connectInProcess(t, DefaultConfig(), be)

	callTool(t, cs, "list_services", map[string]any{"provider": "google"})
	if be.lastHierarchy.Provider != "google" {
		t.Errorf("provider not forwarded: %+v", be.lastHierarchy)
	}
}

func TestTool_DescribeResource_UsesKVRenderer(t *testing.T) {
	be := &testBackend{describeRsrcOut: []map[string]any{{"name": "id", "type": "string"}}}
	cs := connectInProcess(t, DefaultConfig(), be)

	res := callTool(t, cs, "describe_resource", map[string]any{
		"provider": "google", "service": "compute", "resource": "networks",
	})
	text := firstText(t, res)
	if !strings.Contains(text, "# Resource") {
		t.Errorf("expected KV title, got %q", text)
	}
	if be.lastHierarchy.Resource != "networks" {
		t.Errorf("hierarchy not forwarded: %+v", be.lastHierarchy)
	}
}

func TestTool_DescribeMethod_RequiresFourSegments(t *testing.T) {
	be := &testBackend{describeMethOut: []map[string]any{{"name": "project", "required": true}}}
	cs := connectInProcess(t, DefaultConfig(), be)

	const wantMethod = "select_one"
	callTool(t, cs, "describe_method", map[string]any{
		"provider": "google", "service": "compute", "resource": "networks", "method": wantMethod,
	})
	if be.lastHierarchy.Method != wantMethod {
		t.Errorf("method not forwarded: %+v", be.lastHierarchy)
	}
}

func TestTool_ValidateSelectQuery_SuccessAndFailure(t *testing.T) {
	be := &testBackend{validateOut: []map[string]any{{"plan": "ok"}}}
	cs := connectInProcess(t, DefaultConfig(), be)

	res := callTool(t, cs, "validate_select_query", map[string]any{"sql": "select 1"})
	out := structuredAs[dto.ValidationResultDTO](t, res)
	if !out.Valid {
		t.Errorf("expected valid=true, got %+v", out)
	}
	if be.lastValidateSQL != "select 1" {
		t.Errorf("sql not forwarded: %q", be.lastValidateSQL)
	}

	be.validateErr = mcpTestErr("syntax fail")
	res = callTool(t, cs, "validate_select_query", map[string]any{"sql": "bad"})
	out = structuredAs[dto.ValidationResultDTO](t, res)
	if out.Valid {
		t.Errorf("expected valid=false")
	}
	if len(out.Errors) == 0 || !strings.Contains(out.Errors[0], "syntax fail") {
		t.Errorf("expected error message, got %+v", out.Errors)
	}
}

func TestTool_RunSelectQuery_ForwardsRowLimit(t *testing.T) {
	be := &testBackend{runJSONOut: []map[string]any{{"a": 1}}}
	cs := connectInProcess(t, DefaultConfig(), be)

	callTool(t, cs, "run_select_query", map[string]any{"sql": "select 1", "row_limit": 7})
	if be.lastQueryJSON.SQL != "select 1" || be.lastQueryJSON.RowLimit != 7 {
		t.Errorf("input not forwarded: %+v", be.lastQueryJSON)
	}
}

func TestTool_RunMutation_RefusedInReadOnly(t *testing.T) {
	readOnly := true
	cfg := DefaultConfig()
	cfg.Server.IsReadOnly = &readOnly
	be := &testBackend{}
	cs := connectInProcess(t, cfg, be)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "run_mutation_query", Arguments: map[string]any{"sql": "delete from x"},
	})
	if err != nil {
		t.Fatalf("CallTool transport-level err: %v", err)
	}
	if !res.IsError {
		t.Errorf("expected tool error, got success: %+v", res)
	}
	if be.lastExecQuery != "" {
		t.Errorf("backend.ExecQuery should not have been called, got %q", be.lastExecQuery)
	}
	if got := firstText(t, res); !strings.Contains(got, "read-only") {
		t.Errorf("expected refusal message to mention read-only, got %q", got)
	}
}

func TestTool_RunLifecycleOperation_PositiveAndReadOnly(t *testing.T) {
	// Positive path on a writable server.
	be := &testBackend{execOut: map[string]any{"messages": []string{"ok"}, "timestamp": "now"}}
	cs := connectInProcess(t, DefaultConfig(), be)
	callTool(t, cs, "run_lifecycle_operation", map[string]any{"sql": "EXEC x.y.z @a='1'"})
	if be.lastExecQuery != "EXEC x.y.z @a='1'" {
		t.Errorf("exec query not forwarded: %q", be.lastExecQuery)
	}

	// Refusal path on a read-only server.
	readOnly := true
	cfg := DefaultConfig()
	cfg.Server.IsReadOnly = &readOnly
	be2 := &testBackend{}
	cs2 := connectInProcess(t, cfg, be2)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := cs2.CallTool(ctx, &mcp.CallToolParams{
		Name: "run_lifecycle_operation", Arguments: map[string]any{"sql": "EXEC x.y.z"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError {
		t.Errorf("expected refusal IsError=true")
	}
	if be2.lastExecQuery != "" {
		t.Errorf("backend should not have been called: %q", be2.lastExecQuery)
	}
}

func TestRegistration_EnabledToolsFilters(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EnabledTools = []string{"server_info"}
	be := &testBackend{}
	cs := connectInProcess(t, cfg, be)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tools, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	names := map[string]bool{}
	for _, tool := range tools.Tools {
		names[tool.Name] = true
	}
	if !names["server_info"] {
		t.Errorf("server_info should be present")
	}
	for _, denied := range []string{"list_providers", "run_select_query", "run_mutation_query"} {
		if names[denied] {
			t.Errorf("%s should be filtered out, tools: %v", denied, names)
		}
	}
}

func TestPrompt_WriteSafeSelect_RegisteredAndReturnsCanonicalText(t *testing.T) {
	cs := connectInProcess(t, DefaultConfig(), &testBackend{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := cs.GetPrompt(ctx, &mcp.GetPromptParams{Name: "write_safe_select"})
	if err != nil {
		t.Fatalf("GetPrompt: %v", err)
	}
	if len(res.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(res.Messages))
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("content not TextContent: %T", res.Messages[0].Content)
	}
	if tc.Text != ExplainerPromptWriteSafeSelectTool {
		t.Errorf("prompt text mismatch.\nwant: %q\ngot:  %q", ExplainerPromptWriteSafeSelectTool, tc.Text)
	}
}

func TestPrompt_EnabledPromptsFilters(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EnabledPrompts = []string{"some_other_prompt"}
	cs := connectInProcess(t, cfg, &testBackend{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	prompts, err := cs.ListPrompts(ctx, nil)
	if err != nil {
		t.Fatalf("ListPrompts: %v", err)
	}
	for _, p := range prompts.Prompts {
		if p.Name == "write_safe_select" {
			t.Errorf("write_safe_select should be filtered out")
		}
	}
}

type mcpTestError string

func (e mcpTestError) Error() string { return string(e) }

func mcpTestErr(s string) error { return mcpTestError(s) }
