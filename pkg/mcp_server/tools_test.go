package mcp_server //nolint:testpackage,revive // exercise internal wiring

import (
	"context"
	"encoding/json"
	"os"
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
	listRegistryOut  []map[string]any
	pullProviderOut  map[string]any
	reloadCredsOut   dto.CredentialsReloadDTO

	// Capture last inputs for assertions
	lastHierarchy   dto.HierarchyInput
	lastQueryJSON   dto.QueryJSONInput
	lastExecQuery   string
	lastValidateSQL string
	lastRegistry    dto.RegistryInput
	lastReloadCreds dto.CredentialsReloadInput
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
	if b.runJSONOut == nil {
		// SDK validates QueryResultDTO.Rows as a JSON array; nil is rejected.
		return []map[string]any{}, nil
	}
	return b.runJSONOut, nil
}
func (b *testBackend) ListProviders(_ context.Context) ([]map[string]any, error) {
	return nilOrEmpty(b.listProvidersOut), nil
}
func (b *testBackend) ListServices(_ context.Context, h dto.HierarchyInput) ([]map[string]any, error) {
	b.lastHierarchy = h
	return nilOrEmpty(b.listServicesOut), nil
}
func (b *testBackend) ListResources(_ context.Context, h dto.HierarchyInput) ([]map[string]any, error) {
	b.lastHierarchy = h
	return nilOrEmpty(b.listResourcesOut), nil
}
func (b *testBackend) ListMethods(_ context.Context, h dto.HierarchyInput) ([]map[string]any, error) {
	b.lastHierarchy = h
	return nilOrEmpty(b.listMethodsOut), nil
}
func (b *testBackend) DescribeResource(_ context.Context, h dto.HierarchyInput) ([]map[string]any, error) {
	b.lastHierarchy = h
	return nilOrEmpty(b.describeRsrcOut), nil
}
func (b *testBackend) DescribeMethod(_ context.Context, h dto.HierarchyInput) ([]map[string]any, error) {
	b.lastHierarchy = h
	return nilOrEmpty(b.describeMethOut), nil
}
func (b *testBackend) ListRegistry(_ context.Context, in dto.RegistryInput) ([]map[string]any, error) {
	b.lastRegistry = in
	return nilOrEmpty(b.listRegistryOut), nil
}
func (b *testBackend) PullProvider(_ context.Context, in dto.RegistryInput) (map[string]any, error) {
	b.lastRegistry = in
	if b.pullProviderOut == nil {
		return map[string]any{}, nil
	}
	return b.pullProviderOut, nil
}
func (b *testBackend) ReloadCredentials(
	_ context.Context,
	in dto.CredentialsReloadInput,
) (dto.CredentialsReloadDTO, error) {
	b.lastReloadCreds = in
	out := b.reloadCredsOut
	if out.Providers == nil {
		out.Providers = []dto.ProviderCredentialStatusDTO{}
	}
	return out, nil
}

// nilOrEmpty ensures we return a non-nil slice so the SDK's schema validation
// accepts it as "array" rather than "null".
func nilOrEmpty(v []map[string]any) []map[string]any {
	if v == nil {
		return []map[string]any{}
	}
	return v
}

// connectInProcess wires a freshly-built server to an in-memory client and returns the client session.
// The client does NOT advertise elicitation, which mirrors the robot test harness.
func connectInProcess(t *testing.T, cfg *Config, backend Backend) *mcp.ClientSession {
	return connectInProcessWith(t, cfg, backend, nil)
}

