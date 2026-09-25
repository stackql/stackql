package sink

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

// The OTLP/HTTP exporter is configured only through the standard
// OTEL_EXPORTER_OTLP_* environment variables; a logs-specific variable
// takes precedence over its generic counterpart.
const (
	otlpLogsEndpointEnv  = "OTEL_EXPORTER_OTLP_LOGS_ENDPOINT"
	otlpEndpointEnv      = "OTEL_EXPORTER_OTLP_ENDPOINT"
	otlpLogsHeadersEnv   = "OTEL_EXPORTER_OTLP_LOGS_HEADERS"
	otlpHeadersEnv       = "OTEL_EXPORTER_OTLP_HEADERS"
	otlpLogsTimeoutEnv   = "OTEL_EXPORTER_OTLP_LOGS_TIMEOUT"
	otlpTimeoutEnv       = "OTEL_EXPORTER_OTLP_TIMEOUT"
	otlpBatchSizeEnv     = "OTEL_BLRP_MAX_EXPORT_BATCH_SIZE"
	otlpLogsPath         = "/v1/logs"
	otlpDefaultTimeout   = 10 * time.Second
	otlpDefaultBatchSize = 512
	otlpMaxAttempts      = 4
	otlpInitialBackoff   = 500 * time.Millisecond
	otlpBackoffFactor    = 2
	otlpMaxRetryAfter    = 30 * time.Second
	otlpErrorBodyLimit   = 1024
)

// otlpHTTPConfig is the resolved exporter configuration.
type otlpHTTPConfig struct {
	endpoint  string
	headers   map[string]string
	timeout   time.Duration
	batchSize int
}

// otlpHTTPSink batches OTelLogsData payloads and POSTs them as OTLP/JSON to
// an OTLP/HTTP logs endpoint: one request per batchSize records, with 429
// and 5xx responses retried after a backoff. Close ships the remainder, so
// the batch spans the sink's lifetime (one statement for the CLI writer).
type otlpHTTPSink struct {
	cfg     otlpHTTPConfig
	client  *http.Client
	backoff time.Duration
	sleep   func(time.Duration)

	mu      sync.Mutex
	pending OTelLogsData
	count   int
}

// NewOTLPHTTPSinkFromEnv returns the exporter configured from the
// environment, or false when neither OTEL_EXPORTER_OTLP_LOGS_ENDPOINT nor
// OTEL_EXPORTER_OTLP_ENDPOINT is set.
func NewOTLPHTTPSinkFromEnv() (Sink, bool) {
	cfg, ok := otlpHTTPConfigFromEnv()
	if !ok {
		return nil, false
	}
	return newOTLPHTTPSink(cfg), true
}

func newOTLPHTTPSink(cfg otlpHTTPConfig) *otlpHTTPSink {
	return &otlpHTTPSink{
		cfg:     cfg,
		client:  &http.Client{Timeout: cfg.timeout},
		backoff: otlpInitialBackoff,
		sleep:   time.Sleep,
	}
}

// otlpHTTPConfigFromEnv resolves the endpoint (the logs endpoint verbatim,
// else the generic endpoint with the logs path appended), headers, timeout
// and batch size.
func otlpHTTPConfigFromEnv() (otlpHTTPConfig, bool) {
	cfg := otlpHTTPConfig{
		endpoint:  os.Getenv(otlpLogsEndpointEnv),
		headers:   otlpParseHeaders(otlpEnv(otlpLogsHeadersEnv, otlpHeadersEnv)),
		timeout:   otlpDefaultTimeout,
		batchSize: otlpDefaultBatchSize,
	}
	if cfg.endpoint == "" {
		base := strings.TrimRight(os.Getenv(otlpEndpointEnv), "/")
		if base == "" {
			return cfg, false
		}
		cfg.endpoint = base + otlpLogsPath
	}
	if ms, err := strconv.Atoi(otlpEnv(otlpLogsTimeoutEnv, otlpTimeoutEnv)); err == nil && ms > 0 {
		cfg.timeout = time.Duration(ms) * time.Millisecond
	}
	if n, err := strconv.Atoi(os.Getenv(otlpBatchSizeEnv)); err == nil && n > 0 {
		cfg.batchSize = n
	}
	return cfg, true
}

// otlpEnv returns the logs-specific variable when set, else the generic one.
func otlpEnv(specific, generic string) string {
	if v := os.Getenv(specific); v != "" {
		return v
	}
	return os.Getenv(generic)
}

// otlpParseHeaders parses the key=value,key=value list; values may be
// percent-encoded as the specification allows.
func otlpParseHeaders(raw string) map[string]string {
	headers := map[string]string{}
	for _, pair := range strings.Split(raw, ",") {
		key, value, found := strings.Cut(pair, "=")
		key = strings.TrimSpace(key)
		if !found || key == "" {
			continue
		}
		value = strings.TrimSpace(value)
		if unescaped, err := url.PathUnescape(value); err == nil {
			value = unescaped
		}
		headers[key] = value
	}
	return headers
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
	if s.count >= s.cfg.batchSize {
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
	return fmt.Errorf("otlp http sink: export to %s failed: %w", s.cfg.endpoint, err)
}

// send performs one export attempt, reporting whether a failure is
// retryable and any Retry-After hint.
func (s *otlpHTTPSink) send(ctx context.Context, body []byte) (bool, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.endpoint, bytes.NewReader(body))
	if err != nil {
		return false, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range s.cfg.headers {
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
