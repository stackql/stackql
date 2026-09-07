package output

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stackql/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/buildinfo"
	"github.com/stackql/stackql/pkg/sink"
)

// The emitted attribute set is a versioned interface (issue #738): bump
// otelAttributeSchemaVersion when it changes and update the schema test.
const (
	otelFormatStr              = "otel"
	otelAttributeSchemaVersion = "1.0.0"
	otelScopeName              = "github.com/stackql/stackql/internal/stackql/output"
	otelServiceName            = "stackql"
	otelTraceParentEnv         = "TRACEPARENT"
	otelServiceNameEnv         = "OTEL_SERVICE_NAME"
	otelResourceAttrsEnv       = "OTEL_RESOURCE_ATTRIBUTES"
	otelTraceIDBytes           = 16
	otelSpanIDBytes            = 8
	otelTraceParentParts       = 4
)

// otelInvocationTraceID is shared by every statement of one process,
// honouring a TRACEPARENT supplied by the environment.
//
//nolint:gochecknoglobals // per-process trace identity
var otelInvocationTraceID = sync.OnceValue(func() string {
	parts := strings.Split(os.Getenv(otelTraceParentEnv), "-")
	if len(parts) == otelTraceParentParts && len(parts[1]) == 2*otelTraceIDBytes {
		return parts[1]
	}
	return sink.OTelRandomID(otelTraceIDBytes)
})

// OTelWriter emits one OTLP/JSON LogsData per result row plus a completion
// record per statement, so a result set can be shipped as a timestamped
// inventory snapshot. Every record of a statement shares the snapshot
// instant, snapshot id and span id.
type OTelWriter struct {
	errWriter  io.Writer
	encoder    sink.Sink
	startTime  time.Time
	snapshotID string
	spanID     string
	query      string
	queryHash  string
	statement  otelStatementContext
}

// otelStatementContext is what a lightweight parse of the statement yields:
// the primary table and any cloud scoping parameters.
type otelStatementContext struct {
	provider string
	service  string
	resource string
	cloud    []sink.OTelAttribute
}

// otelRecordPayload adapts a prepared record to the sink decorator.
type otelRecordPayload struct {
	record sink.OTelLogRecord
}

func (p otelRecordPayload) OTelLogRecords() []sink.OTelLogRecord {
	return []sink.OTelLogRecord{p.record}
}

// writerSink is the io.Writer-backed sink the decorator writes through: one
// compact JSON document per line, flushed per record like the jsonl writer.
type writerSink struct {
	writer io.Writer
}

func (s writerSink) Record(_ context.Context, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err = s.writer.Write(append(b, '\n')); err != nil {
		return err
	}
	if flusher, ok := s.writer.(writeFlusher); ok {
		return flusher.Flush()
	}
	return nil
}

func (writerSink) Close() error { return nil }

// NewOTelWriter returns the writer for one statement: query is the submitted
// text and startTime the snapshot instant stamped on every record.
func NewOTelWriter(writer io.Writer, errWriter io.Writer, query string, startTime time.Time) IOutputWriter {
	if startTime.IsZero() {
		startTime = time.Now()
	}
	statement := newOTelStatementContext(query)
	queryHash := sha256.Sum256([]byte(strings.TrimSpace(query)))
	return &OTelWriter{
		errWriter:  errWriter,
		encoder:    sink.NewOTelSink(writerSink{writer: writer}, otelResource(statement), otelScope()),
		startTime:  startTime,
		snapshotID: uuid.NewString(),
		spanID:     sink.OTelRandomID(otelSpanIDBytes),
		query:      query,
		queryHash:  hex.EncodeToString(queryHash[:]),
		statement:  statement,
	}
}

func otelScope() sink.OTelScope {
	return sink.OTelScope{Name: otelScopeName, Version: otelAttributeSchemaVersion}
}

// otelResource is service.*, the derivable cloud.* conventions, then the
// standard OTEL_RESOURCE_ATTRIBUTES pairs.
func otelResource(statement otelStatementContext) sink.OTelResource {
	serviceName := os.Getenv(otelServiceNameEnv)
	if serviceName == "" {
		serviceName = otelServiceName
	}
	attrs := append([]sink.OTelAttribute{}, statement.cloud...)
	for _, pair := range strings.Split(os.Getenv(otelResourceAttrsEnv), ",") {
		key, value, found := strings.Cut(pair, "=")
		if !found || strings.TrimSpace(key) == "" {
			continue
		}
		if unescaped, err := url.PathUnescape(value); err == nil {
			value = unescaped
		}
		attrs = append(attrs, sink.OTelString(strings.TrimSpace(key), strings.TrimSpace(value)))
	}
	return sink.OTelResource{
		ServiceName:    serviceName,
		ServiceVersion: buildinfo.Get().GetSemVersion(),
		Attributes:     attrs,
	}
}

// newOTelStatementContext parses a single-table SELECT for its
// provider.service.resource and the equality parameters that map onto the
// cloud resource conventions; anything more complex yields nothing.
func newOTelStatementContext(query string) otelStatementContext {
	var rv otelStatementContext
	stmt, err := sqlparser.Parse(query)
	if err != nil {
		return rv
	}
	sel, ok := stmt.(*sqlparser.Select)
	if !ok || len(sel.From) != 1 {
		return rv
	}
	aliased, ok := sel.From[0].(*sqlparser.AliasedTableExpr)
	if !ok {
		return rv
	}
	tableName, ok := aliased.Expr.(sqlparser.TableName)
	if !ok || tableName.QualifierSecond.IsEmpty() {
		return rv
	}
	rv.provider = tableName.QualifierSecond.GetRawVal()
	rv.service = tableName.Qualifier.GetRawVal()
	rv.resource = tableName.Name.GetRawVal()
	params := map[string]string{}
	if sel.Where != nil {
		otelCollectEqualities(sel.Where.Expr, params)
	}
	rv.cloud = otelCloudAttributes(rv.provider, params)
	return rv
}

