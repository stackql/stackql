package audit

import (
	"strings"

	"github.com/stackql/stackql/pkg/sink"
)

// Log formats accepted by audit.format / --mcp.log.format.
const (
	FormatJSONL = "jsonl"
	FormatOTel  = "otel"
)

// The emitted attribute set is a versioned interface (issue #729): bump
// AttributeSchemaVersion when it changes and update the schema test. The
// gen_ai.* / mcp.* conventions are Development status in the pinned release.
const (
	AttributeSchemaVersion = "1.0.0"
	scopeName              = "github.com/stackql/stackql/pkg/mcp_server/audit"
	serviceName            = "stackql"
	callIDBytes            = 8
)

// WireContext carries per-call transport facts that only the OTel encoding
// serialises; the JSONL Event stays byte-compatible.
type WireContext struct {
	ProtocolVersion string
	SessionID       string
	// TraceParent is the W3C traceparent supplied by the caller in _meta.
	TraceParent string
	// RowsReturned is -1 when the tool does not return rows.
	RowsReturned int
}

// NewOTelSink decorates inner with the OTLP/JSON encoding under the stackql
// resource and this package's attribute schema scope.
func NewOTelSink(inner sink.Sink, serviceVersion string) sink.Sink {
	return sink.NewOTelSink(
		inner,
		sink.OTelResource{ServiceName: serviceName, ServiceVersion: serviceVersion},
		sink.OTelScope{Name: scopeName, Version: AttributeSchemaVersion},
	)
}

// OTelLogRecords maps the event onto the GenAI / MCP semantic conventions:
// one record for the tool invocation and, when the call went through the
// approval gate, one for the elicitation decision. Records of one call share
// gen_ai.tool.call.id and correlate on the session.
func (e Event) OTelLogRecords() []sink.OTelLogRecord {
	callID := sink.OTelRandomID(callIDBytes)
	common := []sink.OTelAttribute{
		sink.OTelString("gen_ai.tool.name", e.Tool),
		sink.OTelString("gen_ai.tool.call.id", callID),
		sink.OTelString("mcp.protocol.version", e.Wire.ProtocolVersion),
		sink.OTelString("mcp.session.id", e.Wire.SessionID),
		sink.OTelString("stackql.mode", e.Mode),
		sink.OTelString("stackql.decision", e.Decision),
	}
	invocation := append([]sink.OTelAttribute{
		sink.OTelString("gen_ai.operation.name", "execute_tool"),
		sink.OTelString("mcp.method.name", "tools/call"),
	}, common...)
	invocation = append(invocation,
		sink.OTelString("stackql.query", e.SQL),
		sink.OTelString("stackql.query_class", e.QueryClass),
		sink.OTelString("stackql.provider", argString(e.Args, "provider")),
		sink.OTelString("stackql.query.source", argString(e.Args, "source")),
		sink.OTelInt("stackql.duration_ms", e.DurationMs),
	)
	if e.Wire.RowsReturned >= 0 {
		invocation = append(invocation, sink.OTelInt("stackql.rows_returned", int64(e.Wire.RowsReturned)))
	}
	severity := sink.OTelSeverityInfo
	if e.Error != "" {
		severity = sink.OTelSeverityError
		invocation = append(invocation,
			sink.OTelString("error.type", errorType(e.Decision)),
			sink.OTelString("error.message", e.Error),
		)
	}
	records := []sink.OTelLogRecord{{
		Time:           e.Timestamp,
		Severity:       severity,
		Body:           "execute_tool " + e.Tool,
		Attributes:     invocation,
		TraceParent:    e.Wire.TraceParent,
		CorrelationKey: e.Wire.SessionID,
	}}
	if strings.HasPrefix(e.Decision, "needs_approval_") {
		records = append(records, sink.OTelLogRecord{
			Time:           e.Timestamp,
			Severity:       sink.OTelSeverityInfo,
			Body:           "elicitation " + e.Tool,
			Attributes:     append([]sink.OTelAttribute{sink.OTelString("mcp.method.name", "elicitation/create")}, common...),
			TraceParent:    e.Wire.TraceParent,
			CorrelationKey: e.Wire.SessionID,
		})
	}
	return records
}

// errorType classifies a failed call by its gate decision; execution failures
// past the gate are reported as tool errors.
func errorType(decision string) string {
	switch decision {
	case DecisionAllow, DecisionNeedsApprovalAccepted:
		return "tool_error"
	default:
		return decision
	}
}

func argString(args map[string]any, key string) string {
	if v, ok := args[key].(string); ok {
		return v
	}
	return ""
}
