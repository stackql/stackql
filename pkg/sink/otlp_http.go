package sink

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	otlpDefaultTimeout   = 10 * time.Second
	otlpDefaultBatchSize = 512
	otlpMaxAttempts      = 4
	otlpInitialBackoff   = 500 * time.Millisecond
	otlpBackoffFactor    = 2
	otlpMaxRetryAfter    = 30 * time.Second
	otlpErrorBodyLimit   = 1024
)

// OTLPHTTPConfig configures the OTLP/HTTP logs exporter. Endpoint is the
// full logs URL (for example http://collector:4318/v1/logs) and is required;
// the rest are optional.
type OTLPHTTPConfig struct {
	Endpoint string `json:"endpoint" yaml:"endpoint"`

	// Headers are sent on every export request, for example an ingestion
	// token or API key.
	Headers map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`

	// TimeoutMS bounds each request; zero means 10000.
	TimeoutMS int `json:"timeout_ms,omitempty" yaml:"timeout_ms,omitempty"`

	// BatchSize is the number of records per request; zero means 512.
	BatchSize int `json:"batch_size,omitempty" yaml:"batch_size,omitempty"`
}

// otlpHTTPSink batches OTelLogsData payloads and POSTs them as OTLP/JSON to
// an OTLP/HTTP logs endpoint: one request per batchSize records, with 429
// and 5xx responses retried after a backoff. Close ships the remainder, so
// the batch spans the sink's lifetime (one statement for the CLI writer).
type otlpHTTPSink struct {
	endpoint  string
	headers   map[string]string
	batchSize int
	client    *http.Client
	backoff   time.Duration
	sleep     func(time.Duration)

	mu      sync.Mutex
	pending OTelLogsData
	count   int
}

// NewOTLPHTTPSink returns the exporter for cfg; the endpoint is required.
func NewOTLPHTTPSink(cfg OTLPHTTPConfig) (Sink, error) {
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return nil, errors.New("otlp http sink: endpoint is required")
	}
	return newOTLPHTTPSink(cfg), nil
}

func newOTLPHTTPSink(cfg OTLPHTTPConfig) *otlpHTTPSink {
	timeout := otlpDefaultTimeout
	if cfg.TimeoutMS > 0 {
		timeout = time.Duration(cfg.TimeoutMS) * time.Millisecond
	}
	batchSize := otlpDefaultBatchSize
	if cfg.BatchSize > 0 {
		batchSize = cfg.BatchSize
	}
	return &otlpHTTPSink{
		endpoint:  strings.TrimSpace(cfg.Endpoint),
		headers:   cfg.Headers,
		batchSize: batchSize,
		client:    &http.Client{Timeout: timeout},
		backoff:   otlpInitialBackoff,
		sleep:     time.Sleep,
	}
}

// Record adds the payload's records to the pending batch and ships the
// batch once it holds batchSize records.
func (s *otlpHTTPSink) Record(ctx context.Context, payload any) error {
	data, ok := payload.(OTelLogsData)
	if !ok {
		return fmt.Errorf("otlp http sink: payload is %T, want OTelLogsData", payload)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rl := range data.ResourceLogs {
		s.add(rl)
	}
	if s.count >= s.batchSize {
		return s.flush(ctx)
	}
	return nil
}

// Close ships whatever is still pending.
func (s *otlpHTTPSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.flush(context.Background())
}

// add merges rl into the batch, sharing the resource and scope envelope
// with the preceding records when it is identical.
func (s *otlpHTTPSink) add(rl OTelResourceLogs) {
	for _, sl := range rl.ScopeLogs {
		s.count += len(sl.LogRecords)
	}
	n := len(s.pending.ResourceLogs)
	if n > 0 && otlpSameEnvelope(s.pending.ResourceLogs[n-1], rl) {
		last := &s.pending.ResourceLogs[n-1].ScopeLogs[0]
		last.LogRecords = append(last.LogRecords, rl.ScopeLogs[0].LogRecords...)
		return
	}
	s.pending.ResourceLogs = append(s.pending.ResourceLogs, rl)
}

// otlpSameEnvelope reports whether two single-scope ResourceLogs share
// resource attributes, scope and schema URLs.
func otlpSameEnvelope(a, b OTelResourceLogs) bool {
	if len(a.ScopeLogs) != 1 || len(b.ScopeLogs) != 1 || a.SchemaURL != b.SchemaURL {
		return false
	}
	return a.ScopeLogs[0].Scope == b.ScopeLogs[0].Scope &&
		a.ScopeLogs[0].SchemaURL == b.ScopeLogs[0].SchemaURL &&
		reflect.DeepEqual(a.Resource, b.Resource)
}

// flush ships the pending batch; the caller holds mu.
func (s *otlpHTTPSink) flush(ctx context.Context) error {
	if s.count == 0 {
		return nil
	}
	body, err := json.Marshal(s.pending)
	s.pending, s.count = OTelLogsData{}, 0
	if err != nil {
		return fmt.Errorf("otlp http sink: marshal batch: %w", err)
	}
	return s.post(ctx, body)
}

// post sends one batch, retrying retryable failures after the server's
// Retry-After or an exponential backoff, up to otlpMaxAttempts.
func (s *otlpHTTPSink) post(ctx context.Context, body []byte) error {
	delay := s.backoff
	var err error
	for attempt := 1; attempt <= otlpMaxAttempts; attempt++ {
		var retryable bool
		var retryAfter time.Duration
		retryable, retryAfter, err = s.send(ctx, body)
		if err == nil {
			return nil
		}
		if !retryable || attempt == otlpMaxAttempts {
			break
		}
		wait := delay
		if retryAfter > 0 {
			wait = retryAfter
		}
		s.sleep(wait)
		delay *= otlpBackoffFactor
	}
	return fmt.Errorf("otlp http sink: export to %s failed: %w", s.endpoint, err)
}

// send performs one export attempt, reporting whether a failure is
// retryable and any Retry-After hint.
func (s *otlpHTTPSink) send(ctx context.Context, body []byte) (bool, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(body))
	if err != nil {
		return false, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range s.headers {
		req.Header.Set(k, v)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return true, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, resp.Body)
		return false, 0, nil
	}
	detail, _ := io.ReadAll(io.LimitReader(resp.Body, otlpErrorBodyLimit))
	retryable := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError
	return retryable, otlpRetryAfter(resp.Header.Get("Retry-After")),
		fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(detail)))
}

// otlpRetryAfter parses a delay-seconds Retry-After, capped; zero when
// absent or in the HTTP-date form.
func otlpRetryAfter(raw string) time.Duration {
	secs, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || secs <= 0 {
		return 0
	}
	return min(time.Duration(secs)*time.Second, otlpMaxRetryAfter)
}
