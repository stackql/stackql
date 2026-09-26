package sink //nolint:testpackage // exercise the exporter directly

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"
)

// otlpCapture is an OTLP/HTTP endpoint that records every export request
// and answers with the queued statuses (200 once they run out).
type otlpCapture struct {
	mu         sync.Mutex
	statuses   []int
	retryAfter string
	paths      []string
	headers    []http.Header
	bodies     []OTelLogsData
}

func (c *otlpCapture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var data OTelLogsData
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paths = append(c.paths, r.URL.Path)
	c.headers = append(c.headers, r.Header.Clone())
	c.bodies = append(c.bodies, data)
	status := http.StatusOK
	if len(c.statuses) > 0 {
		status, c.statuses = c.statuses[0], c.statuses[1:]
	}
	if status >= http.StatusBadRequest && c.retryAfter != "" {
		w.Header().Set("Retry-After", c.retryAfter)
	}
	w.WriteHeader(status)
}

func (c *otlpCapture) records(i int) []OTelWireLogRecord {
	body := c.bodies[i]
	if len(body.ResourceLogs) != 1 || len(body.ResourceLogs[0].ScopeLogs) != 1 {
		return nil
	}
	return body.ResourceLogs[0].ScopeLogs[0].LogRecords
}

// otlpTestSink wires an exporter with an instant, recorded sleep to the
// capture server; n single-record LogsData payloads are returned to feed it.
func otlpTestSink(t *testing.T, capture *otlpCapture, batchSize int, n int) (*otlpHTTPSink, []OTelLogsData, *[]time.Duration) {
	t.Helper()
	srv := httptest.NewServer(capture)
	t.Cleanup(srv.Close)
	s := newOTLPHTTPSink(OTLPHTTPConfig{
		Endpoint:  srv.URL + "/v1/logs",
		Headers:   map[string]string{"api-key": "secret"},
		TimeoutMS: 1000,
		BatchSize: batchSize,
	})
	waits := &[]time.Duration{}
	s.sleep = func(d time.Duration) { *waits = append(*waits, d) }
	encoder := &captureSink{}
	otel := NewOTelSink(encoder, OTelResource{ServiceName: "svc"}, OTelScope{Name: "scope", Version: "1"})
	payloads := make([]OTelLogsData, 0, n)
	for i := 0; i < n; i++ {
		if err := otel.Record(context.Background(), map[string]any{"i": i}); err != nil {
			t.Fatalf("encode: %v", err)
		}
		payloads = append(payloads, encoder.payloads[i].(OTelLogsData))
	}
	return s, payloads, waits
}

func otlpRecordAll(t *testing.T, s *otlpHTTPSink, payloads []OTelLogsData) error {
	t.Helper()
	for _, p := range payloads {
		if err := s.Record(context.Background(), p); err != nil {
			return err
		}
	}
	return s.Close()
}

func TestOTLPHTTPSink_BatchesBySizeAndSharesEnvelope(t *testing.T) {
	capture := &otlpCapture{}
	s, payloads, _ := otlpTestSink(t, capture, 2, 5)
	if err := otlpRecordAll(t, s, payloads); err != nil {
		t.Fatalf("export: %v", err)
	}
	if got := len(capture.bodies); got != 3 {
		t.Fatalf("expected 3 requests for 5 records at batch size 2, got %d", got)
	}
	for i, want := range []int{2, 2, 1} {
		if got := len(capture.records(i)); got != want {
			t.Fatalf("request %d: %d records, want %d", i, got, want)
		}
		if capture.paths[i] != "/v1/logs" || capture.headers[i].Get("api-key") != "secret" ||
			capture.headers[i].Get("Content-Type") != "application/json" {
			t.Fatalf("request %d: path %q headers %v", i, capture.paths[i], capture.headers[i])
		}
	}
	// The single resource envelope in a batch is the one every payload carried.
	if !reflect.DeepEqual(capture.bodies[0].ResourceLogs[0].Resource, payloads[0].ResourceLogs[0].Resource) {
		t.Fatalf("resource envelope was not preserved: %+v", capture.bodies[0].ResourceLogs[0].Resource)
	}
	if got := capture.records(2)[0].Body; !reflect.DeepEqual(got, payloads[4].ResourceLogs[0].ScopeLogs[0].LogRecords[0].Body) {
		t.Fatalf("records were reordered: last body %+v", got)
	}
}

func TestOTLPHTTPSink_RetriesRetryableStatusesWithBackoff(t *testing.T) {
	capture := &otlpCapture{statuses: []int{http.StatusServiceUnavailable, http.StatusTooManyRequests}}
	s, payloads, waits := otlpTestSink(t, capture, 10, 1)
	if err := otlpRecordAll(t, s, payloads); err != nil {
		t.Fatalf("export: %v", err)
	}
	if got := len(capture.bodies); got != 3 {
		t.Fatalf("expected 2 retries then success, got %d requests", got)
	}
	if want := []time.Duration{otlpInitialBackoff, otlpBackoffFactor * otlpInitialBackoff}; !reflect.DeepEqual(*waits, want) {
		t.Fatalf("backoff waits %v, want %v", *waits, want)
	}
}

