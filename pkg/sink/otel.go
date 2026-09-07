package sink

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// OTel log severity numbers per the OpenTelemetry logs data model.
const (
	OTelSeverityInfo  = 9
	OTelSeverityError = 17

	// DefaultOTelSchemaURL pins the semantic conventions release the stable
	// attributes (service.*, error.*) are taken from.
	DefaultOTelSchemaURL = "https://opentelemetry.io/schemas/1.44.0"

	otelTraceIDBytes = 16
	otelSpanIDBytes  = 8
	hexCharsPerByte  = 2
	// otelMaxTrackedKeys bounds the correlation key -> trace id map.
	otelMaxTrackedKeys = 1024
)

// OTelRecorder is implemented by payloads that map themselves onto log
// records. Payloads that do not implement it are decorated generically: the
// JSON object's top-level fields become attributes and the whole object the
// body.
type OTelRecorder interface {
	OTelLogRecords() []OTelLogRecord
}

// OTelLogRecord is one log record before envelope decoration; the sink adds
// the resource, scope, observed time, and trace/span ids.
type OTelLogRecord struct {
	Time     time.Time
	Severity int
	Body     string
	// Attributes are emitted in order; empty string values are dropped.
	Attributes []OTelAttribute
	// TraceParent is a caller-supplied W3C traceparent, honoured when well
	// formed; otherwise one trace id is generated per CorrelationKey (e.g. a
	// session) so related records correlate.
	TraceParent    string
	CorrelationKey string
}

// OTelResource identifies the emitting service.
type OTelResource struct {
	ServiceName    string
	ServiceVersion string
	// Attributes are appended after service.name / service.version.
	Attributes []OTelAttribute
}

// OTelScope identifies the instrumentation that produced the records; use
// Version for the emitted attribute schema so consumers can pin against it.
type OTelScope struct {
	Name      string
	Version   string
	SchemaURL string
}

// OTelAttribute is one OTLP/JSON key/value pair.
type OTelAttribute struct {
	Key   string       `json:"key"`
	Value OTelAnyValue `json:"value"`
}

// OTelAnyValue is the OTLP/JSON encoding of an attribute value; exactly one
// field is set.
type OTelAnyValue struct {
	StringValue *string  `json:"stringValue,omitempty"`
	IntValue    *string  `json:"intValue,omitempty"`
	DoubleValue *float64 `json:"doubleValue,omitempty"`
	BoolValue   *bool    `json:"boolValue,omitempty"`
}

// OTelString / OTelInt / OTelDouble / OTelBool build typed attributes.
func OTelString(key, v string) OTelAttribute {
	return OTelAttribute{Key: key, Value: OTelAnyValue{StringValue: &v}}
}

func OTelInt(key string, v int64) OTelAttribute {
	s := fmt.Sprintf("%d", v)
	return OTelAttribute{Key: key, Value: OTelAnyValue{IntValue: &s}}
}

func OTelDouble(key string, v float64) OTelAttribute {
	return OTelAttribute{Key: key, Value: OTelAnyValue{DoubleValue: &v}}
}

func OTelBool(key string, v bool) OTelAttribute {
	return OTelAttribute{Key: key, Value: OTelAnyValue{BoolValue: &v}}
}

// OTelWireLogRecord is an OTLP/JSON LogRecord.
type OTelWireLogRecord struct {
	TimeUnixNano         string          `json:"timeUnixNano"`
	ObservedTimeUnixNano string          `json:"observedTimeUnixNano"`
	SeverityNumber       int             `json:"severityNumber"`
	SeverityText         string          `json:"severityText"`
	Body                 OTelAnyValue    `json:"body"`
	Attributes           []OTelAttribute `json:"attributes"`
	TraceID              string          `json:"traceId"`
	SpanID               string          `json:"spanId"`
}

type otelInstrumentationScope struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// OTelScopeLogs is an OTLP/JSON ScopeLogs.
type OTelScopeLogs struct {
	Scope      otelInstrumentationScope `json:"scope"`
	LogRecords []OTelWireLogRecord      `json:"logRecords"`
	SchemaURL  string                   `json:"schemaUrl"`
}

type otelResource struct {
	Attributes []OTelAttribute `json:"attributes"`
}

// OTelResourceLogs is an OTLP/JSON ResourceLogs.
type OTelResourceLogs struct {
	Resource  otelResource    `json:"resource"`
	ScopeLogs []OTelScopeLogs `json:"scopeLogs"`
	SchemaURL string          `json:"schemaUrl"`
}

// OTelLogsData is one OTLP/JSON logs export payload; the file sink writes
// one per line, the shape the collector's otlp_json_file receiver ingests.
type OTelLogsData struct {
	ResourceLogs []OTelResourceLogs `json:"resourceLogs"`
}

// otelSink decorates every payload as OTLP/JSON log records and forwards
// the result to the wrapped sink.
type otelSink struct {
	inner    Sink
	resource OTelResource
	scope    OTelScope
	now      func() time.Time

	mu       sync.Mutex
	traceIDs map[string]string
}

// NewOTelSink wraps inner so every recorded payload is written as OTLP/JSON.
func NewOTelSink(inner Sink, resource OTelResource, scope OTelScope) Sink {
	if scope.SchemaURL == "" {
		scope.SchemaURL = DefaultOTelSchemaURL
	}
	return &otelSink{inner: inner, resource: resource, scope: scope, now: time.Now, traceIDs: map[string]string{}}
}

func (s *otelSink) Record(ctx context.Context, payload any) error {
	data, err := s.Encode(payload)
	if err != nil {
		return err
	}
	return s.inner.Record(ctx, data)
}