// otelCollectEqualities gathers `column = 'literal'` terms of an AND chain.
func otelCollectEqualities(expr sqlparser.Expr, out map[string]string) {
	switch node := expr.(type) {
	case *sqlparser.AndExpr:
		otelCollectEqualities(node.Left, out)
		otelCollectEqualities(node.Right, out)
	case *sqlparser.ComparisonExpr:
		col, isCol := node.Left.(*sqlparser.ColName)
		val, isVal := node.Right.(*sqlparser.SQLVal)
		if node.Operator == sqlparser.EqualStr && isCol && isVal && val.Type == sqlparser.StrVal {
			out[strings.ToLower(col.Name.GetRawVal())] = string(val.Val)
		}
	}
}

func otelCloudAttributes(providerName string, params map[string]string) []sink.OTelAttribute {
	var attrs []sink.OTelAttribute
	switch {
	case strings.HasPrefix(providerName, "google"):
		attrs = append(attrs, sink.OTelString("cloud.provider", "gcp"))
	case strings.HasPrefix(providerName, "aws"):
		attrs = append(attrs, sink.OTelString("cloud.provider", "aws"))
	case strings.HasPrefix(providerName, "azure"):
		attrs = append(attrs, sink.OTelString("cloud.provider", "azure"))
	case providerName == "oci":
		attrs = append(attrs, sink.OTelString("cloud.provider", "oracle_cloud"))
	}
	for param, key := range map[string]string{
		"project":        "cloud.account.id",
		"subscriptionid": "cloud.account.id",
		"region":         "cloud.region",
		"zone":           "cloud.availability_zone",
	} {
		if v := params[param]; v != "" {
			attrs = append(attrs, sink.OTelString(key, v))
		}
	}
	return attrs
}

// contextAttributes are the stackql.* attributes common to every record of
// the statement.
func (ow *OTelWriter) contextAttributes() []sink.OTelAttribute {
	return []sink.OTelAttribute{
		sink.OTelString("stackql.snapshot.id", ow.snapshotID),
		sink.OTelString("stackql.query", ow.query),
		sink.OTelString("stackql.query.hash", ow.queryHash),
		sink.OTelString("stackql.provider", ow.statement.provider),
		sink.OTelString("stackql.service", ow.statement.service),
		sink.OTelString("stackql.resource", ow.statement.resource),
	}
}

func (ow *OTelWriter) record(severity int, body string, attrs []sink.OTelAttribute) error {
	return ow.encoder.Record(context.Background(), otelRecordPayload{sink.OTelLogRecord{
		Time:        ow.startTime,
		Severity:    severity,
		Body:        body,
		Attributes:  attrs,
		TraceParent: fmt.Sprintf("00-%s-%s-01", otelInvocationTraceID(), ow.spanID),
	}})
}

// writeRow emits one row: the full row JSON as the body, scalar columns as
// typed attributes, nested columns as JSON strings, plus the row's index and
// a fingerprint of the canonical row JSON so drift is one comparison.
func (ow *OTelWriter) writeRow(row map[string]interface{}, index int) error {
	body, err := json.Marshal(row)
	if err != nil {
		return err
	}
	attrs, _ := sink.OTelAttributesFromJSON(body)
	fingerprint := sha256.Sum256(body)
	attrs = append(attrs, ow.contextAttributes()...)
	attrs = append(attrs,
		sink.OTelInt("stackql.row.index", int64(index)),
		sink.OTelString("stackql.row.fingerprint", "sha256:"+hex.EncodeToString(fingerprint[:])),
	)
	return ow.record(sink.OTelSeverityInfo, string(body), attrs)
}

// writeCompletion marks the snapshot whole; a zero-row statement still emits
// it so an absent resource is visible as drift rather than as silence.
func (ow *OTelWriter) writeCompletion(rows int) error {
	body := "snapshot complete"
	if ow.statement.provider != "" {
		body += " " + strings.Join([]string{ow.statement.provider, ow.statement.service, ow.statement.resource}, ".")
	}
	attrs := append(ow.contextAttributes(),
		sink.OTelBool("stackql.snapshot.complete", true),
		sink.OTelInt("stackql.rows_returned", int64(rows)),
		sink.OTelInt("stackql.duration_ms", time.Since(ow.startTime).Milliseconds()),
	)
	return ow.record(sink.OTelSeverityInfo, body, attrs)
}

func (ow *OTelWriter) Write(res sqldata.ISQLResultStream) error {
	rows := 0
	for {
		r, err := res.Read()
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		if r != nil {
			for _, row := range r.ToArr() {
				if rowErr := ow.writeRow(row, rows); rowErr != nil {
					return rowErr
				}
				rows++
			}
		}
		if err != nil {
			return ow.writeCompletion(rows)
		}
	}
}

func (ow *OTelWriter) WriteError(err error, errorPresentation string) error {
	if errorPresentation == stderrPressentationStr {
		return writeStderrError(ow.errWriter, err)
	}
	attrs := append(ow.contextAttributes(),
		sink.OTelString("error.type", fmt.Sprintf("%T", err)),
		sink.OTelString("error.message", err.Error()),
	)
	return ow.record(sink.OTelSeverityError, err.Error(), attrs)
}
