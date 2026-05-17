// Package sink provides a generic record-and-close interface for writing
// JSON-marshalable payloads to a destination, and the concrete implementations
// (file, nop) the rest of the codebase consumes.
//
// The interface is deliberately payload-agnostic: callers pass any value that
// json.Marshal can handle, and the sink takes responsibility for serialisation,
// transport, and durability semantics.  This lets unrelated subsystems (audit,
// future activity/telemetry channels, etc) share the same plumbing -- file
// rotation, GC, alternate transports plug in once and benefit everyone.
package sink

import "context"

// Sink is the generic destination contract.  Implementations must be safe for
// concurrent calls from multiple goroutines.
type Sink interface {
	// Record serialises and writes one payload.  Errors include transport
	// failures (full disk, broken pipe, etc) and marshalling failures.
	Record(ctx context.Context, payload any) error
	// Close flushes any buffered state and releases the underlying resource.
	Close() error
}

// NopSink discards every payload.  Useful as a zero-overhead default when a
// subsystem is configured off.
type nopSink struct{}

// NewNopSink returns a Sink that ignores every Record call.
func NewNopSink() Sink { return &nopSink{} }

func (*nopSink) Record(_ context.Context, _ any) error { return nil }
func (*nopSink) Close() error                          { return nil }
