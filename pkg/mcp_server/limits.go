package mcp_server //nolint:revive // fine for now

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/stackql/stackql/pkg/mcp_server/dto"
)

// requestLimitError reports that server.max_concurrent_requests or
// server.request_timeout stopped a call, as distinct from a backend failure.
type requestLimitError struct {
	message string
	cause   error
}

func (e *requestLimitError) Error() string {
	if e.cause == nil {
		return e.message
	}
	return e.message + ": " + e.cause.Error()
}

func (e *requestLimitError) Unwrap() error { return e.cause }

func isRequestLimitError(err error) bool {
	var limitErr *requestLimitError
	return errors.As(err, &limitErr)
}

// limitedBackend enforces server.max_concurrent_requests and
// server.request_timeout around backend work.  The limits sit here, below the
// tool gate, and not around the whole tools/call: a call that is refused by
// policy, or that waits on a user's approval, holds no slot and spends none of
// its deadline.  For clients older than 2026-07-28 the SDK answers the
// approval prompt inside the tools/call, so limits applied as receiving
// middleware would count that wait.
type limitedBackend struct {
	inner   Backend
	slots   *semaphore.Weighted // nil when concurrency is unbounded
	limit   int
	timeout time.Duration // non-positive when calls carry no deadline
}

// newLimitedBackend wraps inner with the configured limits.  A non-positive
// value disables that limit; with both disabled inner is returned unwrapped.
func newLimitedBackend(inner Backend, maxConcurrent int, timeout time.Duration) Backend {
	if maxConcurrent <= 0 && timeout <= 0 {
		return inner
	}
	rv := &limitedBackend{inner: inner, limit: maxConcurrent, timeout: timeout}
	if maxConcurrent > 0 {
		rv.slots = semaphore.NewWeighted(int64(maxConcurrent))
	}
	return rv
}

// limited runs work inside the limits.  The deadline covers the wait for a
// slot as well as the work, waiting honours cancellation, and the slot is
// released on every exit path.  With no deadline the wait for a slot is
// bounded only by the caller.
func limited[T any](
	ctx context.Context, b *limitedBackend, isWrite bool, work func(context.Context) (T, error),
) (T, error) {
	runCtx := ctx
	if b.timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, b.timeout)
		defer cancel()
	}
	if b.slots != nil {
		if acquireErr := b.slots.Acquire(runCtx, 1); acquireErr != nil {
			var zero T
			return zero, b.refusal(ctx, acquireErr)
		}
		defer b.slots.Release(1)
	}
	out, err := work(runCtx)
	if err != nil && ctx.Err() == nil && errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		err = b.timeoutError(err, isWrite)
	}
	return out, err
}

// refusal explains a failed wait for a slot.  A caller that went away gets its
// own error back; otherwise the deadline passed with every slot still in use.
func (b *limitedBackend) refusal(ctx context.Context, acquireErr error) error {
	if ctx.Err() != nil {
		return acquireErr
	}
	return &requestLimitError{message: fmt.Sprintf(
		"server busy: all %d request slots (server.max_concurrent_requests) stayed in use for the %s "+
			"server.request_timeout; the call did not run, retry later", b.limit, b.timeout)}
}

// timeoutError names the setting that ended the call.  A write may already
// have reached the provider, so the client is told not to retry it blindly.
func (b *limitedBackend) timeoutError(cause error, isWrite bool) error {
	message := fmt.Sprintf("server.request_timeout of %s exceeded", b.timeout)
	if isWrite {
		message += "; the statement may already have taken effect, check before retrying"
	}
	return &requestLimitError{message: message, cause: cause}
}

func (b *limitedBackend) Ping(ctx context.Context) error { return b.inner.Ping(ctx) }

func (b *limitedBackend) Close() error { return b.inner.Close() }

// ServerInfo does no backend work, so it stays answerable on a saturated server.
func (b *limitedBackend) ServerInfo(ctx context.Context, args any) (dto.ServerInfoOutput, error) {
	return b.inner.ServerInfo(ctx, args)
}

func (b *limitedBackend) ExecQuery(ctx context.Context, query string) (map[string]any, error) {
	return limited(ctx, b, true, func(runCtx context.Context) (map[string]any, error) {
		return b.inner.ExecQuery(runCtx, query)
	})
}

func (b *limitedBackend) ValidateQuery(ctx context.Context, query string) ([]map[string]any, error) {
	return limited(ctx, b, false, func(runCtx context.Context) ([]map[string]any, error) {
		return b.inner.ValidateQuery(runCtx, query)
	})
}

func (b *limitedBackend) RunQueryJSON(ctx context.Context, input dto.QueryJSONInput) ([]map[string]any, error) {
	return limited(ctx, b, false, func(runCtx context.Context) ([]map[string]any, error) {
		return b.inner.RunQueryJSON(runCtx, input)
	})
}

func (b *limitedBackend) ListProviders(ctx context.Context) ([]map[string]any, error) {
	return limited(ctx, b, false, b.inner.ListProviders)
}

func (b *limitedBackend) ListServices(ctx context.Context, hI dto.HierarchyInput) ([]map[string]any, error) {
	return b.hierarchy(ctx, hI, b.inner.ListServices)
}

func (b *limitedBackend) ListResources(ctx context.Context, hI dto.HierarchyInput) ([]map[string]any, error) {
	return b.hierarchy(ctx, hI, b.inner.ListResources)
}

func (b *limitedBackend) ListMethods(ctx context.Context, hI dto.HierarchyInput) ([]map[string]any, error) {
	return b.hierarchy(ctx, hI, b.inner.ListMethods)
}

func (b *limitedBackend) DescribeMethod(ctx context.Context, hI dto.HierarchyInput) ([]map[string]any, error) {
	return b.hierarchy(ctx, hI, b.inner.DescribeMethod)
}

// hierarchy runs one of the read-only discovery calls inside the limits.
func (b *limitedBackend) hierarchy(
	ctx context.Context, hI dto.HierarchyInput,
	call func(context.Context, dto.HierarchyInput) ([]map[string]any, error),
) ([]map[string]any, error) {
	return limited(ctx, b, false, func(runCtx context.Context) ([]map[string]any, error) {
		return call(runCtx, hI)
	})
}

func (b *limitedBackend) ListRegistry(ctx context.Context, input dto.RegistryInput) ([]map[string]any, error) {
	return limited(ctx, b, false, func(runCtx context.Context) ([]map[string]any, error) {
		return b.inner.ListRegistry(runCtx, input)
	})
}

func (b *limitedBackend) PullProvider(ctx context.Context, input dto.RegistryInput) (map[string]any, error) {
	return limited(ctx, b, false, func(runCtx context.Context) (map[string]any, error) {
		return b.inner.PullProvider(runCtx, input)
	})
}

func (b *limitedBackend) ReloadCredentials(
	ctx context.Context, input dto.CredentialsReloadInput,
) (dto.CredentialsReloadDTO, error) {
	return limited(ctx, b, false, func(runCtx context.Context) (dto.CredentialsReloadDTO, error) {
		return b.inner.ReloadCredentials(runCtx, input)
	})
}