func (s *otelSink) Close() error { return s.inner.Close() }

// Encode renders one payload as a LogsData.
func (s *otelSink) Encode(payload any) (OTelLogsData, error) {
	var records []OTelLogRecord
	if r, ok := payload.(OTelRecorder); ok {
		records = r.OTelLogRecords()
	} else {
		generic, err := genericOTelLogRecord(payload, s.now())
		if err != nil {
			return OTelLogsData{}, err
		}
		records = []OTelLogRecord{generic}
	}
	observed := unixNano(s.now())
	wire := make([]OTelWireLogRecord, 0, len(records))
	for _, r := range records {
		traceID, spanID := s.traceContext(r.TraceParent, r.CorrelationKey)
		wire = append(wire, OTelWireLogRecord{
			TimeUnixNano:         unixNano(r.Time),
			ObservedTimeUnixNano: observed,
			SeverityNumber:       r.Severity,
			SeverityText:         severityText(r.Severity),
			Body:                 OTelAnyValue{StringValue: &r.Body},
			Attributes:           compactOTelAttributes(r.Attributes),
			TraceID:              traceID,
			SpanID:               spanID,
		})
	}
	resourceAttrs := append([]OTelAttribute{
		OTelString("service.name", s.resource.ServiceName),
		OTelString("service.version", s.resource.ServiceVersion),
	}, s.resource.Attributes...)
	return OTelLogsData{ResourceLogs: []OTelResourceLogs{{
		Resource: otelResource{Attributes: compactOTelAttributes(resourceAttrs)},
		ScopeLogs: []OTelScopeLogs{{
			Scope:      otelInstrumentationScope{Name: s.scope.Name, Version: s.scope.Version},
			LogRecords: wire,
			SchemaURL:  s.scope.SchemaURL,
		}},
		SchemaURL: s.scope.SchemaURL,
	}}}, nil
}

// genericOTelLogRecord decorates an arbitrary JSON-marshalable payload: the
// compact JSON is the body and each top-level scalar field an attribute
// (nested values are carried as JSON strings), keyed in sorted order.
func genericOTelLogRecord(payload any, now time.Time) (OTelLogRecord, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return OTelLogRecord{}, fmt.Errorf("otel sink: marshal payload: %w", err)
	}
	record := OTelLogRecord{Time: now, Severity: OTelSeverityInfo, Body: string(raw)}
	// Not a JSON object (scalar or array): the body alone carries it.
	record.Attributes, _ = OTelAttributesFromJSON(raw)
	return record, nil
}

// OTelAttributesFromJSON maps a JSON object's top-level fields to typed
// attributes in key order; nested values are carried as JSON strings and
// nulls as empty strings (dropped on encode). ok is false when raw is not
// an object.
func OTelAttributesFromJSON(raw []byte) ([]OTelAttribute, bool) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, false
	}
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	attrs := make([]OTelAttribute, 0, len(keys))
	for _, k := range keys {
		attrs = append(attrs, otelAttributeFromJSON(k, fields[k]))
	}
	return attrs, true
}

func otelAttributeFromJSON(key string, raw json.RawMessage) OTelAttribute {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return OTelString(key, string(raw))
	}
	switch t := v.(type) {
	case string:
		return OTelString(key, t)
	case bool:
		return OTelBool(key, t)
	case json.Number:
		if i, intErr := t.Int64(); intErr == nil {
			return OTelInt(key, i)
		}
		f, floatErr := t.Float64()
		if floatErr != nil {
			return OTelString(key, t.String())
		}
		if f == float64(int64(f)) {
			return OTelInt(key, int64(f))
		}
		return OTelDouble(key, f)
	case nil:
		return OTelString(key, "")
	default:
		return OTelString(key, string(raw))
	}
}

// traceContext honours a well-formed traceparent, otherwise reuses one
// generated trace id per correlation key.
func (s *otelSink) traceContext(traceParent, key string) (string, string) {
	if parts := strings.Split(traceParent, "-"); len(parts) == 4 &&
		len(parts[1]) == hexCharsPerByte*otelTraceIDBytes && len(parts[2]) == hexCharsPerByte*otelSpanIDBytes {
		return parts[1], parts[2]
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	traceID, known := s.traceIDs[key]
	if !known || key == "" {
		if len(s.traceIDs) >= otelMaxTrackedKeys {
			s.traceIDs = map[string]string{}
		}
		traceID = OTelRandomID(otelTraceIDBytes)
		s.traceIDs[key] = traceID
	}
	return traceID, OTelRandomID(otelSpanIDBytes)
}

// OTelRandomID returns n random bytes as lowercase hex (16 bytes for a trace
// id, 8 for a span or call id).
func OTelRandomID(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return strings.Repeat("0", hexCharsPerByte*n)
	}
	return hex.EncodeToString(b)
}

func unixNano(t time.Time) string { return fmt.Sprintf("%d", t.UnixNano()) }

func severityText(severity int) string {
	switch {
	case severity >= OTelSeverityError:
		return "ERROR"
	case severity >= 13: //nolint:mnd // WARN band start per the OTel logs data model
		return "WARN"
	case severity >= OTelSeverityInfo:
		return "INFO"
	default:
		return "DEBUG"
	}
}

// compactOTelAttributes drops attributes with an empty string value.
func compactOTelAttributes(attrs []OTelAttribute) []OTelAttribute {
	out := make([]OTelAttribute, 0, len(attrs))
	for _, a := range attrs {
		if a.Value.StringValue != nil && *a.Value.StringValue == "" {
			continue
		}
		out = append(out, a)
	}
	return out
}
