package mcp_server //nolint:testpackage,revive // exercise internal wiring

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// blockingBackend blocks ListProviders until release is closed or ctx ends.
type blockingBackend struct {
	testBackend
	release chan struct{}
	active  atomic.Int32
	peak    atomic.Int32
}

func (b *blockingBackend) ListProviders(ctx context.Context) ([]map[string]any, error) {
	n := b.active.Add(1)
	defer b.active.Add(-1)
	for {
		p := b.peak.Load()
		if n <= p || b.peak.CompareAndSwap(p, n) {
			break
		}
	}
	select {
	case <-b.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return []map[string]any{}, nil
}

func TestMaxConcurrentRequestsLimitsToolCalls(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Server.MaxConcurrentRequests = 1
	cfg.Server.RequestTimeout = 0
	be := &blockingBackend{release: make(chan struct{})}
	cs := connectInProcess(t, cfg, be)

	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "list_providers"})
		}()
	}
	time.Sleep(150 * time.Millisecond)
	close(be.release)
	wg.Wait()
	if got := be.peak.Load(); got != 1 {
		t.Fatalf("peak concurrent backend calls = %d, want 1", got)
	}
}

func TestRequestTimeoutAppliedToToolCalls(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Server.RequestTimeout = Duration(20 * time.Millisecond)
	be := &blockingBackend{release: make(chan struct{})}
	cs := connectInProcess(t, cfg, be)
	defer close(be.release)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "list_providers"})
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("tool call still blocked well past request_timeout")
	}
}

func TestQueuedToolCallReleasesPermitOnCancel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Server.MaxConcurrentRequests = 1
	cfg.Server.RequestTimeout = Duration(50 * time.Millisecond)
	be := &blockingBackend{release: make(chan struct{})}
	cs := connectInProcess(t, cfg, be)
	defer close(be.release)

	// First call times out and must free its permit; later calls must proceed.
	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, _ = cs.CallTool(ctx, &mcp.CallToolParams{Name: "list_providers"})
		expired := ctx.Err() != nil
		cancel()
		if expired {
			t.Fatalf("call %d did not return before client deadline: permit leaked", i)
		}
	}
}

func TestNonPositiveLimitsDisableControls(t *testing.T) {
	if newRequestSemaphore(0) != nil || newRequestSemaphore(-1) != nil {
		t.Fatal("non-positive limit must not create a semaphore")
	}
}
