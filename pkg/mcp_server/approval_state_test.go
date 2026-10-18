package mcp_server //nolint:testpackage,revive // exercise internal approval wiring

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Issue #827: an approval answer is honoured only with a valid RequestState
// bound to the exact tool, arguments, SQL and session of the prompt.

const currentRevisionMeta = `"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28",` +
	`"io.modelcontextprotocol/clientCapabilities":{"elicitation":{"form":{}}}}`

const currentRevisionMetaNoElicit = `"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28",` +
	`"io.modelcontextprotocol/clientCapabilities":{}}`

const currentRevisionDeleteArgs = `"name":"run_mutation_query","arguments":{"sql":"delete from t"}`

// currentRevisionRetry builds tools/call params carrying an answer and,
// when state is non-empty, the echoed requestState.
func currentRevisionRetry(meta, call, action, state string) string {
	out := `{` + meta + `,` + call + `,"inputResponses":{"` + approvalInputKey + `":{"action":"` + action + `"}}`
	if state != "" {
		out += `,"requestState":` + mustMarshal(state)
	}
	return out + `}`
}

// expectApprovalPrompt sends a first-pass 2026-07-28 delete call and
// returns the requestState of the input-required approval result.
func (c *rawClientConn) expectApprovalPrompt(id int64) string {
	c.t.Helper()
	c.send(id, "tools/call", `{`+currentRevisionMeta+`,`+currentRevisionDeleteArgs+`}`)
	var first struct {
		ResultType    string                     `json:"resultType"`
		InputRequests map[string]json.RawMessage `json:"inputRequests"`
		RequestState  string                     `json:"requestState"`
	}
	if err := json.Unmarshal(c.expectResponse(id), &first); err != nil {
		c.t.Fatalf("first result: %v", err)
	}
	if first.ResultType != "input_required" || first.InputRequests[approvalInputKey] == nil {
		c.t.Fatalf("expected an input_required result keyed %q, got %+v", approvalInputKey, first)
	}
	if first.RequestState == "" {
		c.t.Fatalf("approval prompt must carry a requestState")
	}
	return first.RequestState
}

// expectRefusal sends a tools/call and asserts a tool error containing want.
func (c *rawClientConn) expectRefusal(id int64, params, want string) {
	c.t.Helper()
	c.send(id, "tools/call", params)
	got := string(c.expectResponse(id))
	if !strings.Contains(got, `"isError":true`) || !strings.Contains(got, want) {
		c.t.Fatalf("expected a refusal containing %q, got %s", want, got)
	}
}

func TestApproval_UnsolicitedAcceptRefused_CurrentRevision(t *testing.T) {
	be := &testBackend{execOut: map[string]any{"timestamp": "now"}}
	c := newRawClientConn(t, DefaultConfig(), be)
	c.expectRefusal(1, currentRevisionRetry(currentRevisionMeta, currentRevisionDeleteArgs, "accept", ""),
		"no approval state")
	if be.execCalls != 0 {
		t.Fatalf("unsolicited accept must not execute, backend ran %d times", be.execCalls)
	}
}

// legacyHandshake runs a 2025-06-18 initialize with the given capabilities.
func (c *rawClientConn) legacyHandshake(capabilities string) {
	c.t.Helper()
	c.send(1, "initialize", `{"protocolVersion":"2025-06-18","capabilities":`+capabilities+
		`,"clientInfo":{"name":"legacy","version":"0"}}`)
	c.expectResponse(1)
	c.send(0, "notifications/initialized", `{}`)
}

const legacyUnsolicitedAccept = `{"name":"run_mutation_query","arguments":{"sql":"delete from t"},` +
	`"inputResponses":{"` + approvalInputKey + `":{"action":"accept"}}}`

func TestApproval_UnsolicitedAcceptRefused_LegacyRevision(t *testing.T) {
	be := &testBackend{execOut: map[string]any{"timestamp": "now"}}
	c := newRawClientConn(t, DefaultConfig(), be)
	c.legacyHandshake(`{"elicitation":{}}`)
	c.expectRefusal(2, legacyUnsolicitedAccept, "no approval state")
	forged := `{"name":"run_mutation_query","arguments":{"sql":"delete from t"},` +
		`"inputResponses":{"` + approvalInputKey + `":{"action":"accept"}},"requestState":"e30.AAAA"}`
	c.expectRefusal(3, forged, "not issued by this server")
	if be.execCalls != 0 {
		t.Fatalf("unsolicited accept must not execute, backend ran %d times", be.execCalls)
	}
}

