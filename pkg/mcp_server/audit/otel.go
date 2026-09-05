package audit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/stackql/stackql/pkg/sink"
)

// Log formats accepted by audit.format / --mcp.log.format.
const (
	FormatJSONL = "jsonl"
	FormatOTel  = "otel"
)

// The emitted attribute set is a versioned interface (issue #729): bump
// AttributeSchemaVersion when it changes and update the schema test.
const (
	// AttributeSchemaVersion is recorded as the instrumentation scope version.
	AttributeSchemaVersion = "1.0.0"
	// SemconvSchemaURL pins the OpenTelemetry semantic conventions release the
	// stable attributes (error.*, service.*) are taken from. The GenAI / MCP
	// conventions (gen_ai.*, mcp.*) are Development status in that release.
	SemconvSchemaURL = "https://opentelemetry.io/schemas/1.44.0"
	scopeName        = "github.com/stackql/stackql/pkg/mcp_server/audit"
	serviceName      = "stackql"

	// OTel log severity numbers (INFO = 9, ERROR = 17).
	severityInfo  = 9
	severityError = 17

	traceIDBytes = 16
	spanIDBytes  = 8
	// maxTrackedSessions bounds the session -> trace id map on long-lived servers.
	maxTrackedSessions = 1024
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

// Attribute is one OTLP/JSON key/value pair; exactly one Value field is set.
type Attribute struct {
	Key   string   `json:"key"`
	Value AnyValue `json:"value"`
}

// AnyValue is the OTLP/JSON encoding of an attribute value.
type AnyValue struct {
	StringValue *string `json:"stringValue,omitempty"`
	IntValue    *string `json:"intValue,omitempty"`
	BoolValue   *bool   `json:"boolValue,omitempty"`
}

// LogRecord is an OTLP/JSON LogRecord.
type LogRecord struct {
	TimeUnixNano         string      `json:"timeUnixNano"`
	ObservedTimeUnixNano string      `json:"observedTimeUnixNano"`
	SeverityNumber       int         `json:"severityNumber"`
	SeverityText         string      `json:"severityText"`
	Body                 AnyValue    `json:"body"`
	Attributes           []Attribute `json:"attributes"`
	TraceID              string      `json:"traceId"`
	SpanID               string      `json:"spanId"`
}

type instrumentationScope struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type scopeLogs struct {
	Scope      instrumentationScope `json:"scope"`
	LogRecords []LogRecord          `json:"logRecords"`
	SchemaURL  string               `json:"schemaUrl"`
}

type resource struct {
	Attributes []Attribute `json:"attributes"`
}

type resourceLogs struct {
	Resource  resource    `json:"resource"`
	ScopeLogs []scopeLogs `json:"scopeLogs"`
	SchemaURL string      `json:"schemaUrl"`
}

// LogsData is one OTLP/JSON logs export payload; the file sink writes one
// per line, the shape the collector's otlp_json_file receiver ingests as-is.
type LogsData struct {
	ResourceLogs []resourceLogs `json:"resourceLogs"`
}

// otelSink maps each Event onto OTLP/JSON log records and forwards them to
// the wrapped sink.
type otelSink struct {
	inner          sink.Sink
	serviceVersion string
	now            func() time.Time

	mu       sync.Mutex
	traceIDs map[string]string // session id -> trace id
}

// NewOTelSink wraps inner so every recorded Event is written as OTLP/JSON.
func NewOTelSink(inner sink.Sink, serviceVersion string) sink.Sink {
	return &otelSink{inner: inner, serviceVersion: serviceVersion, now: time.Now, traceIDs: map[string]string{}}
}

func (s *otelSink) Record(ctx context.Context, payload any) error {
	event, isEvent := payload.(Event)
	if !isEvent {
		return fmt.Errorf("otel audit sink: unsupported payload %T", payload)
	}
	return s.inner.Record(ctx, s.Encode(event))
}

func (s *otelSink) Close() error { return s.inner.Close() }

// Encode renders one Event as a LogsData: a record for the tool invocation
// and, when the call went through the approval gate, one for the elicitation
// decision.
func (s *otelSink) Encode(event Event) LogsData {
	traceID, spanID := s.traceContext(event.Wire)
	callID := newID(spanIDBytes)
	observed := strconv(s.now().UnixNano())
	started := strconv(event.Timestamp.UnixNano())

	common := []Attribute{
		str("gen_ai.tool.name", event.Tool),
		str("gen_ai.tool.call.id", callID),
		str("mcp.protocol.version", event.Wire.ProtocolVersion),
		str("mcp.session.id", event.Wire.SessionID),
		str("stackql.mode", event.Mode),
		str("stackql.decision", event.Decision),
	}
	invocation := append([]Attribute{
		str("gen_ai.operation.name", "execute_tool"),
		str("mcp.method.name", "tools/call"),
	}, common...)
	invocation = append(invocation,
		str("stackql.query", event.SQL),
		str("stackql.query_class", event.QueryClass),
		str("stackql.provider", argString(event.Args, "provider")),
		str("stackql.query.source", argString(event.Args, "source")),
		integer("stackql.duration_ms", event.DurationMs),
	)
	if event.Wire.RowsReturned >= 0 {
		invocation = append(invocation, integer("stackql.rows_returned", int64(event.Wire.RowsReturned)))
	}
	severity, severityText := severityInfo, "INFO"
	if event.Error != "" {
		severity, severityText = severityError, "ERROR"
		invocation = append(invocation,
			str("error.type", errorType(event.Decision)),
			str("error.message", event.Error),
		)
	}
	records := []LogRecord{{
		TimeUnixNano:         started,
		ObservedTimeUnixNano: observed,
		SeverityNumber:       severity,
		SeverityText:         severityText,
		Body:                 stringValue("execute_tool " + event.Tool),
		Attributes:           compact(invocation),
		TraceID:              traceID,
		SpanID:               spanID,
	}}
	if strings.HasPrefix(event.Decision, "needs_approval_") {
		decision := append([]Attribute{str("mcp.method.name", "elicitation/create")}, common...)
		records = append(records, LogRecord{
			TimeUnixNano:         started,
			ObservedTimeUnixNano: observed,
			SeverityNumber:       severityInfo,
			SeverityText:         "INFO",
			Body:                 stringValue("elicitation " + event.Tool),
			Attributes:           compact(decision),
			TraceID:              traceID,
			SpanID:               newID(spanIDBytes),
		})
	}
	return LogsData{ResourceLogs: []resourceLogs{{
		Resource: resource{Attributes: compact([]Attribute{
			str("service.name", serviceName),
			str("service.version", s.serviceVersion),
		})},
		ScopeLogs: []scopeLogs{{
			Scope:      instrumentationScope{Name: scopeName, Version: AttributeSchemaVersion},
			LogRecords: records,
			SchemaURL:  SemconvSchemaURL,
		}},
		SchemaURL: SemconvSchemaURL,
	}}}
}

// traceContext honours a caller-supplied W3C traceparent, otherwise reuses
// one generated trace id per session so an agent session's records correlate.
func (s *otelSink) traceContext(wire WireContext) (string, string) {
	if parts := strings.Split(wire.TraceParent, "-"); len(parts) == 4 &&
		len(parts[1]) == 2*traceIDBytes && len(parts[2]) == 2*spanIDBytes {
		return parts[1], parts[2]
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	traceID, known := s.traceIDs[wire.SessionID]
	if !known || wire.SessionID == "" {
		if len(s.traceIDs) >= maxTrackedSessions {
			s.traceIDs = map[string]string{}
		}
		traceID = newID(traceIDBytes)
		s.traceIDs[wire.SessionID] = traceID
	}
	return traceID, newID(spanIDBytes)
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

func newID(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return strings.Repeat("0", 2*n)
	}
	return hex.EncodeToString(b)
}

func strconv(v int64) string { return fmt.Sprintf("%d", v) }

func stringValue(v string) AnyValue { return AnyValue{StringValue: &v} }

func str(key, v string) Attribute { return Attribute{Key: key, Value: stringValue(v)} }

func integer(key string, v int64) Attribute {
	s := strconv(v)
	return Attribute{Key: key, Value: AnyValue{IntValue: &s}}
}

func argString(args map[string]any, key string) string {
	if v, ok := args[key].(string); ok {
		return v
	}
	return ""
}

// compact drops attributes with an empty string value.
func compact(attrs []Attribute) []Attribute {
	out := attrs[:0]
	for _, a := range attrs {
		if a.Value.StringValue != nil && *a.Value.StringValue == "" {
			continue
		}
		out = append(out, a)
	}
	return out
}
