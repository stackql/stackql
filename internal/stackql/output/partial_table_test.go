//nolint:testpackage // The test stream exercises the package's output writer behavior.
package output

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stackql/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	dto "github.com/stackql/stackql/internal/stackql/runtimeconfig"
)

type partialTableStream struct {
	sqldata.ISQLResultStream
	batch   sqldata.ISQLResult
	failure error
	read    bool
}

func (s *partialTableStream) Read() (sqldata.ISQLResult, error) {
	if !s.read {
		s.read = true
		return s.batch, nil
	}
	return nil, s.failure
}
func (s *partialTableStream) GetColumns() []sqldata.ISQLColumn { return s.batch.GetColumns() }
func (s *partialTableStream) Close() error                     { return nil }

func TestTableWriterRetainsCompletedRowsOnStreamFailure(t *testing.T) {
	failure := errors.New("continuation failed")
	table := sqldata.NewSQLTable(0, "partial")
	column := sqldata.NewSQLColumn(table, "id", 0, 25, 1024, 0, "TextFormat")
	batch := sqldata.NewSQLResult([]sqldata.ISQLColumn{column}, 1, 0, []sqldata.ISQLRow{sqldata.NewSQLRow([]interface{}{"one"})})
	stream := &partialTableStream{batch: batch, failure: failure}
	var output, stderr bytes.Buffer
	writer, err := GetOutputWriter(&output, &stderr, internaldto.OutputContext{RuntimeContext: dto.RuntimeCtx{OutputFormat: "table"}, Result: stream})
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Write(stream); !errors.Is(err, failure) {
		t.Fatalf("stream failure = %v", err)
	}
	if !strings.Contains(output.String(), "one") || !strings.Contains(output.String(), "id") {
		t.Fatalf("completed table was discarded: %q", output.String())
	}
}

func TestCSVWriterRetainsCompletedRowsOnStreamFailure(t *testing.T) {
	failure := errors.New("continuation failed")
	table := sqldata.NewSQLTable(0, "partial")
	column := sqldata.NewSQLColumn(table, "id", 0, 25, 1024, 0, "TextFormat")
	batch := sqldata.NewSQLResult([]sqldata.ISQLColumn{column}, 1, 0, []sqldata.ISQLRow{sqldata.NewSQLRow([]interface{}{"one"})})
	stream := &partialTableStream{batch: batch, failure: failure}
	var output, stderr bytes.Buffer
	writer, err := GetOutputWriter(&output, &stderr, internaldto.OutputContext{RuntimeContext: dto.RuntimeCtx{OutputFormat: "csv", Delimiter: ","}, Result: stream})
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Write(stream); !errors.Is(err, failure) {
		t.Fatalf("stream failure = %v", err)
	}
	if !strings.Contains(output.String(), "one") || !strings.Contains(output.String(), "id") {
		t.Fatalf("completed csv rows were discarded: %q", output.String())
	}
}

type batchedStream struct {
	sqldata.ISQLResultStream
	batches []sqldata.ISQLResult
	next    int
}

func (s *batchedStream) Read() (sqldata.ISQLResult, error) {
	if s.next >= len(s.batches) {
		return nil, io.EOF
	}
	b := s.batches[s.next]
	s.next++
	return b, nil
}
func (s *batchedStream) GetColumns() []sqldata.ISQLColumn { return s.batches[0].GetColumns() }
func (s *batchedStream) Close() error                     { return nil }

func TestCSVWriterRetainsEarlierChunksOnUndecodableChunk(t *testing.T) {
	table := sqldata.NewSQLTable(0, "partial")
	column := sqldata.NewSQLColumn(table, "id", 0, 25, 1024, 0, "TextFormat")
	cols := []sqldata.ISQLColumn{column}
	good := sqldata.NewSQLResult(cols, 1, 0, []sqldata.ISQLRow{sqldata.NewSQLRow([]interface{}{"one"})})
	// two values for a single column: tabulateResults fails on this chunk
	bad := sqldata.NewSQLResult(cols, 1, 0, []sqldata.ISQLRow{sqldata.NewSQLRow([]interface{}{"x", "y"})})
	stream := &batchedStream{batches: []sqldata.ISQLResult{good, bad}}
	var output, stderr bytes.Buffer
	writer, err := GetOutputWriter(&output, &stderr, internaldto.OutputContext{RuntimeContext: dto.RuntimeCtx{OutputFormat: "csv", Delimiter: ","}, Result: stream})
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Write(stream); err == nil {
		t.Fatal("expected decode error for malformed chunk")
	}
	if !strings.Contains(output.String(), "one") {
		t.Fatalf("rows from earlier chunks were discarded: %q", output.String())
	}
}
