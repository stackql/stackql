package input_data_staging //nolint:revive,stylecheck,testpackage // exercise result preparation

import (
	"io"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stackql/stackql/internal/stackql/typing"
)

func TestNativeEmptyResultPreservesSchema(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRowsWithColumnDefinition(
		sqlmock.NewColumn("alias").OfType("TEXT", ""),
	))
	rows, err := db.Query("SELECT name AS alias WHERE false")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rows.Close() })
	cfg, err := typing.NewTypingConfig("sqlite3")
	if err != nil {
		t.Fatal(err)
	}
	output := NewNaiveNativeResultSetPreparator(rows, nil, cfg, nil).PrepareNativeResultSet()
	if output.GetError() != nil {
		t.Fatal(output.GetError())
	}
	stream := output.GetSQLResult()
	columns := stream.GetColumns()
	if len(columns) != 1 || columns[0].GetName() != "alias" {
		t.Fatalf("unexpected columns: %v", columns)
	}
	result, err := stream.Read()
	if err != io.EOF || result == nil || len(result.GetRows()) != 0 {
		t.Fatalf("expected zero rows and EOF, got %v, %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