// The reproduction in the issue: no elicitation capability, no state.
func TestApproval_UnsolicitedAcceptRefused_NoElicitation(t *testing.T) {
	be := &testBackend{execOut: map[string]any{"timestamp": "now"}}
	c := newRawClientConn(t, DefaultConfig(), be)
	c.legacyHandshake(`{}`)
	c.expectRefusal(2, legacyUnsolicitedAccept, "does not support elicitation")
	if be.execCalls != 0 {
		t.Fatalf("accept without elicitation must not execute, backend ran %d times", be.execCalls)
	}
}

// A state issued for one call is refused on any other: changed SQL, a
// changed non-SQL argument, or a different tool.  A refused attempt consumes
// nothing, so the legitimate retry with the original call still proceeds.
func TestApproval_AlteredCallRefusedThenLegitimateRetryProceeds(t *testing.T) {
	be := &testBackend{execOut: map[string]any{"timestamp": "now"}}
	c := newRawClientConn(t, DefaultConfig(), be)
	state := c.expectApprovalPrompt(1)

	changedSQL := `"name":"run_mutation_query","arguments":{"sql":"delete from u"}`
	c.expectRefusal(2, currentRevisionRetry(currentRevisionMeta, changedSQL, "accept", state), "does not match")
	changedArgs := `"name":"run_mutation_query","arguments":{"sql":"delete from t","format":"json"}`
	c.expectRefusal(3, currentRevisionRetry(currentRevisionMeta, changedArgs, "accept", state), "does not match")
	changedTool := `"name":"run_lifecycle_operation","arguments":{"sql":"delete from t"}`
	c.expectRefusal(4, currentRevisionRetry(currentRevisionMeta, changedTool, "accept", state), "does not match")
	tampered := state[:len(state)-2] + "AA"
	if tampered == state {
		tampered = state[:len(state)-2] + "BB"
	}
	c.expectRefusal(5, currentRevisionRetry(currentRevisionMeta, currentRevisionDeleteArgs, "accept", tampered),
		"not issued by this server")
	if be.execCalls != 0 {
		t.Fatalf("altered calls must not execute, backend ran %d times", be.execCalls)
	}

	c.send(6, "tools/call", currentRevisionRetry(currentRevisionMeta, currentRevisionDeleteArgs, "accept", state))
	if result := c.expectResponse(6); strings.Contains(string(result), `"isError":true`) {
		t.Fatalf("legitimate retry errored: %s", result)
	}
	if be.execCalls != 1 || be.lastExecQuery != "delete from t" {
		t.Fatalf("legitimate retry should execute once, got %d calls, last %q", be.execCalls, be.lastExecQuery)
	}
}

func TestApproval_ReplayRefused(t *testing.T) {
	be := &testBackend{execOut: map[string]any{"timestamp": "now"}}
	c := newRawClientConn(t, DefaultConfig(), be)
	state := c.expectApprovalPrompt(1)
	retry := currentRevisionRetry(currentRevisionMeta, currentRevisionDeleteArgs, "accept", state)
	c.send(2, "tools/call", retry)
	if result := c.expectResponse(2); strings.Contains(string(result), `"isError":true`) {
		t.Fatalf("first accept errored: %s", result)
	}
	c.expectRefusal(3, retry, "already used")
	if be.execCalls != 1 {
		t.Fatalf("replayed accept must not execute again, backend ran %d times", be.execCalls)
	}
}

func TestApproval_ExpiredStateRefused(t *testing.T) {
	be := &testBackend{execOut: map[string]any{"timestamp": "now"}}
	c := newRawClientConn(t, DefaultConfig(), be)
	state := c.expectApprovalPrompt(1)
	c.srv.approvals.now = func() time.Time { return time.Now().Add(approvalStateTTL + time.Second) }
	c.expectRefusal(2, currentRevisionRetry(currentRevisionMeta, currentRevisionDeleteArgs, "accept", state),
		"expired")
	if be.execCalls != 0 {
		t.Fatalf("expired accept must not execute, backend ran %d times", be.execCalls)
	}
}

// The elicitation check runs before the answer is read, even on a retry
// that echoes a valid state.
func TestApproval_RetryWithoutElicitationRefused(t *testing.T) {
	be := &testBackend{execOut: map[string]any{"timestamp": "now"}}
	c := newRawClientConn(t, DefaultConfig(), be)
	state := c.expectApprovalPrompt(1)
	c.expectRefusal(2, currentRevisionRetry(currentRevisionMetaNoElicit, currentRevisionDeleteArgs, "accept", state),
		"does not support elicitation")
	if be.execCalls != 0 {
		t.Fatalf("accept without elicitation must not execute, backend ran %d times", be.execCalls)
	}
}

