package sink //nolint:testpackage // exercise the decorator directly

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

type captureSink struct {
	payloads []any
	closed   bool
}

func (c *captureSink) Record(_ context.Context, p any) error {
	c.payloads = append(c.payloads, p)
	return nil
}

func (c *captureSink) Close() error {
	c.closed = true
	return nil
}

func newTestOTelSink(capture *captureSink) *otelSink {
	s := NewOTelSink(capture,
		OTelResource{ServiceName: "svc", ServiceVersion: "1.2.3", Attributes: []OTelAttribute{OTelString("deployment.environment", "test")}},
		OTelScope{Name: "scope/test", Version: "9.9.9"},
	).(*otelSink)
	s.now = func() time.Time { return time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC) }
	return s
}

func attrKeys(attrs []OTelAttribute) []string {
	keys := make([]string, 0, len(attrs))
	for _, a := range attrs {
		keys = append(keys, a.Key)
	}
	return keys
}

func attrValue(t *testing.T, attrs []OTelAttribute, key string) OTelAnyValue {
	t.Helper()
	for _, a := range attrs {
		if a.Key == key {
			return a.Value
		}
	}
	t.Fatalf("attribute %q missing from %v", key, attrKeys(attrs))
	return OTelAnyValue{}
}

func firstRecord(t *testing.T, data OTelLogsData) OTelWireLogRecord {
	t.Helper()
	if len(data.ResourceLogs) != 1 || len(data.ResourceLogs[0].ScopeLogs) != 1 || len(data.ResourceLogs[0].ScopeLogs[0].LogRecords) == 0 {
		t.Fatalf("unexpected envelope shape: %+v", data)
	}
	return data.ResourceLogs[0].ScopeLogs[0].LogRecords[0]
}

// A record with no OTel mapping of its own is decorated generically: JSON
// body, top-level scalar fields as typed attributes in key order.
func TestOTelSink_GenericRecordDecoration(t *testing.T) {
	capture := &captureSink{}
	s := newTestOTelSink(capture)
	payload := map[string]any{
		"tool":     "run_select_query",
		"rows":     3,
		"ratio":    0.5,
		"ok":       true,
		"nested":   map[string]any{"a": 1},
		"nothing":  nil,
		"emptystr": "",
	}
	if err := s.Record(context.Background(), payload); err != nil {
		t.Fatal(err)
	}
	data, ok := capture.payloads[0].(OTelLogsData)
	if !ok {
		t.Fatalf("inner sink received %T, want OTelLogsData", capture.payloads[0])
	}
	rec := firstRecord(t, data)
	if want := []string{"nested", "ok", "ratio", "rows", "tool"}; !reflect.DeepEqual(attrKeys(rec.Attributes), want) {
		t.Fatalf("attribute keys = %v, want %v (sorted, empties dropped)", attrKeys(rec.Attributes), want)
	}
	if *attrValue(t, rec.Attributes, "rows").IntValue != "3" ||
		*attrValue(t, rec.Attributes, "ratio").DoubleValue != 0.5 ||
		!*attrValue(t, rec.Attributes, "ok").BoolValue ||
		*attrValue(t, rec.Attributes, "nested").StringValue != `{"a":1}` ||
		*attrValue(t, rec.Attributes, "tool").StringValue != "run_select_query" {
		t.Fatalf("attribute typing wrong: %+v", rec.Attributes)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(*rec.Body.StringValue), &body); err != nil || body["tool"] != "run_select_query" {
		t.Fatalf("body must be the record's JSON: %q (%v)", *rec.Body.StringValue, err)
	}
	if rec.SeverityNumber != OTelSeverityInfo || rec.SeverityText != "INFO" {
		t.Fatalf("generic records are INFO: %+v", rec)
	}
	if rec.TimeUnixNano != "1788652800000000000" || rec.ObservedTimeUnixNano != rec.TimeUnixNano {
		t.Fatalf("generic records are timestamped at observation: %s / %s", rec.TimeUnixNano, rec.ObservedTimeUnixNano)
	}
	if len(rec.TraceID) != 32 || len(rec.SpanID) != 16 {
		t.Fatalf("ids must be hex trace(32)/span(16): %q %q", rec.TraceID, rec.SpanID)
	}
}

