package output //nolint:testpackage // do not care

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/pkg/sink"
)

const otelTestQuery = "select name, region, disabled, spec from google.compute.firewalls " +
	"where project = 'p1' and zone = 'z1'"

func otelTestStream(rows ...[]interface{}) sqldata.ISQLResultStream {
	table := sqldata.NewSQLTable(0, "t")
	cols := []sqldata.ISQLColumn{}
	for i, name := range []string{"name", "region", "disabled", "spec"} {
		cols = append(cols, sqldata.NewSQLColumn(table, name, int16(i), 0, 0, 0, "text")) //nolint:gosec // test
	}
	sqlRows := make([]sqldata.ISQLRow, 0, len(rows))
	for _, r := range rows {
		sqlRows = append(sqlRows, sqldata.NewSQLRow(r))
	}
	return sqldata.NewSimpleSQLResultStream(sqldata.NewSQLResult(cols, 0, 0, sqlRows))
}

// otelLines decodes one LogsData per line and returns its single record
// with the resource and scope it was wrapped in.
func otelLines(t *testing.T, out string) []sink.OTelResourceLogs {
	t.Helper()
	var rv []sink.OTelResourceLogs
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		var data sink.OTelLogsData
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			t.Fatalf("line is not LogsData: %v: %q", err, line)
		}
		if len(data.ResourceLogs) != 1 || len(data.ResourceLogs[0].ScopeLogs) != 1 ||
			len(data.ResourceLogs[0].ScopeLogs[0].LogRecords) != 1 {
			t.Fatalf("expected one record per line, got %q", line)
		}
		rv = append(rv, data.ResourceLogs[0])
	}
	return rv
}

func otelRecord(rl sink.OTelResourceLogs) sink.OTelWireLogRecord {
	return rl.ScopeLogs[0].LogRecords[0]
}

func otelAttr(attrs []sink.OTelAttribute, key string) (sink.OTelAnyValue, bool) {
	for _, a := range attrs {
		if a.Key == key {
			return a.Value, true
		}
	}
	return sink.OTelAnyValue{}, false
}

func otelString(t *testing.T, attrs []sink.OTelAttribute, key string) string {
	t.Helper()
	v, ok := otelAttr(attrs, key)
	if !ok || v.StringValue == nil {
		t.Fatalf("missing string attribute %q in %+v", key, attrs)
	}
	return *v.StringValue
}

func otelKeys(attrs []sink.OTelAttribute) []string {
	keys := make([]string, 0, len(attrs))
	for _, a := range attrs {
		keys = append(keys, a.Key)
	}
	return keys
}

// otelSchemaFixture writes two rows and returns the decoded lines.
func otelSchemaFixture(t *testing.T) []sink.OTelResourceLogs {
	t.Helper()
	var out bytes.Buffer
	start := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	w := NewOTelWriter(&out, &bytes.Buffer{}, otelTestQuery, start)
	stream := otelTestStream(
		[]interface{}{"fw-a", "us-east1", false, map[string]interface{}{"priority": 1000}},
		[]interface{}{"fw-b", "us-east1", true, nil},
	)
	if err := w.Write(stream); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	lines := otelLines(t, out.String())
	if len(lines) != 3 {
		t.Fatalf("expected 2 rows + completion, got %d lines", len(lines))
	}
	return lines
}

func TestOTelWriter_AttributeSchema(t *testing.T) {
	lines := otelSchemaFixture(t)
	wantRow := []string{
		"disabled", "name", "region", "spec",
		"stackql.snapshot.id", "stackql.query", "stackql.query.hash",
		"stackql.provider", "stackql.service", "stackql.resource",
		"stackql.row.index", "stackql.row.fingerprint",
	}
	if got := otelKeys(otelRecord(lines[0]).Attributes); !reflect.DeepEqual(got, wantRow) {
		t.Fatalf("row attribute schema drift:\n got %v\nwant %v", got, wantRow)
	}
	wantCompletion := []string{
		"stackql.snapshot.id", "stackql.query", "stackql.query.hash",
		"stackql.provider", "stackql.service", "stackql.resource",
		"stackql.snapshot.complete", "stackql.rows_returned", "stackql.duration_ms",
	}
	if got := otelKeys(otelRecord(lines[2]).Attributes); !reflect.DeepEqual(got, wantCompletion) {
		t.Fatalf("completion attribute schema drift:\n got %v\nwant %v", got, wantCompletion)
	}
	for _, rl := range lines {
		if scope := rl.ScopeLogs[0].Scope; scope.Name != otelScopeName || scope.Version != otelAttributeSchemaVersion {
			t.Fatalf("scope: %+v", scope)
		}
	}
}

func TestOTelWriter_RowValuesAndBody(t *testing.T) {
	lines := otelSchemaFixture(t)
	row := otelRecord(lines[0])
	if v, _ := otelAttr(row.Attributes, "disabled"); v.BoolValue == nil || *v.BoolValue {
		t.Fatalf("disabled should be boolValue false: %+v", v)
	}
	if otelString(t, row.Attributes, "spec") != `{"priority":1000}` {
		t.Fatalf("nested column must be a JSON string: %+v", row.Attributes)
	}
	if _, ok := otelAttr(otelRecord(lines[1]).Attributes, "spec"); ok {
		t.Fatalf("null column must be dropped from attributes")
	}
	if row.Body.StringValue == nil || *row.Body.StringValue != `{"disabled":false,"name":"fw-a","region":"us-east1","spec":{"priority":1000}}` {
		t.Fatalf("body must be the full row JSON: %+v", row.Body)
	}
	if otelString(t, row.Attributes, "stackql.provider") != "google" ||
		otelString(t, row.Attributes, "stackql.service") != "compute" ||
		otelString(t, row.Attributes, "stackql.resource") != "firewalls" ||
		otelString(t, row.Attributes, "stackql.query") != otelTestQuery ||
		len(otelString(t, row.Attributes, "stackql.query.hash")) != 64 {
		t.Fatalf("statement context: %+v", row.Attributes)
	}
}