// A state issued by one server process is refused by another with its own
// random key, which is the substitution a shared approval key opts into.
func TestApproval_StateFromAnotherServerRefused(t *testing.T) {
	issuer := newRawClientConn(t, DefaultConfig(), &testBackend{})
	state := issuer.expectApprovalPrompt(1)

	be := &testBackend{execOut: map[string]any{"timestamp": "now"}}
	other := newRawClientConn(t, DefaultConfig(), be)
	other.expectRefusal(1, currentRevisionRetry(currentRevisionMeta, currentRevisionDeleteArgs, "accept", state),
		"not issued by this server")
	if be.execCalls != 0 {
		t.Fatalf("foreign state must not execute, backend ran %d times", be.execCalls)
	}
}

// Session binding: a state issued on one session is refused on another.  The
// in-memory and stdio transports report an empty session id, so this is
// exercised on the binder directly.
func TestApprovalBinder_SessionAndKeyBinding(t *testing.T) {
	a, err := newApprovalBinder(DefaultConfig())
	if err != nil {
		t.Fatalf("newApprovalBinder: %v", err)
	}
	digest, err := approvalDigest("run_mutation_query", map[string]any{"sql": "delete from t"}, "delete from t")
	if err != nil {
		t.Fatalf("approvalDigest: %v", err)
	}
	onA := approvalBinding{tool: "run_mutation_query", digest: digest, session: "session-a"}
	state, err := a.issue(onA)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	onB := onA
	onB.session = "session-b"
	if verr := a.verify(state, onB); !errors.Is(verr, errApprovalStateMismatch) {
		t.Fatalf("cross-session state: got %v, want mismatch", verr)
	}
	b, err := newApprovalBinder(DefaultConfig())
	if err != nil {
		t.Fatalf("newApprovalBinder: %v", err)
	}
	if verr := b.verify(state, onA); !errors.Is(verr, errApprovalStateInvalid) {
		t.Fatalf("state under another key: got %v, want invalid", verr)
	}
	if verr := a.verify(state, onA); verr != nil {
		t.Fatalf("matching session: %v", verr)
	}
	if verr := a.verify(state, onA); !errors.Is(verr, errApprovalStateReplayed) {
		t.Fatalf("second use: got %v, want replayed", verr)
	}
}

func TestApprovalBinder_KeyFromEnv(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Server.ApprovalKeyEnvVar = "STACKQL_MCP_TEST_APPROVAL_KEY"
	if _, err := newApprovalBinder(cfg); err == nil {
		t.Fatal("an unset approval key env var must fail startup")
	}
	t.Setenv("STACKQL_MCP_TEST_APPROVAL_KEY", "short")
	_, err := newApprovalBinder(cfg)
	if err == nil {
		t.Fatal("a short approval key must fail startup")
	}
	if strings.Contains(err.Error(), "short") {
		t.Fatalf("the error must not echo the key value: %v", err)
	}
}

// statelessInstance serves one MCP server over sessionless Streamable HTTP.
func statelessInstance(t *testing.T, cfg *Config, backend Backend) string {
	t.Helper()
	cfg.Server.Audit.Disabled = true
	cfg.Server.Stateless = true
	mcpSrv, err := newMCPServer(cfg, backend, nil)
	if err != nil {
		t.Fatalf("newMCPServer: %v", err)
	}
	server := mcpSrv.(*simpleMCPServer).server
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{Stateless: true})
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	return ts.URL
}

// statelessClient connects a 2026-07-28 SDK client that advertises
// elicitation.  With manual set, the SDK's automatic retry is disabled so the
// test drives the input-required round trip itself.
func statelessClient(t *testing.T, url string, manual bool) *mcp.ClientSession {
	t.Helper()
	opts := &mcp.ClientOptions{ElicitationHandler: acceptingElicit}
	if manual {
		opts.MultiRoundTrip = &mcp.MultiRoundTripOptions{Disabled: true}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0"}, opts)
	cs, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: url, MaxRetries: -1}, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func callParams(t *testing.T, cs *mcp.ClientSession, params *mcp.CallToolParams) *mcp.CallToolResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := cs.CallTool(ctx, params)
	if err != nil {
		t.Fatalf("CallTool(%s): %v", params.Name, err)
	}
	return res
}

