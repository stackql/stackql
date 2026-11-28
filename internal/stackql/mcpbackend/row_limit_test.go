package mcpbackend //nolint:testpackage // exercise unexported stream drain

import (
	"fmt"
	"io"
	"testing"

	"github.com/stackql/psql-wire/pkg/sqldata"
)

// batchedResultStream replays fixed batches; when eofWithLast is set the final
// batch is delivered together with io.EOF, as SimpleSQLResultStream does.
type batchedResultStream struct {
	batches     []sqldata.ISQLResult
	eofWithLast bool
	next        int
}

func (s *batchedResultStream) GetColumns() []sqldata.ISQLColumn { return nil }

func (s *batchedResultStream) Read() (sqldata.ISQLResult, error) {
	if s.next >= len(s.batches) {
		return nil, io.EOF
	}
	batch := s.batches[s.next]
	s.next++
	if s.eofWithLast && s.next == len(s.batches) {
		return batch, io.EOF
	}
	return batch, nil
}

func (s *batchedResultStream) Write(sqldata.ISQLResult) error { return fmt.Errorf("not implemented") }

func (s *batchedResultStream) Close() error { return nil }

func rowLimitTestBatch(start, size int) sqldata.ISQLResult {
	cols := []sqldata.ISQLColumn{
		sqldata.NewSQLColumn(sqldata.NewSQLTable(0, ""), "id", 0, 0, 0, 0, "text"),
	}
	rows := make([]sqldata.ISQLRow, 0, size)
	for i := start; i < start+size; i++ {
		rows = append(rows, sqldata.NewSQLRow([]interface{}{fmt.Sprintf("%d", i)}))
	}
	return sqldata.NewSQLResult(cols, 0, 0, rows)
}

func TestDrainSQLRowStream_RowLimit(t *testing.T) {
	cases := []struct {
		name        string
		batchSizes  []int
		eofWithLast bool
		rowLimit    int
		want        int
	}{
		{name: "single batch with EOF over limit", batchSizes: []int{5}, eofWithLast: true, rowLimit: 2, want: 2},
		{name: "single batch before EOF over limit", batchSizes: []int{5}, rowLimit: 2, want: 2},
		{name: "limit reached mid second batch", batchSizes: []int{3, 3}, rowLimit: 4, want: 4},
		{name: "final EOF batch crosses limit", batchSizes: []int{3, 3}, eofWithLast: true, rowLimit: 4, want: 4},
		{name: "limit above row count", batchSizes: []int{3, 3}, eofWithLast: true, rowLimit: 10, want: 6},
		{name: "zero limit is unlimited", batchSizes: []int{3, 3}, eofWithLast: true, rowLimit: 0, want: 6},
		{name: "negative limit is unlimited", batchSizes: []int{5}, eofWithLast: true, rowLimit: unlimitedRowLimit, want: 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stream := &batchedResultStream{eofWithLast: tc.eofWithLast}
			start := 0
			for _, size := range tc.batchSizes {
				stream.batches = append(stream.batches, rowLimitTestBatch(start, size))
				start += size
			}
			rows, rowCount, err := drainSQLRowStream(stream, []map[string]interface{}{}, 0, tc.rowLimit)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(rows) != tc.want {
				t.Fatalf("got %d rows, want %d", len(rows), tc.want)
			}
			if rowCount != tc.want {
				t.Fatalf("got rowCount %d, want %d", rowCount, tc.want)
			}
			for i, row := range rows {
				if row["id"] != fmt.Sprintf("%d", i) {
					t.Fatalf("row %d: got id %v, want %d", i, row["id"], i)
				}
			}
		})
	}
}

// A limit already met by an earlier statement's stream must not be exceeded
// by rows from a later one.
func TestDrainSQLRowStream_RowLimitAlreadyReached(t *testing.T) {
	stream := &batchedResultStream{batches: []sqldata.ISQLResult{rowLimitTestBatch(0, 3)}, eofWithLast: true}
	prior := []map[string]interface{}{{"id": "a"}, {"id": "b"}}
	rows, rowCount, err := drainSQLRowStream(stream, prior, len(prior), 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 || rowCount != 2 {
		t.Fatalf("got %d rows (rowCount %d), want 2", len(rows), rowCount)
	}
}
