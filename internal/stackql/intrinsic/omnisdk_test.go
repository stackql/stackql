package intrinsic //nolint:testpackage // tests unexported rowStream and pickMethod

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/lib/pq/oid"
	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/output"
)

type fakeRows struct {
	rows   []omnisdk.Row
	idx    int
	err    error
	closed bool
}

func (f *fakeRows) Next() bool {
	if f.idx < len(f.rows) {
		f.idx++
		return true
	}
	return false
}

func (f *fakeRows) Row() omnisdk.Row { return f.rows[f.idx-1] }
func (f *fakeRows) Err() error       { return f.err }
func (f *fakeRows) Close() error     { f.closed = true; return nil }

type fakeColumnFactory struct{}

func (fakeColumnFactory) GetPlaceholderColumn(
	table sqldata.ISQLTable, colName string, colOID oid.Oid) sqldata.ISQLColumn {
	return sqldata.NewSQLColumn(table, colName, 0, uint32(colOID), 1024, 0, "TextFormat")
}

func newTestStream(rows []omnisdk.Row, columnNames []string) (*rowStream, *fakeRows) {
	cursor := &fakeRows{rows: rows}
	cols := make([]column, 0, len(columnNames))
	for _, name := range columnNames {
		cols = append(cols, column{name: name})
	}
	return &rowStream{
		rows:    cursor,
		columns: cols,
		table:   sqldata.NewSQLTable(0, "t"),
		typCfg:  fakeColumnFactory{},
	}, cursor
}

func drain(t *testing.T, stream *rowStream) [][]interface{} {
	t.Helper()
	var out [][]interface{}
	for {
		res, err := stream.Read()
		if res == nil {
			t.Fatalf("nil result alongside err %v", err)
		}
		for _, row := range res.GetRows() {
			out = append(out, row.GetRowDataNaive())
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return out
			}
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestRowStreamEmitsRowsInColumnOrder(t *testing.T) {
	rows := []omnisdk.Row{
		{"b": "b1", "a": "a1"},
		{"b": "b2", "a": "a2"},
	}
	stream, cursor := newTestStream(rows, []string{"a", "b"})
	got := drain(t, stream)
	want := [][]interface{}{{"a1", "b1"}, {"a2", "b2"}}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if !cursor.closed {
		t.Fatal("underlying cursor was not closed")
	}
}

func TestRowStreamBatchesLargeCursor(t *testing.T) {
	total := streamBatchSize*2 + 3
	rows := make([]omnisdk.Row, 0, total)
	for i := 0; i < total; i++ {
		rows = append(rows, omnisdk.Row{"a": fmt.Sprintf("v%d", i)})
	}
	stream, _ := newTestStream(rows, []string{"a"})
	var reads int
	var seen int
	for {
		res, err := stream.Read()
		reads++
		seen += len(res.GetRows())
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if seen != total {
		t.Fatalf("read %d rows, want %d", seen, total)
	}
	if reads < 3 {
		t.Fatalf("cursor of %d rows was drained in %d reads; batching is not happening", total, reads)
	}
}

func TestRowStreamDerivesColumnsFromFirstRow(t *testing.T) {
	stream, _ := newTestStream([]omnisdk.Row{{"zeta": "z", "alpha": "a"}}, nil)
	got := drain(t, stream)
	want := [][]interface{}{{"a", "z"}}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestRowStreamPropagatesCursorError(t *testing.T) {
	cursor := &fakeRows{err: errors.New("upstream failed")}
	stream := &rowStream{
		rows:   cursor,
		table:  sqldata.NewSQLTable(0, "t"),
		typCfg: fakeColumnFactory{},
	}
	_, err := stream.Read()
	if err == nil || errors.Is(err, io.EOF) {
		t.Fatalf("want upstream error, got %v", err)
	}
}

func TestPickMethodDisambiguates(t *testing.T) {
	if _, err := pickMethod("aws.s3.buckets", map[string]string{"region": "us-east-1"}); err == nil {
		t.Fatal("want ambiguity error, got nil")
	}
	method, err := pickMethod("aws.s3.buckets", map[string]string{
		"region": "us-east-1", methodPredicate: "enumerate"})
	if err != nil {
		t.Fatalf("explicit method: %v", err)
	}
	if method.Path != "aws.s3.buckets.enumerate" {
		t.Fatalf("got %s", method.Path)
	}
}

func TestPickMethodReportsMissingParams(t *testing.T) {
	if _, err := pickMethod("aws.s3.buckets", map[string]string{}); err == nil {
		t.Fatal("want missing-parameter error, got nil")
	}
}

func TestBooleanValuesRenderLikeOtherProviders(t *testing.T) {
	stream, _ := newTestStream(
		[]omnisdk.Row{{"name": "b1", "public": false, "versioning": true}}, nil)
	stream.columns = []column{
		{name: "name", dataType: "string"},
		{name: "public", dataType: "boolean"},
		{name: "versioning", dataType: "boolean"},
	}
	var buf, errBuf bytes.Buffer
	writer, err := output.GetOutputWriter(&buf, &errBuf, internaldto.OutputContext{
		RuntimeContext: dto.RuntimeCtx{OutputFormat: "csv", Delimiter: ","},
		Result:         stream,
	})
	if err != nil {
		t.Fatalf("writer: %v", err)
	}
	if writeErr := writer.Write(stream); writeErr != nil {
		t.Fatalf("write: %v", writeErr)
	}
	want := "name,public,versioning\nb1,false,true\n"
	if buf.String() != want {
		t.Fatalf("got %q, want %q", buf.String(), want)
	}
}

func TestReportedTypeUsesStackqlVocabulary(t *testing.T) {
	for _, tc := range []struct{ declared, want string }{
		{"boolean", "bool"},
		{"string", "string"},
		{"integer", "integer"},
		{"", "text"},
	} {
		if got := (column{dataType: tc.declared}).reportedType(); got != tc.want {
			t.Errorf("declared %q: got %q, want %q", tc.declared, got, tc.want)
		}
	}
}
