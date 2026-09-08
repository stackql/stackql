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

// extractArgsFromQueryInput returns {sql, row_limit} plus the optional query
// library source attribution for audit recording.
func extractArgsFromQueryInput(args any) map[string]any {
	if v, ok := args.(dto.QueryJSONInput); ok {
		out := map[string]any{"sql": v.SQL, "row_limit": v.RowLimit}
		if v.Source != "" {
			out["source"] = v.Source
		}
		return out
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

// boolPtr returns a pointer for the *bool hint fields on mcp.ToolAnnotations.
func boolPtr(v bool) *bool { return &v }

// deriveToolAnnotations defaults behavioural hints from the gate
// classification so hint and enforcement share one source of truth.  Only
// tools statically classified as selects (no SQL input) claim read-only;
// SQL-carrying tools make no claim because their effect depends on the
// submitted statement and enforcement stays with the policy gate.  An
// explicit Annotations value on the tool always wins.
func deriveToolAnnotations(gate toolGate) *mcp.ToolAnnotations {
	if gate.defaultClass == policy.QueryClassSelect && gate.extractSQL == nil {
		return &mcp.ToolAnnotations{ReadOnlyHint: true}
	}
	return nil
}

// addToolWithGate wraps mcp.AddTool with the policy gate + audit middleware.
// It is the single chokepoint at which mode enforcement and audit recording
// are applied, and where the tool description mastered under content/tools
// is attached.  The tool handler itself stays oblivious to these concerns.
func addToolWithGate[In, Out any](
	s *mcp.Server,
	cfg *Config,
	auditSink sink.Sink,
	descriptions toolDescriptions,
	gate toolGate,
	t *mcp.Tool,
	h mcp.ToolHandlerFor[In, Out],
) error {
	if !cfg.IsToolEnabled(t.Name) {
		return nil
	}
	description, hasDescription := descriptions.describe(t.Name)
	if !hasDescription {
		return fmt.Errorf("tool %q has no description under %s", t.Name, embeddedToolsDir)
	}
	t.Description = description
	if t.Annotations == nil {
		t.Annotations = deriveToolAnnotations(gate)
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
				audit.DecisionRefuseImmediate, started, err, wireContext(req, nil))
			return nil, zero, err
		case policy.DecisionNeedsApproval:
			outcome, prompt, err := approvalOutcome(req, t.Name, p.Reason(), sql, p.Class())
			if prompt != nil {
				// First pass: hand the approval prompt back as an input
				// request; the client (or the SDK, for pre-2026-07-28
				// clients) retries the call with the user's answer.
				return prompt, zero, nil
			}
			auditDecision = outcome
			if err != nil {
				recordAudit(ctx, auditSink, cfg, gate, args, sql, p.Class(), mode,
					outcome, started, err, wireContext(req, nil))
				return nil, zero, err
			}
		}

		result, out, err := h(ctx, req, args)
		var produced any
		if err == nil {
			produced = out
		}
		recordAudit(ctx, auditSink, cfg, gate, args, sql, p.Class(), mode,
			auditDecision, started, err, wireContext(req, produced))
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
	return nil
}

// approvalInputKey is the input request id the client echoes back with the
// user's answer (SEP-2322 multi round-trip requests).
const approvalInputKey = "stackql_approval"

// approvalOutcome resolves the gated-write approval as a multi round-trip
// request. On the first pass it returns the elicitation prompt for the caller
// to hand back as an input-required result; on the retry it reads the user's
// answer from InputResponses and returns the audit decision-outcome plus an
// error when the action was refused. The SDK middleware performs the round
// trip on the server side for pre-2026-07-28 clients, so both lifecycle
// models share this path.
func approvalOutcome(
	req *mcp.CallToolRequest,
	toolName, reason, sql string, class policy.QueryClass,
) (string, *mcp.CallToolResult, error) {
	if answer, answered := req.Params.InputResponses[approvalInputKey]; answered {
		res, isElicit := answer.(*mcp.ElicitResult)
		if !isElicit {
			return audit.DecisionNeedsApprovalDeclined,
				nil, fmt.Errorf("tool %q refused: unexpected approval response type %T", toolName, answer)
		}
		switch res.Action {
		case "accept":
			return audit.DecisionNeedsApprovalAccepted, nil, nil
		case "decline":
			return audit.DecisionNeedsApprovalDeclined,
				nil, fmt.Errorf("tool %q refused: user declined approval", toolName)
		case "cancel":
			return audit.DecisionNeedsApprovalCancelled,
				nil, fmt.Errorf("tool %q refused: approval prompt was dismissed", toolName)
		default:
			return audit.DecisionNeedsApprovalDeclined,
				nil, fmt.Errorf("tool %q refused: unexpected elicitation action %q", toolName, res.Action)
		}
	}
	// Capabilities come from the request _meta (2026-07-28) or the
	// initialize params (earlier revisions); the SDK resolves both.
	caps := req.ClientCapabilities()
	if caps == nil || caps.Elicitation == nil {
		err := fmt.Errorf(
			"tool %q refused: %s and the MCP client does not support elicitation. "+
				"Restart the server in 'full_access' mode if you trust this client, "+
				"or use an elicitation-capable client",
			toolName, reason)
		return audit.DecisionNeedsApprovalUnavailable, nil, err
	}
	message := fmt.Sprintf("Approve %s (%s)?", toolName, class.String())
	if sql != "" {
		message = fmt.Sprintf("Approve %s (%s)?\n\nSQL: %s", toolName, class.String(), sql)
	}
	return "", &mcp.CallToolResult{
		InputRequests: mcp.InputRequestMap{
			approvalInputKey: &mcp.ElicitParams{
				Message: message,
				RequestedSchema: map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				},
			},
		},
	}, nil
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
	wire audit.WireContext,
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
		Wire:       wire,
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

// wireContext collects the transport facts the OTel encoding records: the
// negotiated revision, the session, a caller-supplied traceparent from
// _meta, and the row count for row-returning tools.
func wireContext(req *mcp.CallToolRequest, out any) audit.WireContext {
	wire := audit.WireContext{RowsReturned: -1}
	if req == nil {
		return wire
	}
	wire.ProtocolVersion = req.ProtocolVersion()
	if req.Session != nil {
		wire.SessionID = req.Session.ID()
	}
	if req.Params != nil {
		if tp, ok := req.Params.Meta["traceparent"].(string); ok {
			wire.TraceParent = tp
		}
	}
	if rows, ok := out.(dto.QueryResultDTO); ok {
		wire.RowsReturned = len(rows.Rows)
	}
	return wire
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
