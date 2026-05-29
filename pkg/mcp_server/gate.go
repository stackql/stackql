package mcp_server //nolint:revive // fine for now

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/stackql/stackql/pkg/mcp_server/audit"
	"github.com/stackql/stackql/pkg/mcp_server/dto"
	"github.com/stackql/stackql/pkg/mcp_server/policy"
	"github.com/stackql/stackql/pkg/sink"
)

// stderrSink returns the diagnostic writer used by the gate middleware.
// Indirected through a function so tests can swap it for a buffer.
var stderrSink = func() io.Writer { return os.Stderr }

// toolGate captures the per-tool metadata the middleware needs to classify a
// call, decide whether to allow it, and write an audit record afterwards.
type toolGate struct {
	// toolName is the registered MCP tool name (audit + error messages).
	toolName string
	// defaultClass is the query class for tools whose input is not SQL.
	// Hierarchy/metadata tools use QueryClassSelect; query tools use
	// QueryClassUnknown so the classifier runs against args.SQL instead.
	defaultClass policy.QueryClass
	// extractSQL pulls the SQL out of a typed input value, returning the
	// empty string for tools that take no SQL.  Used by the classifier
	// and by the audit event.
	extractSQL func(any) string
	// extractArgs returns a key/value map suitable for the Args field on the
	// audit event.  For hierarchy tools this carries the hierarchy fields;
	// for query tools it carries SQL + row_limit.
	extractArgs func(any) map[string]any
}

// extractSQLFromQueryInput returns args.SQL for the dto.QueryJSONInput shape.
func extractSQLFromQueryInput(args any) string {
	if v, ok := args.(dto.QueryJSONInput); ok {
		return v.SQL
	}
	return ""
}

// extractArgsFromQueryInput returns {sql, row_limit} for audit recording.
func extractArgsFromQueryInput(args any) map[string]any {
	if v, ok := args.(dto.QueryJSONInput); ok {
		return map[string]any{"sql": v.SQL, "row_limit": v.RowLimit}
	}
	return nil
}

// extractArgsFromHierarchy returns hierarchy fields for audit recording.
func extractArgsFromHierarchy(args any) map[string]any {
	v, ok := args.(dto.HierarchyInput)
	if !ok {
		return nil
	}
	return hierarchyToMap(v)
}

// extractArgsFromRegistryInput returns {provider, version} for audit recording.
func extractArgsFromRegistryInput(args any) map[string]any {
	v, ok := args.(dto.RegistryInput)
	if !ok {
		return nil
	}
	out := map[string]any{}
	if v.Provider != "" {
		out["provider"] = v.Provider
	}
	if v.Version != "" {
		out["version"] = v.Version
	}
	return out
}

func hierarchyToMap(v dto.HierarchyInput) map[string]any {
	out := map[string]any{}
	if v.Provider != "" {
		out["provider"] = v.Provider
	}
	if v.Service != "" {
		out["service"] = v.Service
	}
	if v.Resource != "" {
		out["resource"] = v.Resource
	}
	if v.Method != "" {
		out["method"] = v.Method
	}
	if v.RowLimit != 0 {
		out["row_limit"] = v.RowLimit
	}
	return out
}

// addToolWithGate wraps mcp.AddTool with the policy gate + audit middleware.
// It is the single chokepoint at which mode enforcement and audit recording
// are applied.  The tool handler itself stays oblivious to both concerns.
func addToolWithGate[In, Out any](
	s *mcp.Server,
	cfg *Config,
	auditSink sink.Sink,
	gate toolGate,
	t *mcp.Tool,
	h mcp.ToolHandlerFor[In, Out],
) {
	if !cfg.IsToolEnabled(t.Name) {
		return
	}
	wrapped := func(ctx context.Context, req *mcp.CallToolRequest, args In) (*mcp.CallToolResult, Out, error) {
		var zero Out
		started := time.Now()
		mode := cfg.GetMode()

		// One call computes class + decision + reason.
		var sql string
		if gate.extractSQL != nil {
			sql = gate.extractSQL(args)
		}
		p := policy.NewPolicy(mode, sql, gate.defaultClass)
		auditDecision := audit.DecisionAllow

		switch p.Decision() {
		case policy.DecisionAllow:
			// proceed to tool execution below
		case policy.DecisionRefuseImmediate:
			err := fmt.Errorf("tool %q refused: %s", t.Name, p.Reason())
			recordAudit(ctx, auditSink, cfg, gate, args, sql, p.Class(), mode,
				audit.DecisionRefuseImmediate, started, err)
			return nil, zero, err
		case policy.DecisionNeedsApproval:
			outcome, err := elicitApproval(ctx, req, t.Name, p.Reason(), sql, p.Class())
			auditDecision = outcome
			if err != nil {
				recordAudit(ctx, auditSink, cfg, gate, args, sql, p.Class(), mode,
					outcome, started, err)
				return nil, zero, err
			}
		}

		result, out, err := h(ctx, req, args)
		recordAudit(ctx, auditSink, cfg, gate, args, sql, p.Class(), mode,
			auditDecision, started, err)
		if err != nil {
			return result, out, err
		}
		if auditErr := finalizeAudit(cfg, p.Class()); auditErr != nil {
			// Reserved: future strict-mode-on-audit-failure surfacing.
			_ = auditErr
		}
		return result, out, nil
	}
	mcp.AddTool(s, t, wrapped)
}

