package audit //nolint:testpackage // exercise the mapping directly

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

func attrKeys(attrs []sink.OTelAttribute) []string {
	keys := make([]string, 0, len(attrs))
	for _, a := range attrs {
		keys = append(keys, a.Key)
	}
	return keys
}

func attrString(t *testing.T, attrs []sink.OTelAttribute, key string) string {
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
func TestEventOTelLogRecords_AttributeSchema(t *testing.T) {
	records := sampleEvent().OTelLogRecords()
	if len(records) != 1 {
		t.Fatalf("an allowed call is one record, got %d", len(records))
	}
	rec := records[0]
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
		attrString(t, rec.Attributes, "stackql.duration_ms") != "42" ||
		attrString(t, rec.Attributes, "stackql.provider") != "google" {
		t.Fatalf("attribute values: %+v", rec.Attributes)
	}
	if rec.Body != "execute_tool run_select_query" || rec.Severity != sink.OTelSeverityInfo {
		t.Fatalf("body/severity: %+v", rec)
	}
	if !rec.Time.Equal(sampleEvent().Timestamp) || rec.CorrelationKey != "s1" {
		t.Fatalf("time / correlation: %+v", rec)
	}
	if id := attrString(t, rec.Attributes, "gen_ai.tool.call.id"); len(id) != 2*callIDBytes {
		t.Fatalf("call id must be %d hex chars: %q", 2*callIDBytes, id)
	}
}

func TestEventOTelLogRecords_ElicitationDecisionAndError(t *testing.T) {
	ev := sampleEvent()
	ev.Tool = "run_mutation_query"
	ev.Decision = DecisionNeedsApprovalDeclined
	ev.Error = `tool "run_mutation_query" refused: user declined approval`
	ev.Wire.RowsReturned = -1
	ev.Wire.TraceParent = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	records := ev.OTelLogRecords()
	if len(records) != 2 {
		t.Fatalf("a gated call is two records (invocation + decision), got %d", len(records))
	}
	call, decision := records[0], records[1]
	if call.Severity != sink.OTelSeverityError || attrString(t, call.Attributes, "error.type") != DecisionNeedsApprovalDeclined ||
		!strings.Contains(attrString(t, call.Attributes, "error.message"), "declined") {
		t.Fatalf("refusal must be an ERROR record typed by decision: %+v", call)
	}
	if attrString(t, decision.Attributes, "mcp.method.name") != "elicitation/create" ||
		attrString(t, decision.Attributes, "stackql.decision") != DecisionNeedsApprovalDeclined ||
		decision.Body != "elicitation run_mutation_query" {
		t.Fatalf("decision record: %+v", decision)
	}
	if attrString(t, call.Attributes, "gen_ai.tool.call.id") != attrString(t, decision.Attributes, "gen_ai.tool.call.id") {
		t.Fatal("records of one call must share gen_ai.tool.call.id")
	}
	for _, r := range records {
		if r.TraceParent != ev.Wire.TraceParent || r.CorrelationKey != "s1" {
			t.Fatalf("trace context must reach every record: %+v", r)
		}
	}
	for _, a := range call.Attributes {
		if a.Key == "stackql.rows_returned" {
			t.Fatal("stackql.rows_returned must be absent when no rows were produced")
		}
	}
	past := sampleEvent()
	past.Error = "upstream 500"
	if got := attrString(t, past.OTelLogRecords()[0].Attributes, "error.type"); got != "tool_error" {
		t.Fatalf("failures past the gate are tool_error, got %q", got)
	}
}

// Result values never reach the log in either format: a RETURNING secret
// appears in neither the JSONL event nor the OTLP records, and the JSONL
// shape carries none of the wire context.
func TestNewOTelSink_RedactionParityWithJSONL(t *testing.T) {
	const secret = "s3cr3t-value-from-returning"
	capture := &captureSink{}
	otel := NewOTelSink(capture, "0.10.606")
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
	for _, want := range []string{
		`{"resourceLogs":[{"resource":{"attributes":[{"key":"service.name","value":{"stringValue":"stackql"}},{"key":"service.version","value":{"stringValue":"0.10.606"}}]}`,
		`"scope":{"name":"` + scopeName + `","version":"` + AttributeSchemaVersion + `"}`,
		`"schemaUrl":"` + sink.DefaultOTelSchemaURL + `"`,
	} {
		if !strings.Contains(string(otelLine), want) {
			t.Fatalf("otel line lacks %s:\n%s", want, otelLine)
		}
	}
}
