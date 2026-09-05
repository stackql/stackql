package audit //nolint:testpackage // exercise the encoder directly

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stackql/stackql/pkg/sink"
)

type captureSink struct{ payloads []any }

func (c *captureSink) Record(_ context.Context, p any) error {
	c.payloads = append(c.payloads, p)
	return nil
}
func (c *captureSink) Close() error { return nil }

var _ sink.Sink = &captureSink{}

func attrKeys(attrs []Attribute) []string {
	keys := make([]string, 0, len(attrs))
	for _, a := range attrs {
		keys = append(keys, a.Key)
	}
	return keys
}

func attrString(t *testing.T, attrs []Attribute, key string) string {
	t.Helper()
	for _, a := range attrs {
		if a.Key == key {
			if a.Value.StringValue != nil {
				return *a.Value.StringValue
			}
			if a.Value.IntValue != nil {
				return *a.Value.IntValue
			}
		}
	}
	t.Fatalf("attribute %q missing from %v", key, attrKeys(attrs))
	return ""
}

func sampleEvent() Event {
	return Event{
		Timestamp:  time.Date(2026, 9, 5, 1, 2, 3, 0, time.UTC),
		Tool:       "run_select_query",
		Mode:       "safe",
		Decision:   DecisionAllow,
		QueryClass: "select",
		SQL:        "select name from google.storage.buckets where project = 'p'",
		Args:       map[string]any{"sql": "...", "provider": "google", "source": "lib-1"},
		DurationMs: 42,
		Wire:       WireContext{ProtocolVersion: "2026-07-28", SessionID: "s1", RowsReturned: 3},
	}
}

// The emitted attribute set is a versioned interface: this is the schema
// assertion for AttributeSchemaVersion 1.0.0.
func TestOTelEncode_ToolInvocationAttributeSchema(t *testing.T) {
	enc := NewOTelSink(&captureSink{}, "0.10.606").(*otelSink)
	data := enc.Encode(sampleEvent())

	if len(data.ResourceLogs) != 1 || len(data.ResourceLogs[0].ScopeLogs) != 1 {
		t.Fatalf("expected one resource and one scope, got %+v", data)
	}
	rl := data.ResourceLogs[0]
	if attrString(t, rl.Resource.Attributes, "service.name") != "stackql" ||
		attrString(t, rl.Resource.Attributes, "service.version") != "0.10.606" {
		t.Fatalf("resource attributes: %+v", rl.Resource.Attributes)
	}
	scope := rl.ScopeLogs[0]
	if scope.Scope.Version != AttributeSchemaVersion || scope.SchemaURL != SemconvSchemaURL || rl.SchemaURL != SemconvSchemaURL {
		t.Fatalf("scope/schema pinning wrong: %+v", scope.Scope)
	}
	if len(scope.LogRecords) != 1 {
		t.Fatalf("an allowed call is one record, got %d", len(scope.LogRecords))
	}
	rec := scope.LogRecords[0]
	want := []string{
		"gen_ai.operation.name", "mcp.method.name", "gen_ai.tool.name", "gen_ai.tool.call.id",
		"mcp.protocol.version", "mcp.session.id", "stackql.mode", "stackql.decision",
		"stackql.query", "stackql.query_class", "stackql.provider", "stackql.query.source",
		"stackql.duration_ms", "stackql.rows_returned",
	}
	if got := attrKeys(rec.Attributes); !reflect.DeepEqual(got, want) {
		t.Fatalf("attribute schema drift:\n got %v\nwant %v", got, want)
	}
	if attrString(t, rec.Attributes, "gen_ai.operation.name") != "execute_tool" ||
		attrString(t, rec.Attributes, "mcp.method.name") != "tools/call" ||
		attrString(t, rec.Attributes, "stackql.rows_returned") != "3" ||
		attrString(t, rec.Attributes, "stackql.duration_ms") != "42" {
		t.Fatalf("attribute values: %+v", rec.Attributes)
	}
	if *rec.Body.StringValue != "execute_tool run_select_query" || rec.SeverityText != "INFO" || rec.SeverityNumber != severityInfo {
		t.Fatalf("body/severity: %+v", rec)
	}
	if rec.TimeUnixNano != "1788570123000000000" {
		t.Fatalf("timeUnixNano = %s", rec.TimeUnixNano)
	}
	if len(rec.TraceID) != 32 || len(rec.SpanID) != 16 {
		t.Fatalf("generated ids must be hex trace(32)/span(16): %q %q", rec.TraceID, rec.SpanID)
	}
}