func TestOTelWriter_CompletionRecord(t *testing.T) {
	completion := otelRecord(otelSchemaFixture(t)[2])
	if v, _ := otelAttr(completion.Attributes, "stackql.rows_returned"); v.IntValue == nil || *v.IntValue != "2" {
		t.Fatalf("rows_returned: %+v", v)
	}
	if v, _ := otelAttr(completion.Attributes, "stackql.snapshot.complete"); v.BoolValue == nil || !*v.BoolValue {
		t.Fatalf("snapshot.complete: %+v", v)
	}
	if completion.Body.StringValue == nil || *completion.Body.StringValue != "snapshot complete google.compute.firewalls" {
		t.Fatalf("completion body: %+v", completion.Body)
	}
}

// Every record of a statement shares the snapshot instant, id, trace and span.
func TestOTelWriter_StatementIdentity(t *testing.T) {
	lines := otelSchemaFixture(t)
	first := otelRecord(lines[0])
	snapshotID := otelString(t, first.Attributes, "stackql.snapshot.id")
	for i, rl := range lines {
		rec := otelRecord(rl)
		if rec.TimeUnixNano != "1788739200000000000" || rec.TraceID != otelInvocationTraceID() ||
			rec.SpanID != first.SpanID || otelString(t, rec.Attributes, "stackql.snapshot.id") != snapshotID {
			t.Fatalf("record %d does not share the statement identity: %+v", i, rec)
		}
	}
}

func TestOTelWriter_ResourceAttributes(t *testing.T) {
	resource := otelSchemaFixture(t)[0].Resource.Attributes
	if otelString(t, resource, "service.name") != "stackql" ||
		otelString(t, resource, "cloud.provider") != "gcp" ||
		otelString(t, resource, "cloud.account.id") != "p1" ||
		otelString(t, resource, "cloud.availability_zone") != "z1" {
		t.Fatalf("resource attributes: %+v", resource)
	}
}

func TestOTelWriter_ZeroRowsEmitsOnlyCompletion(t *testing.T) {
	var out bytes.Buffer
	w := NewOTelWriter(&out, &bytes.Buffer{}, otelTestQuery, time.Time{})
	if err := w.Write(otelTestStream()); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	lines := otelLines(t, out.String())
	if len(lines) != 1 {
		t.Fatalf("expected exactly one completion record, got %d", len(lines))
	}
	if v, _ := otelAttr(otelRecord(lines[0]).Attributes, "stackql.rows_returned"); v.IntValue == nil || *v.IntValue != "0" {
		t.Fatalf("rows_returned: %+v", v)
	}
}

func TestOTelWriter_FingerprintTracksRowContent(t *testing.T) {
	fingerprint := func(disabled bool) string {
		var out bytes.Buffer
		w := NewOTelWriter(&out, &bytes.Buffer{}, otelTestQuery, time.Time{})
		if err := w.Write(otelTestStream([]interface{}{"fw-a", "us-east1", disabled, nil})); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
		return otelString(t, otelRecord(otelLines(t, out.String())[0]).Attributes, "stackql.row.fingerprint")
	}
	if a, b := fingerprint(false), fingerprint(false); a != b || !strings.HasPrefix(a, "sha256:") {
		t.Fatalf("identical rows must fingerprint identically: %s vs %s", a, b)
	}
	if fingerprint(false) == fingerprint(true) {
		t.Fatalf("a changed column value must change the fingerprint")
	}
}

func TestOTelWriter_WriteError(t *testing.T) {
	var out, errOut bytes.Buffer
	w := NewOTelWriter(&out, &errOut, "select 1", time.Time{})
	if err := w.WriteError(errors.New("boom"), "record"); err != nil {
		t.Fatalf("WriteError() error = %v", err)
	}
	rec := otelRecord(otelLines(t, out.String())[0])
	if rec.SeverityNumber != sink.OTelSeverityError || rec.SeverityText != "ERROR" ||
		otelString(t, rec.Attributes, "error.message") != "boom" ||
		otelString(t, rec.Attributes, "error.type") != "*errors.errorString" {
		t.Fatalf("error record: %+v", rec)
	}
	if err := w.WriteError(errors.New("boom"), stderrPressentationStr); err != nil || strings.TrimSpace(errOut.String()) != "boom" {
		t.Fatalf("stderr presentation: err=%v out=%q", err, errOut.String())
	}
}

func TestGetOutputWriter_OTel(t *testing.T) {
	ctx := internaldto.OutputContext{RuntimeContext: dto.RuntimeCtx{OutputFormat: "otel"}, Query: "select 1"}
	w, err := GetOutputWriter(&bytes.Buffer{}, &bytes.Buffer{}, ctx)
	if err != nil {
		t.Fatalf("GetOutputWriter() error = %v", err)
	}
	if _, ok := w.(*OTelWriter); !ok {
		t.Fatalf("expected *OTelWriter, got %T", w)
	}
}