type typedRecord struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestOTelSink_GenericStructAndScalarPayloads(t *testing.T) {
	s := newTestOTelSink(&captureSink{})
	data, err := s.Encode(typedRecord{Name: "x", Count: 2})
	if err != nil {
		t.Fatal(err)
	}
	rec := firstRecord(t, data)
	if want := []string{"count", "name"}; !reflect.DeepEqual(attrKeys(rec.Attributes), want) {
		t.Fatalf("struct fields should become attributes: %v", attrKeys(rec.Attributes))
	}
	scalar, err := s.Encode("just a line")
	if err != nil {
		t.Fatal(err)
	}
	if rec := firstRecord(t, scalar); len(rec.Attributes) != 0 || *rec.Body.StringValue != `"just a line"` {
		t.Fatalf("non-object payloads carry only a body: %+v", rec)
	}
	if _, err := s.Encode(make(chan int)); err == nil {
		t.Fatal("unmarshalable payloads must error")
	}
}

type mappedRecord struct {
	at   time.Time
	key  string
	tp   string
	fail bool
}

func (m mappedRecord) OTelLogRecords() []OTelLogRecord {
	severity := OTelSeverityInfo
	if m.fail {
		severity = OTelSeverityError
	}
	return []OTelLogRecord{
		{Time: m.at, Severity: severity, Body: "first", CorrelationKey: m.key, TraceParent: m.tp,
			Attributes: []OTelAttribute{OTelString("a", "1"), OTelString("empty", ""), OTelInt("n", 7)}},
		{Time: m.at, Severity: 13, Body: "second", CorrelationKey: m.key, TraceParent: m.tp},
	}
}

// A payload that maps itself is used verbatim; the sink only adds the
// envelope, observed time, severity text and trace context.
func TestOTelSink_RecorderPayloadEnvelope(t *testing.T) {
	s := newTestOTelSink(&captureSink{})
	at := time.Date(2026, 9, 5, 1, 2, 3, 0, time.UTC)
	data, err := s.Encode(mappedRecord{at: at, key: "s1", fail: true})
	if err != nil {
		t.Fatal(err)
	}
	rl := data.ResourceLogs[0]
	if rl.SchemaURL != DefaultOTelSchemaURL || rl.ScopeLogs[0].SchemaURL != DefaultOTelSchemaURL {
		t.Fatalf("schema url defaulted wrong: %+v", rl)
	}
	if want := []string{"service.name", "service.version", "deployment.environment"}; !reflect.DeepEqual(attrKeys(rl.Resource.Attributes), want) {
		t.Fatalf("resource attributes = %v", attrKeys(rl.Resource.Attributes))
	}
	if rl.ScopeLogs[0].Scope.Name != "scope/test" || rl.ScopeLogs[0].Scope.Version != "9.9.9" {
		t.Fatalf("scope = %+v", rl.ScopeLogs[0].Scope)
	}
	records := rl.ScopeLogs[0].LogRecords
	if len(records) != 2 {
		t.Fatalf("both mapped records must be emitted, got %d", len(records))
	}
	first, second := records[0], records[1]
	if first.TimeUnixNano != "1788570123000000000" || first.ObservedTimeUnixNano != "1788652800000000000" {
		t.Fatalf("record time vs observed time: %s / %s", first.TimeUnixNano, first.ObservedTimeUnixNano)
	}
	if first.SeverityText != "ERROR" || second.SeverityText != "WARN" {
		t.Fatalf("severity text derivation: %s / %s", first.SeverityText, second.SeverityText)
	}
	if want := []string{"a", "n"}; !reflect.DeepEqual(attrKeys(first.Attributes), want) {
		t.Fatalf("empty string attributes must be dropped: %v", attrKeys(first.Attributes))
	}
	if first.TraceID != second.TraceID || first.SpanID == second.SpanID {
		t.Fatalf("records of one payload share a trace, not a span: %+v %+v", first, second)
	}
}

