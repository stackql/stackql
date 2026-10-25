package sql_system

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/stackql/stackql/internal/stackql/sqlengine"
)

// queryOnlyEngine satisfies sqlengine.SQLEngine for the one method the cache
// freshness lookups call.
type queryOnlyEngine struct {
	sqlengine.SQLEngine
	db *sql.DB
}

func (e queryOnlyEngine) Query(q string, args ...any) (*sql.Rows, error) {
	return e.db.Query(q, args...)
}

// recordingEngine captures the rendered SQL and bound args and returns no rows.
type recordingEngine struct {
	sqlengine.SQLEngine
	query string
	args  []any
}

func (e *recordingEngine) Query(q string, args ...any) (*sql.Rows, error) {
	e.query = q
	e.args = args
	return nil, sql.ErrNoRows
}

// injectedEncoding is the probe from issue #822: interpolated, it matched
// another request's cache entry.
const injectedEncoding = "missing' OR 1=1 --"

func TestSQLiteTableOldestUpdateUTCBindsRequestEncoding(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`CREATE TABLE probe (modified text, encoded text);
INSERT INTO probe VALUES ('2026-10-01T00:00:00', 'other-request')`)
	if err != nil {
		t.Fatal(err)
	}
	sys := &sqLiteSystem{
		sqlEngine: queryOnlyEngine{db: db},
	}
	if _, hit := sys.TableOldestUpdateUTC("probe", "other-request", "modified", "encoded"); !hit {
		t.Fatal("identical encoding must hit the cache")
	}
	if _, miss := sys.TableOldestUpdateUTC("probe", "missing", "modified", "encoded"); miss {
		t.Fatal("unknown encoding must miss the cache")
	}
	if _, injected := sys.TableOldestUpdateUTC("probe", injectedEncoding, "modified", "encoded"); injected {
		t.Fatal("injected encoding must not match another request's cache entry")
	}
}

func TestPostgresTableOldestUpdateUTCBindsRequestEncoding(t *testing.T) {
	eng := &recordingEngine{}
	sys := &postgresSystem{
		sqlEngine:   eng,
		tableSchema: "stackql_raw",
	}
	sys.TableOldestUpdateUTC("probe", injectedEncoding, "modified", "encoded")
	if !strings.Contains(eng.query, "WHERE encoded = $1") || strings.Contains(eng.query, injectedEncoding) {
		t.Fatalf("encoding must be bound, not interpolated: %s", eng.query)
	}
	if len(eng.args) != 1 || eng.args[0] != injectedEncoding {
		t.Fatalf("unexpected bound args: %v", eng.args)
	}
}
