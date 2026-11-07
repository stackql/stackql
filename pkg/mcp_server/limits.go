package mcp_server //nolint:revive // fine for now

import (
	"context"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// newRequestSemaphore returns a semaphore admitting at most limit concurrent
// tool calls, or nil when limit is not positive (no limit).
func newRequestSemaphore(limit int) *semaphore.Weighted {
	if limit <= 0 {
		return nil
	}
	return semaphore.NewWeighted(int64(limit))
}

// toolCallLimits returns receiving middleware enforcing server.max_concurrent_requests
// and server.request_timeout on tools/call requests.  A nil semaphore or a
// non-positive timeout disables the corresponding control.  Waiting for a
// permit is cancellation-aware, and the permit is released on every exit path.
func toolCallLimits(sem *semaphore.Weighted, timeout time.Duration) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if method != "tools/call" {
				return next(ctx, method, req)
			}
			if timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}
			if sem != nil {
				if err := sem.Acquire(ctx, 1); err != nil {
					return nil, err
				}
				defer sem.Release(1)
			}
			return next(ctx, method, req)
		}
	}
}
