package output //nolint:testpackage // do not care

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
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