// connectInProcessWith is the workhorse helper: callers may pass an elicitation
// handler to opt the client into elicitation-capable behaviour.  The audit
// sink is forced to a nop so tests don't pollute cwd with log files.
func connectInProcessWith(
	t *testing.T,
	cfg *Config,
	backend Backend,
	elicit func(context.Context, *mcp.ElicitRequest) (*mcp.ElicitResult, error),
) *mcp.ClientSession {
	t.Helper()
	if cfg == nil {
		cfg = DefaultConfig()
	}
	// Disable audit in unit tests by default so we don't drop log files
	// next to the test binary.  Tests that exercise audit set this back.
	cfg.Server.Audit.Disabled = true

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
	var clientOpts *mcp.ClientOptions
	if elicit != nil {
		clientOpts = &mcp.ClientOptions{ElicitationHandler: elicit}
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0"}, clientOpts)
	cs, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

// fullAccessConfig returns a DefaultConfig with mode=full_access, used by
// positive-path tests for mutation/lifecycle tools that would otherwise be
// gated by the new default mode (safe).
func fullAccessConfig() *Config {
	cfg := DefaultConfig()
	cfg.Server.Mode = "full_access"
	return cfg
}

// readOnlyConfig returns a DefaultConfig with mode=read_only.
func readOnlyConfig() *Config {
	cfg := DefaultConfig()
	cfg.Server.Mode = "read_only"
	return cfg
}

// deleteSafeConfig returns a DefaultConfig with mode=delete_safe.
func deleteSafeConfig() *Config {
	cfg := DefaultConfig()
	cfg.Server.Mode = "delete_safe"
	return cfg
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
	be := &testBackend{}
	cs := connectInProcess(t, readOnlyConfig(), be)

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
	if got := firstText(t, res); !strings.Contains(got, "read_only") {
		t.Errorf("expected refusal message to mention read_only mode, got %q", got)
	}
}

func TestTool_RunLifecycleOperation_PositiveAndReadOnly(t *testing.T) {
	// Positive path under full_access (no elicitation needed).
	be := &testBackend{execOut: map[string]any{"messages": []string{"ok"}, "timestamp": "now"}}
	cs := connectInProcess(t, fullAccessConfig(), be)
	callTool(t, cs, "run_lifecycle_operation", map[string]any{"sql": "EXEC x.y.z @a='1'"})
	if be.lastExecQuery != "EXEC x.y.z @a='1'" {
		t.Errorf("exec query not forwarded: %q", be.lastExecQuery)
	}

	// Refusal path under read_only.
	be2 := &testBackend{}
	cs2 := connectInProcess(t, readOnlyConfig(), be2)
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
	for _, denied := range []string{"list_providers", "run_select_query", "run_mutation_query", "list_registry", "pull_provider"} {
		if names[denied] {
			t.Errorf("%s should be filtered out, tools: %v", denied, names)
		}
	}
}

func TestTools_AnnotationsDerivedFromGate(t *testing.T) {
	cs := connectInProcess(t, DefaultConfig(), &testBackend{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tools, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	byName := map[string]*mcp.Tool{}
	for _, tool := range tools.Tools {
		byName[tool.Name] = tool
	}
	annotationsFor := func(name string) *mcp.ToolAnnotations {
		tool, ok := byName[name]
		if !ok {
			t.Fatalf("tool %s not published", name)
		}
		return tool.Annotations
	}
	// Statically select-classified tools claim read-only.
	for _, name := range []string{
		"server_info", "list_providers", "list_services", "list_resources",
		"list_methods", "describe_resource", "describe_method",
		"validate_select_query", "list_registry",
	} {
		if a := annotationsFor(name); a == nil || !a.ReadOnlyHint {
			t.Errorf("%s should carry ReadOnlyHint, got %+v", name, a)
		}
	}
	// SQL-carrying tools make no read-only claim; effect depends on the SQL.
	if a := annotationsFor("run_select_query"); a != nil && a.ReadOnlyHint {
		t.Errorf("run_select_query must not claim read-only, got %+v", a)
	}
	// Mutation/lifecycle tools are explicitly destructive.
	for _, name := range []string{"run_mutation_query", "run_lifecycle_operation"} {
		a := annotationsFor(name)
		if a == nil || a.DestructiveHint == nil || !*a.DestructiveHint {
			t.Errorf("%s should carry an explicit true DestructiveHint, got %+v", name, a)
		}
	}
	// Local-state writers: not read-only, idempotent, non-destructive.
	for _, name := range []string{"pull_provider", "reload_credentials"} {
		a := annotationsFor(name)
		if a == nil || a.ReadOnlyHint || !a.IdempotentHint || a.DestructiveHint == nil || *a.DestructiveHint {
			t.Errorf("%s should be non-read-only, idempotent, non-destructive; got %+v", name, a)
		}
	}
}

func TestTool_ListRegistry_RendersTableAndForwardsProvider(t *testing.T) {
	be := &testBackend{listRegistryOut: []map[string]any{
		{"provider": "google", "version": "v1"},
		{"provider": "aws", "version": "v2"},
	}}
	cs := connectInProcess(t, DefaultConfig(), be)

	res := callTool(t, cs, "list_registry", map[string]any{"provider": "google"})
	text := firstText(t, res)
	if !strings.Contains(text, "| provider |") || !strings.Contains(text, "| version |") {
		t.Errorf("missing table header: %q", text)
	}
	if !strings.Contains(text, "| google |") {
		t.Errorf("missing provider cell: %q", text)
	}
	if be.lastRegistry.Provider != "google" {
		t.Errorf("provider not forwarded: %+v", be.lastRegistry)
	}
}

func TestTool_ListRegistry_AllowedInReadOnly(t *testing.T) {
	be := &testBackend{listRegistryOut: []map[string]any{{"provider": "x", "version": "v0"}}}
	cs := connectInProcess(t, readOnlyConfig(), be)
	callTool(t, cs, "list_registry", map[string]any{})
	if be.lastRegistry.Provider != "" {
		t.Errorf("read_only should still reach backend; got %+v", be.lastRegistry)
	}
}

func TestTool_PullProvider_RendersKVAndForwardsArgs(t *testing.T) {
	be := &testBackend{pullProviderOut: map[string]any{"timestamp": "now", "messages": []string{"ok"}}}
	cs := connectInProcess(t, DefaultConfig(), be)

	res := callTool(t, cs, "pull_provider", map[string]any{"provider": "google", "version": "v0.1.2"})
	text := firstText(t, res)
	if !strings.Contains(text, "# Pull Result") {
		t.Errorf("expected KV title, got %q", text)
	}
	if !strings.Contains(text, "timestamp: now") {
		t.Errorf("expected timestamp line, got %q", text)
	}
	if be.lastRegistry.Provider != "google" || be.lastRegistry.Version != "v0.1.2" {
		t.Errorf("registry args not forwarded: %+v", be.lastRegistry)
	}
}

func TestTool_PullProvider_AllowedInReadOnly(t *testing.T) {
	be := &testBackend{pullProviderOut: map[string]any{"timestamp": "now"}}
	cs := connectInProcess(t, readOnlyConfig(), be)
	callTool(t, cs, "pull_provider", map[string]any{"provider": "google"})
	if be.lastRegistry.Provider != "google" {
		t.Errorf("read_only should still allow pull_provider; got %+v", be.lastRegistry)
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

// --- Mode contract: client without elicitation support ---

// callExpectingError invokes a tool and asserts the result carries IsError=true
// with a refusal message containing the given substring.  Used for the no-
// elicitation fallback paths (safe / delete_safe with a non-elicitation client).
func callExpectingError(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any, msgFragment string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(%s): transport-level err %v", name, err)
	}
	if !res.IsError {
		t.Errorf("CallTool(%s): expected IsError=true, got %+v", name, res)
		return
	}
	if got := firstText(t, res); !strings.Contains(got, msgFragment) {
		t.Errorf("CallTool(%s): refusal message %q does not contain %q", name, got, msgFragment)
	}
}

func TestMode_ReadOnly_RefusesAllMutationsAndLifecycle(t *testing.T) {
	be := &testBackend{}
	cs := connectInProcess(t, readOnlyConfig(), be)

	// SELECT proceeds.
	callTool(t, cs, "run_select_query", map[string]any{"sql": "select 1"})

	// Each non-select tool is refused with the read_only message.
	callExpectingError(t, cs, "run_mutation_query",
		map[string]any{"sql": "insert into t values (1)"}, "read_only")
	callExpectingError(t, cs, "run_mutation_query",
		map[string]any{"sql": "delete from t"}, "read_only")
	callExpectingError(t, cs, "run_lifecycle_operation",
		map[string]any{"sql": "EXEC a.b.c"}, "read_only")
	if be.lastExecQuery != "" {
		t.Errorf("backend should not have been called: %q", be.lastExecQuery)
	}
}

func TestMode_Safe_RefusesMutationsWithoutElicitation(t *testing.T) {
	be := &testBackend{}
	cs := connectInProcess(t, DefaultConfig(), be) // default is safe

	callTool(t, cs, "run_select_query", map[string]any{"sql": "select 1"})

	callExpectingError(t, cs, "run_mutation_query",
		map[string]any{"sql": "insert into t values (1)"}, "does not support elicitation")
	callExpectingError(t, cs, "run_mutation_query",
		map[string]any{"sql": "delete from t"}, "does not support elicitation")
	callExpectingError(t, cs, "run_lifecycle_operation",
		map[string]any{"sql": "EXEC a.b.c"}, "does not support elicitation")
	if be.lastExecQuery != "" {
		t.Errorf("backend should not have been called: %q", be.lastExecQuery)
	}
}

func TestMode_DeleteSafe_AllowsCreateRefusesDeleteAndLifecycle(t *testing.T) {
	be := &testBackend{execOut: map[string]any{"timestamp": "now"}}
	cs := connectInProcess(t, deleteSafeConfig(), be)

	// SELECT and INSERT/UPDATE proceed.
	callTool(t, cs, "run_select_query", map[string]any{"sql": "select 1"})
	callTool(t, cs, "run_mutation_query", map[string]any{"sql": "insert into t values (1)"})
	if be.lastExecQuery != "insert into t values (1)" {
		t.Errorf("insert should reach the backend, got %q", be.lastExecQuery)
	}

	// DELETE and EXEC refused (no elicitation).
	be.lastExecQuery = ""
	callExpectingError(t, cs, "run_mutation_query",
		map[string]any{"sql": "delete from t"}, "delete_safe")
	callExpectingError(t, cs, "run_lifecycle_operation",
		map[string]any{"sql": "EXEC a.b.c"}, "delete_safe")
	if be.lastExecQuery != "" {
		t.Errorf("delete/lifecycle should not have reached the backend, got %q", be.lastExecQuery)
	}
}

func TestMode_FullAccess_AllowsEverything(t *testing.T) {
	be := &testBackend{execOut: map[string]any{"timestamp": "now"}}
	cs := connectInProcess(t, fullAccessConfig(), be)

	callTool(t, cs, "run_select_query", map[string]any{"sql": "select 1"})
	callTool(t, cs, "run_mutation_query", map[string]any{"sql": "insert into t values (1)"})
	callTool(t, cs, "run_mutation_query", map[string]any{"sql": "delete from t"})
	callTool(t, cs, "run_lifecycle_operation", map[string]any{"sql": "EXEC a.b.c"})
	if be.lastExecQuery != "EXEC a.b.c" {
		t.Errorf("lifecycle should have reached backend last; got %q", be.lastExecQuery)
	}
}

// --- Mode contract: client WITH elicitation support ---

func acceptingElicit(_ context.Context, _ *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
	return &mcp.ElicitResult{Action: "accept"}, nil
}

func decliningElicit(_ context.Context, _ *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
	return &mcp.ElicitResult{Action: "decline"}, nil
}

func cancellingElicit(_ context.Context, _ *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
	return &mcp.ElicitResult{Action: "cancel"}, nil
}

func TestMode_Safe_ElicitationAcceptProceeds(t *testing.T) {
	be := &testBackend{execOut: map[string]any{"timestamp": "now"}}
	cs := connectInProcessWith(t, DefaultConfig(), be, acceptingElicit)
	callTool(t, cs, "run_mutation_query", map[string]any{"sql": "delete from t"})
	if be.lastExecQuery != "delete from t" {
		t.Errorf("after accept, backend should run mutation; got %q", be.lastExecQuery)
	}
}

func TestMode_Safe_ElicitationDeclineRefuses(t *testing.T) {
	be := &testBackend{}
	cs := connectInProcessWith(t, DefaultConfig(), be, decliningElicit)
	callExpectingError(t, cs, "run_mutation_query",
		map[string]any{"sql": "delete from t"}, "declined approval")
	if be.lastExecQuery != "" {
		t.Errorf("after decline, backend should not run: %q", be.lastExecQuery)
	}
}

func TestMode_Safe_ElicitationCancelRefuses(t *testing.T) {
	be := &testBackend{}
	cs := connectInProcessWith(t, DefaultConfig(), be, cancellingElicit)
	callExpectingError(t, cs, "run_mutation_query",
		map[string]any{"sql": "delete from t"}, "dismissed")
	if be.lastExecQuery != "" {
		t.Errorf("after cancel, backend should not run: %q", be.lastExecQuery)
	}
}

func TestMode_DeleteSafe_ElicitationAcceptAllowsDelete(t *testing.T) {
	be := &testBackend{execOut: map[string]any{"timestamp": "now"}}
	cs := connectInProcessWith(t, deleteSafeConfig(), be, acceptingElicit)
	callTool(t, cs, "run_mutation_query", map[string]any{"sql": "delete from t"})
	if be.lastExecQuery != "delete from t" {
		t.Errorf("after accept, delete should reach backend: %q", be.lastExecQuery)
	}
}

// --- ServerInfo surfaces mode + is_read_only ---

func TestServerInfo_SurfacesMode(t *testing.T) {
	be := &testBackend{
		serverInfoOut: dto.ServerInfoOutput{
			Mode:     "delete_safe",
			ReadOnly: false,
		},
	}
	cs := connectInProcess(t, deleteSafeConfig(), be)
	res := callTool(t, cs, "server_info", map[string]any{})
	out := structuredAs[dto.ServerInfoDTO](t, res)
	if out.Mode != "delete_safe" {
		t.Errorf("expected mode=delete_safe, got %q", out.Mode)
	}
	if out.ReadOnly {
		t.Errorf("expected is_read_only=false")
	}
}

// --- Audit ---

//nolint:gocognit // end-to-end audit test threads many setup steps
func TestAudit_RecordsAllToolCalls(t *testing.T) {
	// Swap the sink-init by constructing the server directly and replacing
	// the gate's sink via the unexported `auditSink` field.  Simpler: build
	// the server with audit disabled, then re-register tools by hand.  Even
	// simpler: route the in-process tests through a config that points at a
	// temp directory and read back the file.
	dir := t.TempDir()
	logPath := dir + "/audit.log"

	cfg := fullAccessConfig()
	cfg.Server.Audit.Disabled = false
	cfg.Server.Audit.File.Path = logPath

	be := &testBackend{execOut: map[string]any{"timestamp": "now"}}

	// Build the server manually so we can override the audit-disabled flag
	// that connectInProcess sets.  We replicate connectInProcessWith's body
	// but skip the audit-Disabled mutation.
	mcpSrv, err := newMCPServer(cfg, be, nil)
	if err != nil {
		t.Fatalf("newMCPServer: %v", err)
	}
	t.Cleanup(func() {
		if closer, ok := mcpSrv.(*simpleMCPServer); ok {
			_ = closer.auditSink.Close()
		}
	})
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

	callTool(t, cs, "run_select_query", map[string]any{"sql": "select 1"})
	callTool(t, cs, "run_mutation_query", map[string]any{"sql": "insert into t values (1)"})

	// Close the sink to flush.
	if closer, ok := mcpSrv.(*simpleMCPServer); ok {
		if err := closer.auditSink.Close(); err != nil {
			t.Fatalf("close sink: %v", err)
		}
	}

	data, err := readAllJSONLines(t, logPath)
	if err != nil {
		t.Fatalf("read audit log: %v", err)
	}
	if len(data) < 2 {
		t.Fatalf("expected at least 2 audit lines, got %d", len(data))
	}
	hasSelect, hasMutation := false, false
	for _, line := range data {
		var ev map[string]any
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("unmarshal audit line: %v", err)
		}
		switch ev["tool"] {
		case "run_select_query":
			hasSelect = true
			if ev["decision"] != "allow" {
				t.Errorf("select should be allow, got %v", ev["decision"])
			}
		case "run_mutation_query":
			hasMutation = true
			if ev["decision"] != "allow" {
				t.Errorf("full_access mutation should be allow, got %v", ev["decision"])
			}
		}
	}
	if !hasSelect || !hasMutation {
		t.Errorf("missing expected events: select=%v mutation=%v", hasSelect, hasMutation)
	}
}

// readAllJSONLines is a small helper used only by TestAudit_RecordsAllToolCalls.
func readAllJSONLines(t *testing.T, path string) ([]string, error) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		out = append(out, line)
	}
	return out, nil
}
