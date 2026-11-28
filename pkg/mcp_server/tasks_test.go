package mcp_server //nolint:testpackage,revive // exercise internal wiring

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sirupsen/logrus"

	"github.com/stackql/stackql/pkg/mcp_server/audit"
)

const taskTestIaCSQL = "INSERT INTO stackql_preview.iac.run (collection, resources) VALUES ('task-test', '[]') RETURNING *"

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func TestMemoryTaskStoreLifecycleOwnershipAndExpiry(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 11, 28, 0, 0, 0, 0, time.UTC)}
	store := newMemoryTaskStore(90*time.Minute, time.Hour, clock.Now)

	created, err := store.create("alice", "running")
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[A-Za-z0-9_-]{32}$`).MatchString(created.ID()) {
		t.Fatalf("task id %q is not 32 base64url characters", created.ID())
	}
	if _, ok := store.get(created.ID(), "mallory"); ok {
		t.Fatal("another principal read the task")
	}
	if store.requestCancel(created.ID(), "mallory") {
		t.Fatal("another principal cancelled the task")
	}
	if !store.requestCancel(created.ID(), "alice") {
		t.Fatal("owner could not request cancellation")
	}
	got, ok := store.get(created.ID(), "alice")
	if !ok || got.Status() != taskStatusWorking || !strings.Contains(got.StatusMessage(), "not interrupted") {
		t.Fatalf("after cancel request: %+v %v", got, ok)
	}
	clock.advance(time.Minute)
	store.finish(created.ID(), &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil)
	got, _ = store.get(created.ID(), "alice")
	if got.Status() != taskStatusCompleted || got.Result() == nil || got.Failure() != nil ||
		!strings.Contains(got.StatusMessage(), "cancellation was requested") {
		t.Fatalf("after finish: status=%s message=%q", got.Status(), got.StatusMessage())
	}
	// A terminal state does not change.
	store.finish(created.ID(), nil, &taskError{Code: -32603, Message: "late"})
	if got, _ = store.get(created.ID(), "alice"); got.Status() != taskStatusCompleted {
		t.Fatalf("terminal task changed to %s", got.Status())
	}
	clock.advance(59 * time.Minute)
	if _, ok = store.get(created.ID(), "alice"); !ok {
		t.Fatal("terminal task evicted before retention")
	}
	clock.advance(2 * time.Minute)
	if _, ok = store.get(created.ID(), "alice"); ok {
		t.Fatal("terminal task outlived retention")
	}

	stuck, _ := store.create("alice", "running")
	clock.advance(91 * time.Minute)
	if _, ok = store.get(stuck.ID(), "alice"); ok {
		t.Fatal("working task outlived its ttl")
	}
	discarded, _ := store.create("alice", "running")
	store.discard(discarded.ID())
	if _, ok = store.get(discarded.ID(), "alice"); ok {
		t.Fatal("discarded task is readable")
	}
}

func TestRunTaskWorkMirrorsSDKErrorHandling(t *testing.T) {
	ok := &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "rows"}}}
	result, failure, toolErr := runTaskWork(context.Background(), nil,
		func(context.Context, Backend) (*mcp.CallToolResult, error) { return ok, nil })
	if result != ok || toolErr != nil || failure != nil {
		t.Fatalf("success: %v %v %v", result, toolErr, failure)
	}
	result, failure, toolErr = runTaskWork(context.Background(), nil,
		func(context.Context, Backend) (*mcp.CallToolResult, error) {
			return nil, errors.New("provider said no")
		})
	if failure != nil || toolErr == nil || result == nil || !result.IsError || firstText(t, result) != "provider said no" {
		t.Fatalf("tool error must complete with an isError result: %v %v %v", result, toolErr, failure)
	}
	_, failure, _ = runTaskWork(context.Background(), nil,
		func(context.Context, Backend) (*mcp.CallToolResult, error) {
			return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: "bad"}
		})
	if failure == nil || failure.Code != jsonrpc.CodeInvalidParams || failure.Message != "bad" {
		t.Fatalf("JSON-RPC error must fail the task: %+v", failure)
	}
	_, failure, _ = runTaskWork(context.Background(), nil,
		func(context.Context, Backend) (*mcp.CallToolResult, error) { panic("boom") })
	if failure == nil || failure.Code != jsonrpc.CodeInternalError || !strings.Contains(failure.Message, "boom") {
		t.Fatalf("panic must fail the task: %+v", failure)
	}
}

func TestWithTaskTimeoutSharesSlotsAndNamesItsSetting(t *testing.T) {
	limited := newLimitedBackend(&testBackend{}, 2, time.Second).(*limitedBackend)
	view := withTaskTimeout(limited, time.Hour).(*limitedBackend)
	if view.slots != limited.slots || view.timeout != time.Hour || view.timeoutSetting != taskTimeoutSetting {
		t.Fatalf("task view = %+v", view)
	}
	if !strings.Contains(view.timeoutError(context.DeadlineExceeded, true).Error(), taskTimeoutSetting) {
		t.Fatal("task timeout error must name server.extensions.tasks.timeout")
	}
	if !strings.Contains(limited.timeoutError(context.DeadlineExceeded, true).Error(), "server.request_timeout") {
		t.Fatal("synchronous timeout error must keep naming server.request_timeout")
	}
	unlimited := withTaskTimeout(&testBackend{}, time.Hour).(*limitedBackend)
	if unlimited.slots != nil || unlimited.timeout != time.Hour {
		t.Fatalf("task view of an unlimited backend = %+v", unlimited)
	}
}

// taskRecordingSink records events and fails the terminal task record.
type taskRecordingSink struct {
	mu     sync.Mutex
	events []audit.Event
}

func (s *taskRecordingSink) Record(_ context.Context, payload any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	event, _ := payload.(audit.Event)
	s.events = append(s.events, event)
	if event.TaskStatus == string(taskStatusCompleted) || event.TaskStatus == string(taskStatusFailed) {
		return errors.New("probe audit failure")
	}
	return nil
}

func (s *taskRecordingSink) Close() error { return nil }

func (s *taskRecordingSink) snapshot() []audit.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]audit.Event(nil), s.events...)
}

// mcpHTTPCall posts one 2026-07-28 JSON-RPC request and returns the response.
func mcpHTTPCall(
	t *testing.T, client *http.Client, endpoint, bearer, method string, params map[string]any, caps map[string]any,
) map[string]any {
	t.Helper()
	params["_meta"] = map[string]any{
		mcp.MetaKeyProtocolVersion:    protocolVersionSessionless,
		mcp.MetaKeyClientCapabilities: caps,
		mcp.MetaKeyClientInfo:         map[string]any{"name": "task-test", "version": "1"},
	}
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	request, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("MCP-Protocol-Version", protocolVersionSessionless)
	request.Header.Set("Mcp-Method", method)
	if name, ok := params["name"].(string); ok && method == "tools/call" {
		request.Header.Set("Mcp-Name", name)
	}
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var payload []byte
	if strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") {
		scanner := bufio.NewScanner(response.Body)
		scanner.Buffer(make([]byte, 1<<20), 1<<20)
		for scanner.Scan() {
			if line := scanner.Text(); strings.HasPrefix(line, "data:") {
				payload = []byte(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
		}
	} else {
		raw := new(strings.Builder)
		if _, err := bufio.NewReader(response.Body).WriteTo(raw); err != nil {
			t.Fatal(err)
		}
		payload = []byte(raw.String())
	}
	var out map[string]any
	if err := json.Unmarshal(payload, &out); err != nil {
		t.Fatalf("%s: undecodable response %q (HTTP %d)", method, payload, response.StatusCode)
	}
	return out
}

var declaresTasks = map[string]any{"extensions": map[string]any{tasksExtensionID: map[string]any{}}}

func resultOf(t *testing.T, response map[string]any) map[string]any {
	t.Helper()
	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected a result: %v", response)
	}
	return result
}

func errorCodeOf(t *testing.T, response map[string]any) float64 {
	t.Helper()
	e, ok := response["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected a JSON-RPC error: %v", response)
	}
	return e["code"].(float64)
}

func pollTask(t *testing.T, call func(string, map[string]any) map[string]any, id string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		got := resultOf(t, call("tasks/get", map[string]any{"taskId": id}))
		if got["status"] != string(taskStatusWorking) {
			return got
		}
		if time.Now().After(deadline) {
			t.Fatalf("task %s still working", id)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func assertCreateTaskResult(t *testing.T, created map[string]any) string {
	t.Helper()
	if created["resultType"] != "task" || created["status"] != string(taskStatusWorking) {
		t.Fatalf("not a CreateTaskResult: %v", created)
	}
	for _, field := range []string{"taskId", "createdAt", "lastUpdatedAt", "ttlMs", "pollIntervalMs"} {
		if created[field] == nil {
			t.Fatalf("CreateTaskResult lacks %s: %v", field, created)
		}
	}
	return created["taskId"].(string)
}

// A strict audit-write failure on completion fails the task with an error
// object, as the same failure fails a synchronous call.
func TestTaskFailureCarriesErrorObject(t *testing.T) {
	cfg := fullAccessConfig()
	cfg.Server.Transport = serverTransportHTTP
	cfg.Server.Stateless = true
	cfg.Server.AllowUnauthenticated = true
	cfg.Server.Audit.FailureMode = audit.FailureModeStrict
	cfg.Server.Extensions.Tasks.Enabled = true
	backend := &testBackend{execOut: map[string]any{"rows": []any{}}}
	recorder := &taskRecordingSink{}

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0"},
		&mcp.ServerOptions{Capabilities: buildServerCapabilities(cfg)})
	approvals, err := newApprovalBinder(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := registerTools(server, cfg, backend, logrus.New(), recorder, approvals); err != nil {
		t.Fatal(err)
	}
	if err := newTaskManager(cfg.Server.Extensions.Tasks, backend).register(server); err != nil {
		t.Fatal(err)
	}
	handler, err := secureHTTPHandler(newMCPStreamableHTTPHandler(server, cfg), cfg)
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()
	call := func(method string, params map[string]any) map[string]any {
		return mcpHTTPCall(t, httpServer.Client(), httpServer.URL, "", method, params, declaresTasks)
	}

	created := resultOf(t, call("tools/call", map[string]any{
		"name": "run_mutation_query", "arguments": map[string]any{"sql": taskTestIaCSQL}}))
	id := assertCreateTaskResult(t, created)
	done := pollTask(t, call, id)
	failure, _ := done["error"].(map[string]any)
	if done["status"] != string(taskStatusFailed) || failure == nil || done["result"] != nil {
		t.Fatalf("task = %v, want failed with an error object", done)
	}
	if failure["code"].(float64) != jsonrpc.CodeInternalError ||
		!strings.Contains(failure["message"].(string), "audit record could not be written") {
		t.Fatalf("error object = %v", failure)
	}
	if backend.execCalls != 1 || backend.lastExecQuery != taskTestIaCSQL {
		t.Fatalf("backend ran %d times (%q)", backend.execCalls, backend.lastExecQuery)
	}
	events := recorder.snapshot()
	if len(events) != 2 || events[0].TaskID != id || events[0].TaskStatus != string(taskStatusWorking) ||
		events[1].TaskID != id || events[1].TaskStatus != string(taskStatusCompleted) {
		t.Fatalf("audit records = %+v, want a working decision record then the completion record", events)
	}
	if events[0].Decision != audit.DecisionAllow || events[0].SQL != taskTestIaCSQL {
		t.Fatalf("decision record = %+v", events[0])
	}
}

// Over HTTP each task belongs to the principal that created it.
func TestTaskIsNotReadableByAnotherPrincipal(t *testing.T) {
	issuer := newOAuthIntegrationIssuer(t)
	cfg := DefaultHTTPConfig()
	cfg.Server.Mode = "full_access"
	cfg.Server.Stateless = true
	cfg.Server.Audit.Disabled = true
	cfg.Server.OAuth = validOAuthConfig()
	cfg.Server.OAuth.JWKSURL = issuer.server.URL
	cfg.Server.OAuth.HTTPClient = issuer.server.Client()
	cfg.Server.Extensions.Tasks.Enabled = true

	backend := &testBackend{execOut: map[string]any{"rows": []any{map[string]any{"status": "applied"}}}}
	mcpServer, err := newMCPServer(cfg, backend, nil)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := secureHTTPHandler(newMCPStreamableHTTPHandler(mcpServer.(*simpleMCPServer).server, cfg), cfg)
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewTLSServer(handler)
	defer httpServer.Close()
	tokenFor := func(subject string) string { return issuer.token(t, claimsWith(oauthTestClaims(), "sub", subject)) }
	alice, mallory := tokenFor("alice"), tokenFor("mallory")
	callAs := func(token string) func(string, map[string]any) map[string]any {
		return func(method string, params map[string]any) map[string]any {
			return mcpHTTPCall(t, httpServer.Client(), httpServer.URL, token, method, params, declaresTasks)
		}
	}

	created := resultOf(t, callAs(alice)("tools/call", map[string]any{
		"name": "run_mutation_query", "arguments": map[string]any{"sql": taskTestIaCSQL}}))
	id := assertCreateTaskResult(t, created)
	for _, method := range []string{"tasks/get", "tasks/cancel", "tasks/update"} {
		if code := errorCodeOf(t, callAs(mallory)(method, map[string]any{"taskId": id})); code != jsonrpc.CodeInvalidParams {
			t.Fatalf("%s by another principal: code %v, want -32602", method, code)
		}
	}
	done := pollTask(t, callAs(alice), id)
	if done["status"] != string(taskStatusCompleted) || done["result"] == nil {
		t.Fatalf("owner's task = %v", done)
	}
	if code := errorCodeOf(t, callAs(mallory)("tasks/get", map[string]any{"taskId": id})); code != jsonrpc.CodeInvalidParams {
		t.Fatalf("completed task readable by another principal: code %v", code)
	}
}

// Without the declared extension, or for a statement that is not an IaC run,
// run_mutation_query answers synchronously as before.
func TestTaskEligibilityKeepsOtherCallsSynchronous(t *testing.T) {
	cfg := fullAccessConfig()
	cfg.Server.Transport = serverTransportHTTP
	cfg.Server.Stateless = true
	cfg.Server.AllowUnauthenticated = true
	cfg.Server.Audit.Disabled = true
	cfg.Server.Extensions.Tasks.Enabled = true
	backend := &testBackend{execOut: map[string]any{"rows": []any{}}}
	mcpServer, err := newMCPServer(cfg, backend, nil)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := secureHTTPHandler(newMCPStreamableHTTPHandler(mcpServer.(*simpleMCPServer).server, cfg), cfg)
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()
	for _, tc := range []struct {
		name string
		sql  string
		caps map[string]any
	}{
		{"undeclared", taskTestIaCSQL, map[string]any{}},
		{"revisioned declaration", taskTestIaCSQL,
			map[string]any{"extensions": map[string]any{tasksExtensionID: map[string]any{"revision": "9"}}}},
		{"ordinary mutation", "INSERT INTO google.compute.networks (project, name) VALUES ('p', 'n')", declaresTasks},
	} {
		got := resultOf(t, mcpHTTPCall(t, httpServer.Client(), httpServer.URL, "", "tools/call", map[string]any{
			"name": "run_mutation_query", "arguments": map[string]any{"sql": tc.sql}}, tc.caps))
		if got["resultType"] == "task" || got["taskId"] != nil || got["content"] == nil {
			t.Fatalf("%s: want a synchronous result, got %v", tc.name, got)
		}
	}
}