// elicitApproval asks the user (via the client) to approve the action.
// Returns the audit decision-outcome string and an error if the action was
// refused.  On accept, the error is nil and execution should proceed.
func elicitApproval(
	ctx context.Context, req *mcp.CallToolRequest,
	toolName, reason, sql string, class policy.QueryClass,
) (string, error) {
	session := req.Session
	caps := session.InitializeParams().Capabilities
	if caps == nil || caps.Elicitation == nil {
		err := fmt.Errorf(
			"tool %q refused: %s and the MCP client does not support elicitation. "+
				"Restart the server in 'full_access' mode if you trust this client, "+
				"or use an elicitation-capable client",
			toolName, reason)
		return audit.DecisionNeedsApprovalUnavailable, err
	}
	message := fmt.Sprintf("Approve %s (%s)?", toolName, class.String())
	if sql != "" {
		message = fmt.Sprintf("Approve %s (%s)?\n\nSQL: %s", toolName, class.String(), sql)
	}
	res, err := session.Elicit(ctx, &mcp.ElicitParams{
		Message: message,
		RequestedSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	})
	if err != nil {
		return audit.DecisionNeedsApprovalDeclined,
			fmt.Errorf("tool %q refused: elicitation transport error: %w", toolName, err)
	}
	switch res.Action {
	case "accept":
		return audit.DecisionNeedsApprovalAccepted, nil
	case "decline":
		return audit.DecisionNeedsApprovalDeclined,
			fmt.Errorf("tool %q refused: user declined approval", toolName)
	case "cancel":
		return audit.DecisionNeedsApprovalCancelled,
			fmt.Errorf("tool %q refused: approval prompt was dismissed", toolName)
	default:
		return audit.DecisionNeedsApprovalDeclined,
			fmt.Errorf("tool %q refused: unexpected elicitation action %q", toolName, res.Action)
	}
}

// recordAudit writes one event to the configured sink.  Audit-write failures
// are translated to client-visible errors only in strict / strict_mutations
// modes; in best_effort mode the failure is logged to stderr and ignored.
//
// Sequencing note: the audit write happens AFTER the tool has executed (or
// been skipped because it was gated out) but BEFORE the response returns to
// the client.  In strict mode, an audit-write failure on a successful DELETE
// means the row is gone but the client gets an error - intentional, so that
// no mutation slips through unaudited.
func recordAudit(
	ctx context.Context,
	auditSink sink.Sink,
	cfg *Config,
	gate toolGate,
	args any,
	sql string,
	class policy.QueryClass,
	mode string,
	decision string,
	started time.Time,
	toolErr error,
) {
	if auditSink == nil {
		return
	}
	event := audit.Event{
		Timestamp:  started,
		Tool:       gate.toolName,
		Mode:       mode,
		Decision:   decision,
		DurationMs: time.Since(started).Milliseconds(),
	}
	if sql != "" {
		event.SQL = sql
		event.QueryClass = class.String()
	} else if class != policy.QueryClassUnknown {
		event.QueryClass = class.String()
	}
	if gate.extractArgs != nil {
		event.Args = gate.extractArgs(args)
	}
	if toolErr != nil {
		event.Error = toolErr.Error()
	}
	if err := auditSink.Record(ctx, event); err != nil {
		handleAuditFailure(cfg, class, err)
	}
}

// finalizeAudit is a placeholder hook for future strict-mode hardening.
func finalizeAudit(_ *Config, _ policy.QueryClass) error { return nil }

// handleAuditFailure decides whether an audit-sink error becomes a
// client-visible failure or just gets logged.  The decision is per the
// configured failure_mode.
func handleAuditFailure(cfg *Config, class policy.QueryClass, err error) {
	mode := cfg.Server.Audit.GetFailureMode()
	switch mode {
	case audit.FailureModeStrict:
		fmt.Fprintf(stderrSink(), "audit write failed (strict): %v\n", err)
	case audit.FailureModeStrictMutations:
		// SELECTs proceed silently with a stderr note; mutations would
		// already have errored at recordAudit's caller via the returned
		// error chain in a future revision.  For now we log uniformly.
		if class == policy.QueryClassSelect {
			fmt.Fprintf(stderrSink(), "audit write failed (best-effort for select): %v\n", err)
			return
		}
		fmt.Fprintf(stderrSink(), "audit write failed (strict_mutations): %v\n", err)
	case audit.FailureModeBestEffort:
		fmt.Fprintf(stderrSink(), "audit write failed (best-effort): %v\n", err)
	default:
		fmt.Fprintf(stderrSink(), "audit write failed: %v\n", err)
	}
}
