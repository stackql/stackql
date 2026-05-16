// Package audit records what the MCP server agreed to do on behalf of an LLM
// client.  The audit answers "what did the agent do," not "what did the agent
// see" - result rows from SELECTs are intentionally not recorded.
package audit

import (
	"context"
	"time"
)

// Decision strings recorded in Event.Decision.  These constants are the
// audit contract; downstream tools may parse the log against these values.
const (
	DecisionAllow                    = "allow"
	DecisionRefuseImmediate          = "refuse_immediate"
	DecisionNeedsApprovalAccepted    = "needs_approval_accepted"
	DecisionNeedsApprovalDeclined    = "needs_approval_declined"
	DecisionNeedsApprovalCancelled   = "needs_approval_cancelled"
	DecisionNeedsApprovalUnavailable = "needs_approval_unavailable"
)

// FailureMode strings.  These are the legal values for AuditConfig.FailureMode.
const (
	FailureModeStrict          = "strict"
	FailureModeStrictMutations = "strict_mutations"
	FailureModeBestEffort      = "best_effort"
)

// Event is one record in the audit log.  All fields are primitive types so the
// struct never leaks internal/SDK types across the package boundary.
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

// Sink is the audit destination.  Implementations must be safe for concurrent
// calls from multiple goroutines.
type Sink interface {
	Record(ctx context.Context, event Event) error
	Close() error
}

// nopSink discards every event.  Used when audit is disabled.
type nopSink struct{}

// NewNopSink returns a Sink that ignores every Record call.
func NewNopSink() Sink { return &nopSink{} }

func (*nopSink) Record(_ context.Context, _ Event) error { return nil }
func (*nopSink) Close() error                            { return nil }
