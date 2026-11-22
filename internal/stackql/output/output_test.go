package output //nolint:testpackage // do not care

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stackql/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	dto "github.com/stackql/stackql/internal/stackql/runtimeconfig"
)

type flushCountingWriter struct {
	b          bytes.Buffer
	flushCount int
}

func (w *flushCountingWriter) Write(p []byte) (int, error) {
	return w.b.Write(p)
}

func (w *flushCountingWriter) Flush() error {
	w.flushCount++
	return nil
}

func (w *flushCountingWriter) String() string {
	return w.b.String()
}

type errResultStream struct {
	err error
}

func (s *errResultStream) Read() (sqldata.ISQLResult, error) {
	return nil, s.err
}

func (s *errResultStream) GetColumns() []sqldata.ISQLColumn {
	return nil
}

func (s *errResultStream) Write(sqldata.ISQLResult) error {
	return nil
}

func (s *errResultStream) Close() error {
	return nil
}

func TestJSONLWriter_writeRows(t *testing.T) {
	var b bytes.Buffer
	w := &JSONLWriter{writer: &b, errWriter: &b}
	rows := []map[string]interface{}{
		{"k": "v1", "n": 1},
		{"k": "v2", "n": 2},
	}

	if err := w.writeRows(rows); err != nil {
		t.Fatalf("writeRows() error = %v", err)
	}

	out := strings.TrimSpace(b.String())
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 JSONL lines, got %d: %q", len(lines), out)
	}
	if !strings.Contains(lines[0], `"k":"v1"`) || !strings.Contains(lines[0], `"n":1`) {
		t.Fatalf("first line mismatch: %q", lines[0])
	}
	if !strings.Contains(lines[1], `"k":"v2"`) || !strings.Contains(lines[1], `"n":2`) {
		t.Fatalf("second line mismatch: %q", lines[1])
	}
}

func TestJSONLWriter_WriteError_Record(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	w := &JSONLWriter{writer: &out, errWriter: &errOut}

	if err := w.WriteError(errors.New("boom"), "record"); err != nil {
		t.Fatalf("WriteError() error = %v", err)
	}

	line := strings.TrimSpace(out.String())
	if !strings.Contains(line, `"error":"boom"`) {
		t.Fatalf("expected error record in JSONL output, got %q", line)
	}
	if errOut.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", errOut.String())
	}
}

func TestGetOutputWriter_JSONL(t *testing.T) {
	ctx := internaldto.OutputContext{RuntimeContext: dto.RuntimeCtx{OutputFormat: "jsonl"}}
	w, err := GetOutputWriter(&bytes.Buffer{}, &bytes.Buffer{}, ctx)
	if err != nil {
		t.Fatalf("GetOutputWriter() error = %v", err)
	}
	if _, ok := w.(*JSONLWriter); !ok {
		t.Fatalf("expected *JSONLWriter, got %T", w)
	}
}

func TestJSONLWriter_writeRows_EagerFlushPerRow(t *testing.T) {
	target := &flushCountingWriter{}
	w := &JSONLWriter{writer: target, errWriter: io.Discard}
	rows := []map[string]interface{}{
		{"k": "v1", "n": 1},
		{"k": "v2", "n": 2},
	}

	if err := w.writeRows(rows); err != nil {
		t.Fatalf("writeRows() error = %v", err)
	}

	if target.flushCount != len(rows) {
		t.Fatalf("expected flush count %d, got %d", len(rows), target.flushCount)
	}

	out := strings.TrimSpace(target.String())
	if got := len(strings.Split(out, "\n")); got != len(rows) {
		t.Fatalf("expected %d lines, got %d: %q", len(rows), got, out)
	}
}

func TestJSONLWriter_Write_PropagatesStreamError(t *testing.T) {
	w := &JSONLWriter{writer: &bytes.Buffer{}, errWriter: io.Discard}
	wantErr := errors.New("stream broke")

	err := w.Write(&errResultStream{err: wantErr})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func newBatchResult(t *testing.T, start, count int) sqldata.ISQLResult {
	t.Helper()
	columns := []sqldata.ISQLColumn{
		sqldata.NewSQLColumn(sqldata.NewSQLTable(0, ""), "id", 0, 25, 1024, 0, "text"),
	}
	rows := make([]sqldata.ISQLRow, 0, count)
	for i := start; i < start+count; i++ {
		rows = append(rows, sqldata.NewSQLRow([]interface{}{fmt.Sprintf("row-%d", i)}))
	}
	return sqldata.NewSQLResult(columns, 0, 0, rows)
}

func TestJSONWriter_Write_MultipleBatchesIsOneArray(t *testing.T) {
	for _, batchSizes := range [][]int{{100}, {100, 1}, {100, 100, 50}} {
		stream := sqldata.NewChannelSQLResultStream(nil)
		total := 0
		go func(sizes []int) {
			start := 0
			for _, size := range sizes {
				stream.Write(newBatchResult(t, start, size)) //nolint:errcheck // channel write
				start += size
			}
			stream.Close() //nolint:errcheck // channel close
		}(batchSizes)
		for _, size := range batchSizes {
			total += size
		}

		var out bytes.Buffer
		w := NewJSONWriter(&out, &out)
		if err := w.Write(stream); err != nil {
			t.Fatalf("batches %v: Write() error = %v", batchSizes, err)
		}

		var rows []map[string]interface{}
		if err := json.Unmarshal(out.Bytes(), &rows); err != nil {
			t.Fatalf("batches %v: output is not one JSON document: %v", batchSizes, err)
		}
		if len(rows) != total {
			t.Fatalf("batches %v: expected %d rows, got %d", batchSizes, total, len(rows))
		}
		if rows[0]["id"] != "row-0" || rows[total-1]["id"] != fmt.Sprintf("row-%d", total-1) {
			t.Fatalf("batches %v: rows out of order: first %v, last %v", batchSizes, rows[0], rows[total-1])
		}
	}
}

func TestJSONWriter_Write_RetainsCompletedRowsOnStreamFailure(t *testing.T) {
	failure := errors.New("continuation failed")
	batch := newBatchResult(t, 0, 2)
	stream := &partialTableStream{batch: batch, failure: failure}

	var out bytes.Buffer
	w := NewJSONWriter(&out, io.Discard)
	if err := w.Write(stream); !errors.Is(err, failure) {
		t.Fatalf("expected %v, got %v", failure, err)
	}

	var rows []map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &rows); err != nil {
		t.Fatalf("partial output is not one JSON document: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 partial rows, got %d: %v", len(rows), rows)
	}
	if rows[0]["id"] != "row-0" || rows[1]["id"] != "row-1" {
		t.Fatalf("partial rows were not retained in order: %v", rows)
	}
}