func acceptWith(state string) *mcp.CallToolParams {
	return &mcp.CallToolParams{
		Name:           "run_mutation_query",
		Arguments:      map[string]any{"sql": "delete from t"},
		InputResponses: mcp.InputResponseMap{approvalInputKey: &mcp.ElicitResult{Action: "accept"}},
		RequestState:   state,
	}
}

// Sessionless HTTP (stateless: true, the 2026-07-28 transport): the approval
// round trip works end to end, and an unsolicited or replayed accept is
// refused.  With approval_key_env_var shared, a prompt issued by one
// instance is accepted by another; with per-process keys it is not.
//
//nolint:gocognit // one scenario per deployment shape
func TestApproval_StatelessHTTP(t *testing.T) {
	deleteCall := &mcp.CallToolParams{Name: "run_mutation_query", Arguments: map[string]any{"sql": "delete from t"}}

	t.Run("automatic round trip", func(t *testing.T) {
		be := &testBackend{execOut: map[string]any{"timestamp": "now"}}
		cs := statelessClient(t, statelessInstance(t, DefaultConfig(), be), false)
		if res := callParams(t, cs, deleteCall); res.IsError {
			t.Fatalf("accepted call errored: %s", firstText(t, res))
		}
		if be.execCalls != 1 {
			t.Fatalf("accepted delete should execute once, backend ran %d times", be.execCalls)
		}
	})

	t.Run("unsolicited and replayed accept", func(t *testing.T) {
		be := &testBackend{execOut: map[string]any{"timestamp": "now"}}
		cs := statelessClient(t, statelessInstance(t, DefaultConfig(), be), true)
		if res := callParams(t, cs, acceptWith("")); !res.IsError ||
			!strings.Contains(firstText(t, res), "no approval state") {
			t.Fatalf("unsolicited accept should be refused, got %+v", res)
		}
		first := callParams(t, cs, deleteCall)
		if !first.NeedsInput() || first.RequestState == "" {
			t.Fatalf("expected an input-required prompt with state, got %+v", first)
		}
		if res := callParams(t, cs, acceptWith(first.RequestState)); res.IsError {
			t.Fatalf("legitimate retry errored: %s", firstText(t, res))
		}
		if res := callParams(t, cs, acceptWith(first.RequestState)); !res.IsError ||
			!strings.Contains(firstText(t, res), "already used") {
			t.Fatalf("replayed accept should be refused, got %+v", res)
		}
		if be.execCalls != 1 {
			t.Fatalf("backend ran %d times, want exactly 1", be.execCalls)
		}
	})

	t.Run("instances sharing approval_key_env_var", func(t *testing.T) {
		t.Setenv("STACKQL_MCP_TEST_APPROVAL_KEY", strings.Repeat("k", minApprovalKeyBytes))
		shared := func() *Config {
			cfg := DefaultConfig()
			cfg.Server.ApprovalKeyEnvVar = "STACKQL_MCP_TEST_APPROVAL_KEY"
			return cfg
		}
		beA := &testBackend{execOut: map[string]any{"timestamp": "now"}}
		beB := &testBackend{execOut: map[string]any{"timestamp": "now"}}
		beC := &testBackend{execOut: map[string]any{"timestamp": "now"}}
		instA := statelessClient(t, statelessInstance(t, shared(), beA), true)
		instB := statelessClient(t, statelessInstance(t, shared(), beB), true)
		instC := statelessClient(t, statelessInstance(t, DefaultConfig(), beC), true)

		first := callParams(t, instA, deleteCall)
		if !first.NeedsInput() || first.RequestState == "" {
			t.Fatalf("expected an input-required prompt with state, got %+v", first)
		}
		if res := callParams(t, instC, acceptWith(first.RequestState)); !res.IsError ||
			!strings.Contains(firstText(t, res), "not issued by this server") {
			t.Fatalf("an instance with its own key should refuse, got %+v", res)
		}
		if res := callParams(t, instB, acceptWith(first.RequestState)); res.IsError {
			t.Fatalf("an instance sharing the key should accept, got %s", firstText(t, res))
		}
		if beA.execCalls != 0 || beB.execCalls != 1 || beC.execCalls != 0 {
			t.Fatalf("executions A=%d B=%d C=%d, want 0/1/0", beA.execCalls, beB.execCalls, beC.execCalls)
		}
	})
}