func TestOTLPHTTPSink_HonoursRetryAfter(t *testing.T) {
	capture := &otlpCapture{statuses: []int{http.StatusServiceUnavailable}, retryAfter: "3"}
	s, payloads, waits := otlpTestSink(t, capture, 10, 1)
	if err := otlpRecordAll(t, s, payloads); err != nil {
		t.Fatalf("export: %v", err)
	}
	if want := []time.Duration{3 * time.Second}; !reflect.DeepEqual(*waits, want) {
		t.Fatalf("waits %v, want %v", *waits, want)
	}
}

func TestOTLPHTTPSink_DoesNotRetryClientErrors(t *testing.T) {
	capture := &otlpCapture{statuses: []int{http.StatusBadRequest}}
	s, payloads, waits := otlpTestSink(t, capture, 10, 1)
	err := otlpRecordAll(t, s, payloads)
	if err == nil || len(capture.bodies) != 1 || len(*waits) != 0 {
		t.Fatalf("expected one failed attempt, got err=%v requests=%d waits=%v", err, len(capture.bodies), *waits)
	}
}

func TestOTLPHTTPSink_GivesUpAfterMaxAttempts(t *testing.T) {
	statuses := make([]int, otlpMaxAttempts+1)
	for i := range statuses {
		statuses[i] = http.StatusBadGateway
	}
	capture := &otlpCapture{statuses: statuses}
	s, payloads, _ := otlpTestSink(t, capture, 10, 1)
	err := otlpRecordAll(t, s, payloads)
	if err == nil || len(capture.bodies) != otlpMaxAttempts {
		t.Fatalf("expected %d attempts then an error, got err=%v requests=%d", otlpMaxAttempts, err, len(capture.bodies))
	}
}

func TestOTLPHTTPSink_RejectsForeignPayloads(t *testing.T) {
	s, _, _ := otlpTestSink(t, &otlpCapture{}, 10, 0)
	if err := s.Record(context.Background(), map[string]any{"not": "logsdata"}); err == nil {
		t.Fatal("expected an error for a non-LogsData payload")
	}
}

func TestNewOTLPHTTPSink_ConfigDefaults(t *testing.T) {
	if _, err := NewOTLPHTTPSink(OTLPHTTPConfig{Endpoint: "  "}); err == nil {
		t.Fatal("an empty endpoint must be rejected")
	}
	s := newOTLPHTTPSink(OTLPHTTPConfig{Endpoint: " http://collector:4318/v1/logs "})
	if s.endpoint != "http://collector:4318/v1/logs" || s.client.Timeout != otlpDefaultTimeout ||
		s.batchSize != otlpDefaultBatchSize {
		t.Fatalf("defaults not applied: endpoint=%q timeout=%v batch=%d", s.endpoint, s.client.Timeout, s.batchSize)
	}
	s = newOTLPHTTPSink(OTLPHTTPConfig{Endpoint: "http://c/v1/logs", TimeoutMS: 2500, BatchSize: 7,
		Headers: map[string]string{"api-key": "a b"}})
	if s.client.Timeout != 2500*time.Millisecond || s.batchSize != 7 ||
		!reflect.DeepEqual(s.headers, map[string]string{"api-key": "a b"}) {
		t.Fatalf("explicit config not honoured: timeout=%v batch=%d headers=%v", s.client.Timeout, s.batchSize, s.headers)
	}
	if _, err := NewOTLPHTTPSink(OTLPHTTPConfig{Endpoint: "http://c/v1/logs"}); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
}

func TestMultiSink_FansOutAndJoinsCloseErrors(t *testing.T) {
	a, b := &captureSink{}, &captureSink{}
	closeErr := errors.New("boom")
	m := NewMultiSink(a, b, failingCloseSink{err: closeErr})
	if err := m.Record(context.Background(), "x"); err != nil {
		t.Fatalf("record: %v", err)
	}
	if len(a.payloads) != 1 || len(b.payloads) != 1 {
		t.Fatalf("payload was not fanned out: %d %d", len(a.payloads), len(b.payloads))
	}
	if err := m.Close(); !errors.Is(err, closeErr) || !a.closed || !b.closed {
		t.Fatalf("close: err=%v a=%v b=%v", err, a.closed, b.closed)
	}
}

type failingCloseSink struct{ err error }

func (failingCloseSink) Record(_ context.Context, _ any) error { return nil }
func (f failingCloseSink) Close() error                        { return f.err }

func TestOTLPRetryAfter(t *testing.T) {
	cases := map[string]time.Duration{
		"":                              0,
		"abc":                           0,
		"Wed, 21 Oct 2015 07:28:00 GMT": 0,
		"0":                             0,
		"2":                             2 * time.Second,
		strconv.Itoa(3600):              otlpMaxRetryAfter,
	}
	for raw, want := range cases {
		if got := otlpRetryAfter(raw); got != want {
			t.Errorf("otlpRetryAfter(%q) = %v, want %v", raw, got, want)
		}
	}
}