func TestOTelSink_TraceContext(t *testing.T) {
	s := newTestOTelSink(&captureSink{})
	at := time.Now()
	supplied, _ := s.Encode(mappedRecord{at: at, key: "s1", tp: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"})
	if rec := firstRecord(t, supplied); rec.TraceID != "4bf92f3577b34da6a3ce929d0e0e4736" || rec.SpanID != "00f067aa0ba902b7" {
		t.Fatalf("traceparent not honoured: %q %q", rec.TraceID, rec.SpanID)
	}
	malformed, _ := s.Encode(mappedRecord{at: at, key: "s1", tp: "garbage"})
	if rec := firstRecord(t, malformed); len(rec.TraceID) != 32 || rec.TraceID == "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("malformed traceparent must fall back to a generated id: %q", rec.TraceID)
	}
	one, _ := s.Encode(mappedRecord{at: at, key: "s1"})
	two, _ := s.Encode(mappedRecord{at: at, key: "s1"})
	other, _ := s.Encode(mappedRecord{at: at, key: "s2"})
	if firstRecord(t, one).TraceID != firstRecord(t, two).TraceID {
		t.Fatal("records sharing a correlation key must share a trace id")
	}
	if firstRecord(t, one).TraceID != firstRecord(t, malformed).TraceID {
		t.Fatal("the fallback must reuse the key's trace id")
	}
	if firstRecord(t, one).TraceID == firstRecord(t, other).TraceID {
		t.Fatal("different correlation keys must not share a trace id")
	}
	anonA, _ := s.Encode(mappedRecord{at: at})
	anonB, _ := s.Encode(mappedRecord{at: at})
	if firstRecord(t, anonA).TraceID == firstRecord(t, anonB).TraceID {
		t.Fatal("uncorrelated records get their own trace id")
	}
}

func TestOTelSink_TrackedKeysBounded(t *testing.T) {
	s := newTestOTelSink(&captureSink{})
	for i := 0; i < otelMaxTrackedKeys+5; i++ {
		s.traceContext("", "k"+string(rune('a'+i%26))+string(rune(i)))
	}
	if len(s.traceIDs) > otelMaxTrackedKeys {
		t.Fatalf("tracked keys not bounded: %d", len(s.traceIDs))
	}
}

func TestOTelSink_WireShapeAndClose(t *testing.T) {
	capture := &captureSink{}
	s := NewOTelSink(capture, OTelResource{ServiceName: "svc"}, OTelScope{Name: "n", Version: "1", SchemaURL: "https://example/schema"})
	if err := s.Record(context.Background(), map[string]any{"k": "v"}); err != nil {
		t.Fatal(err)
	}
	line, err := json.Marshal(capture.payloads[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`{"resourceLogs":[{"resource":{"attributes":[{"key":"service.name","value":{"stringValue":"svc"}}]}`,
		`"scopeLogs":[{"scope":{"name":"n","version":"1"},"logRecords":[{"timeUnixNano":"`,
		`"body":{"stringValue":"{\"k\":\"v\"}"}`,
		`"attributes":[{"key":"k","value":{"stringValue":"v"}}]`,
		`"schemaUrl":"https://example/schema"}]`,
	} {
		if !strings.Contains(string(line), want) {
			t.Fatalf("wire line lacks %s:\n%s", want, line)
		}
	}
	if strings.Contains(string(line), "service.version") {
		t.Fatalf("empty resource attributes must be dropped: %s", line)
	}
	if err := s.Close(); err != nil || !capture.closed {
		t.Fatalf("Close must propagate to the inner sink (err=%v closed=%v)", err, capture.closed)
	}
}

type failingSink struct{}

func (failingSink) Record(context.Context, any) error { return errors.New("disk full") }
func (failingSink) Close() error                      { return nil }

func TestOTelSink_PropagatesInnerErrors(t *testing.T) {
	s := NewOTelSink(failingSink{}, OTelResource{}, OTelScope{})
	if err := s.Record(context.Background(), map[string]any{}); err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("inner sink error must surface, got %v", err)
	}
}
