package mcp_server //nolint:testpackage,revive // exercise internal wiring

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/stackql/stackql/pkg/mcp_server/audit"
	"github.com/stackql/stackql/pkg/mcp_server/dto"
)

// gatedBackend parks its calls until release is closed.  It honours
// cancellation unless deaf is set, which models work that cannot be interrupted.
type gatedBackend struct {
	testBackend
	release chan struct{}
	deaf    bool
	calls   atomic.Int32
	active  atomic.Int32
	peak    atomic.Int32
}

func newGatedBackend() *gatedBackend { return &gatedBackend{release: make(chan struct{})} }

func (b *gatedBackend) park(ctx context.Context) error {
	b.calls.Add(1)
	n := b.active.Add(1)
	defer b.active.Add(-1)
	for p := b.peak.Load(); n > p && !b.peak.CompareAndSwap(p, n); p = b.peak.Load() {
	}
	done := ctx.Done()
	if b.deaf {
		done = nil
	}
	select {
	case <-b.release:
		return nil
	case <-done:
		return ctx.Err()
	}
}

func (b *gatedBackend) ListProviders(ctx context.Context) ([]map[string]any, error) {
	return []map[string]any{}, b.park(ctx)
}

func (b *gatedBackend) ExecQuery(ctx context.Context, _ string) (map[string]any, error) {
	return map[string]any{}, b.park(ctx)
}

func (b *gatedBackend) ValidateQuery(ctx context.Context, _ string) ([]map[string]any, error) {
	return []map[string]any{}, b.park(ctx)
}

func TestLimitedBackendBoundsConcurrentWork(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		be := newGatedBackend()
		lb := newLimitedBackend(be, 2, 0)
		var wg sync.WaitGroup
		for range 5 {
			wg.Go(func() {
				if _, err := lb.ListProviders(t.Context()); err != nil {
					t.Errorf("queued call failed: %v", err)
				}
			})
		}
		synctest.Wait()
		if got := be.active.Load(); got != 2 {
			t.Errorf("calls in flight = %d, want the limit of 2", got)
		}
		close(be.release)
		wg.Wait()
		if calls, peak := be.calls.Load(), be.peak.Load(); calls != 5 || peak != 2 {
			t.Errorf("calls = %d, peak = %d; want all 5 to run and never more than 2 at once", calls, peak)
		}
	})
}

func TestLimitedBackendTimeoutEndsCallAndFreesSlot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		be := newGatedBackend()
		lb := newLimitedBackend(be, 1, 30*time.Second)
		started := time.Now()
		_, err := lb.ListProviders(t.Context())
		if !isRequestLimitError(err) || !errors.Is(err, context.DeadlineExceeded) ||
			!strings.Contains(err.Error(), "server.request_timeout of 30s exceeded") {
			t.Errorf("hung call error = %v, want the request_timeout error wrapping the deadline", err)
		}
		if waited := time.Since(started); waited != 30*time.Second {
			t.Errorf("hung call returned after %s, want 30s", waited)
		}
		// A leaked slot would leave this call queued until it is refused as busy.
		close(be.release)
		if _, err = lb.ListProviders(t.Context()); err != nil {
			t.Errorf("follow-up call failed, so the timed-out call kept its slot: %v", err)
		}
		if got := be.calls.Load(); got != 2 {
			t.Errorf("backend calls = %d, want the follow-up to reach the backend", got)
		}
	})
}

func TestLimitedBackendRefusesWhenNoSlotFrees(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		be := newGatedBackend()
		be.deaf = true
		lb := newLimitedBackend(be, 1, 30*time.Second)
		var wg sync.WaitGroup
		wg.Go(func() { _, _ = lb.ListProviders(t.Context()) })
		synctest.Wait()

		_, err := lb.ListProviders(t.Context())
		if !isRequestLimitError(err) || !strings.Contains(err.Error(), "server busy") ||
			!strings.Contains(err.Error(), "server.max_concurrent_requests") {
			t.Errorf("queued call error = %v, want the busy refusal", err)
		}
		if got := be.calls.Load(); got != 1 {
			t.Errorf("backend calls = %d, want the refused call to never run", got)
		}
		if _, err = lb.ServerInfo(t.Context(), nil); err != nil {
			t.Errorf("server_info must stay answerable on a saturated server: %v", err)
		}
		close(be.release)
		wg.Wait()
		if _, err = lb.ListProviders(t.Context()); err != nil {
			t.Errorf("call after the holder finished failed: %v", err)
		}
	})
}

func TestLimitedBackendQueuedCallHonoursCallerCancel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		be := newGatedBackend()
		lb := newLimitedBackend(be, 1, 0)
		var wg sync.WaitGroup
		wg.Go(func() { _, _ = lb.ListProviders(t.Context()) })
		synctest.Wait()

		ctx, cancel := context.WithCancel(t.Context())
		queued := make(chan error, 1)
		go func() {
			_, err := lb.ListProviders(ctx)
			queued <- err
		}()
		synctest.Wait()
		cancel()
		if err := <-queued; !errors.Is(err, context.Canceled) || isRequestLimitError(err) {
			t.Errorf("cancelled queued call error = %v, want the caller's own cancellation", err)
		}
		close(be.release)
		wg.Wait()
		if got := be.calls.Load(); got != 1 {
			t.Errorf("backend calls = %d, want the cancelled call to never run", got)
		}
	})
}

func TestLimitedBackendWriteTimeoutWarnsAgainstBlindRetry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lb := newLimitedBackend(newGatedBackend(), 0, time.Second)
		const warning = "may already have taken effect"
		if _, err := lb.ListProviders(t.Context()); err == nil || strings.Contains(err.Error(), warning) {
			t.Errorf("read timeout error = %v, want a timeout without the write warning", err)
		}
		if _, err := lb.ExecQuery(t.Context(), "delete from t"); err == nil || !strings.Contains(err.Error(), warning) {
			t.Errorf("write timeout error = %v, want the write warning", err)
		}
	})
}