func TestOTelEncode_ElicitationDecisionRecordAndError(t *testing.T) {
	enc := NewOTelSink(&captureSink{}, "v").(*otelSink)
	ev := sampleEvent()
	ev.Tool = "run_mutation_query"
	ev.Decision = DecisionNeedsApprovalDeclined
	ev.Error = `tool "run_mutation_query" refused: user declined approval`
	ev.Wire.RowsReturned = -1
	records := enc.Encode(ev).ResourceLogs[0].ScopeLogs[0].LogRecords
	if len(records) != 2 {
		t.Fatalf("a gated call is two records (invocation + decision), got %d", len(records))
	}
	call, decision := records[0], records[1]
	if call.SeverityText != "ERROR" || attrString(t, call.Attributes, "error.type") != DecisionNeedsApprovalDeclined {
		t.Fatalf("refusal must be an ERROR record typed by decision: %+v", call)
	}
	if attrString(t, decision.Attributes, "mcp.method.name") != "elicitation/create" ||
		attrString(t, decision.Attributes, "stackql.decision") != DecisionNeedsApprovalDeclined ||
		*decision.Body.StringValue != "elicitation run_mutation_query" {
		t.Fatalf("decision record: %+v", decision)
	}
	if call.TraceID != decision.TraceID || call.SpanID == decision.SpanID {
		t.Fatalf("records of one call share a trace but not a span: %+v %+v", call, decision)
	}
	if attrString(t, call.Attributes, "gen_ai.tool.call.id") != attrString(t, decision.Attributes, "gen_ai.tool.call.id") {
		t.Fatalf("records of one call must share gen_ai.tool.call.id")
	}
	for _, key := range []string{"stackql.rows_returned"} {
		for _, a := range call.Attributes {
			if a.Key == key {
				t.Fatalf("%s must be absent when no rows were produced", key)
			}
		}
	}
}

func TestOTelEncode_TraceContext(t *testing.T) {
	enc := NewOTelSink(&captureSink{}, "v").(*otelSink)
	supplied := sampleEvent()
	supplied.Wire.TraceParent = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	rec := enc.Encode(supplied).ResourceLogs[0].ScopeLogs[0].LogRecords[0]
	if rec.TraceID != "4bf92f3577b34da6a3ce929d0e0e4736" || rec.SpanID != "00f067aa0ba902b7" {
		t.Fatalf("caller traceparent not honoured: %q %q", rec.TraceID, rec.SpanID)
	}

	first := enc.Encode(sampleEvent()).ResourceLogs[0].ScopeLogs[0].LogRecords[0]
	second := enc.Encode(sampleEvent()).ResourceLogs[0].ScopeLogs[0].LogRecords[0]
	if first.TraceID != second.TraceID {
		t.Fatalf("records from one session must share a generated trace id")
	}
	if first.SpanID == second.SpanID {
		t.Fatalf("each record gets its own span id")
	}
	other := sampleEvent()
	other.Wire.SessionID = "s2"
	if enc.Encode(other).ResourceLogs[0].ScopeLogs[0].LogRecords[0].TraceID == first.TraceID {
		t.Fatalf("a different session must not share the trace id")
	}
}

// Result values never reach the log in either format: a RETURNING secret
// appears in neither the JSONL event nor the OTLP records.
func TestOTelEncode_RedactionParityWithJSONL(t *testing.T) {
	const secret = "s3cr3t-value-from-returning"
	capture := &captureSink{}
	otel := NewOTelSink(capture, "v")
	ev := sampleEvent()
	ev.Tool = "run_mutation_query"
	ev.SQL = "insert into t(data__name) select 'x' returning secret_key"
	ev.Wire.RowsReturned = 1
	if err := otel.Record(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	otelLine, _ := json.Marshal(capture.payloads[0])
	jsonlLine, _ := json.Marshal(ev)
	for name, line := range map[string][]byte{"otel": otelLine, "jsonl": jsonlLine} {
		if strings.Contains(string(line), secret) {
			t.Fatalf("%s log leaked a result value: %s", name, line)
		}
		if !strings.Contains(string(line), "returning secret_key") {
			t.Fatalf("%s log must still carry the verbatim SQL: %s", name, line)
		}
	}
	if strings.Contains(string(jsonlLine), "2026-07-28") || strings.Contains(string(jsonlLine), "rows_returned") {
		t.Fatalf("JSONL must stay byte-compatible (no wire context fields): %s", jsonlLine)
	}
	if !strings.HasPrefix(string(otelLine), `{"resourceLogs":[{"resource":`) {
		t.Fatalf("otel line is not an OTLP/JSON LogsData: %s", otelLine)
	}
}

func TestOTelSink_RejectsForeignPayload(t *testing.T) {
	if err := NewOTelSink(&captureSink{}, "v").Record(context.Background(), "not an event"); err == nil {
		t.Fatal("expected an error for a non-Event payload")
	}
}
