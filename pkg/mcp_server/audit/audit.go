// Package audit holds the MCP-specific audit event shape and the string
// constants for decisions and failure modes recorded in that shape.
//
// This package intentionally does NOT define the sink interface or any sink
// implementation -- those live under pkg/sink and are generic across
// subsystems (audit, future activity/telemetry channels, etc).  When new
// kinds of sinks (file rotation policies, alternative transports, GC) land,
// they benefit every consumer, not just audit.
package audit

import "time"

// Decision strings recorded in Event.Decision.  These constants are the audit
// contract; downstream tools may parse the log against these values.
const (
	DecisionAllow                    = "allow"
	DecisionRefuseImmediate          = "refuse_immediate"
	DecisionNeedsApprovalAccepted    = "needs_approval_accepted"
	DecisionNeedsApprovalDeclined    = "needs_approval_declined"
	DecisionNeedsApprovalCancelled   = "needs_approval_cancelled"
	DecisionNeedsApprovalUnavailable = "needs_approval_unavailable"
)

// FailureMode strings.  Legal values for the MCP server's audit.failure_mode.
const (
	FailureModeStrict          = "strict"
	FailureModeStrictMutations = "strict_mutations"
	FailureModeBestEffort      = "best_effort"
)

// Event is one record in the MCP audit log.  All fields are primitive types so
// the struct never leaks internal/SDK types across the package boundary.
//
// Event is what flows through a pkg/sink.Sink: the gate middleware constructs
// it, the sink JSON-marshals it.  The sink itself is unaware of the audit
// semantics; rotating the file, adding a new transport, etc are all done in
// pkg/sink without touching this file.
type Event struct {
	Timestamp  time.Time      `json:"timestamp"`
	Tool       string         `json:"tool"`
	Mode       string         `json:"mode"`
	Decision   string         `json:"decision"`
	QueryClass string         `json:"query_class,omitempty"`
	SQL        string         `json:"sql,omitempty"`
	Args       map[string]any `json:"args,omitempty"`
	DurationMs int64          `json:"duration_ms"`
	Error      string         `json:"error,omitempty"`
}