// panickingBackend fails the first discovery call the hard way.
type panickingBackend struct{ testBackend }

func (b *panickingBackend) ListProviders(context.Context) ([]map[string]any, error) {
	panic("backend panic")
}

func TestLimitedBackendReleasesSlotWhenWorkPanics(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lb := newLimitedBackend(&panickingBackend{}, 1, time.Second)
		func() {
			defer func() { _ = recover() }()
			_, _ = lb.ListProviders(t.Context())
		}()
		if _, err := lb.ListServices(t.Context(), dto.HierarchyInput{}); err != nil {
			t.Errorf("call after a panic failed, so the panicking call kept its slot: %v", err)
		}
	})
}

func TestNonPositiveLimitsLeaveBackendUnwrapped(t *testing.T) {
	be := &testBackend{}
	if newLimitedBackend(be, 0, 0) != Backend(be) || newLimitedBackend(be, -1, -time.Second) != Backend(be) {
		t.Fatal("non-positive limits must return the backend unwrapped")
	}
}

// recordingSink keeps the audit events the gate writes.
type recordingSink struct {
	mu     sync.Mutex
	events []audit.Event
}

func (s *recordingSink) Record(_ context.Context, payload any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if event, ok := payload.(audit.Event); ok {
		s.events = append(s.events, event)
	}
	return nil
}

func (s *recordingSink) Close() error { return nil }

func (s *recordingSink) errorFor(tool string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, event := range s.events {
		if event.Tool == tool {
			return event.Error
		}
	}
	return ""
}

// A limit reaches the client as a tool error that names the setting, lands in
// the audit log, and is never reported as a verdict on the statement.
func TestRequestLimitsSurfaceAsAuditedToolErrors(t *testing.T) {
	cfg := fullAccessConfig()
	cfg.Server.Audit.Disabled = false
	be := newGatedBackend()
	be.deaf = true
	audited := &recordingSink{}
	cs := connectWithSink(t, cfg, newLimitedBackend(be, 1, 50*time.Millisecond), audited)

	holder := make(chan struct{})
	go func() {
		defer close(holder)
		_, _ = cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "list_providers"})
	}()
	t.Cleanup(func() {
		close(be.release)
		<-holder
	})
	for waited := time.Duration(0); be.active.Load() == 0; waited += time.Millisecond {
		if waited > 5*time.Second {
			t.Fatal("the slot holder never reached the backend")
		}
		time.Sleep(time.Millisecond)
	}

	busy := callTool(t, cs, "validate_select_query", map[string]any{"sql": "select 1"})
	if text := firstText(t, busy); !busy.IsError || !strings.Contains(text, "server busy") {
		t.Fatalf("queued call: IsError=%v text=%q, want the busy refusal as a tool error", busy.IsError, text)
	}
	if got := audited.errorFor("validate_select_query"); !strings.Contains(got, "server busy") {
		t.Fatalf("busy refusal audit error = %q, want it recorded", got)
	}
}

func TestRequestTimeoutAppliedToToolCalls(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Server.RequestTimeout = Duration(20 * time.Millisecond)
	be := newGatedBackend()
	defer close(be.release)
	cs := connectInProcess(t, cfg, be)

	res := callTool(t, cs, "list_providers", map[string]any{})
	if text := firstText(t, res); !res.IsError || !strings.Contains(text, "server.request_timeout of 20ms exceeded") {
		t.Fatalf("hung tool call: IsError=%v text=%q, want the request_timeout tool error", res.IsError, text)
	}
}

// A client older than 2026-07-28 answers the approval prompt inside the
// tools/call.  The limits must not cover that wait: an open prompt holds no
// slot and a slow answer does not spend the request deadline.
func TestApprovalWaitIsOutsideRequestLimits(t *testing.T) {
	cfg := DefaultConfig() // safe mode: the delete needs approval
	cfg.Server.MaxConcurrentRequests = 1
	cfg.Server.RequestTimeout = Duration(50 * time.Millisecond)
	be := &testBackend{execOut: map[string]any{"timestamp": "now"}}
	c := newRawClientConn(t, cfg, be)
	c.send(1, "initialize",
		`{"protocolVersion":"2025-06-18","capabilities":{"elicitation":{}},"clientInfo":{"name":"legacy","version":"0"}}`)
	c.expectResponse(1)
	c.send(0, "notifications/initialized", `{}`)

	c.send(2, "tools/call", legacyMutationCall)
	prompt, ok := c.read().(*jsonrpc.Request)
	if !ok || prompt.Method != "elicitation/create" {
		t.Fatalf("expected an elicitation/create request, got %#v", prompt)
	}

	c.send(3, "tools/call", `{"name":"list_providers","arguments":{}}`)
	if other := c.expectResponse(3); strings.Contains(string(other), `"isError":true`) {
		t.Fatalf("an open approval prompt held the only request slot: %s", other)
	}

	time.Sleep(3 * time.Duration(cfg.Server.RequestTimeout))
	if err := c.conn.Write(c.ctx, &jsonrpc.Response{ID: prompt.ID, Result: json.RawMessage(`{"action":"accept"}`)}); err != nil {
		t.Fatalf("write elicitation result: %v", err)
	}
	if result := c.expectResponse(2); strings.Contains(string(result), `"isError":true`) {
		t.Fatalf("a slow approval spent the request deadline: %s", result)
	}
	if be.lastExecQuery != "delete from t" {
		t.Fatalf("approved delete should reach the backend, got %q", be.lastExecQuery)
	}
}
