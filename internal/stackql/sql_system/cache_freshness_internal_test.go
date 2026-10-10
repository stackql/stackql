package sql_system

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/stackql/any-sdk/pkg/db/sqlcontrol"
	"github.com/stackql/any-sdk/public/sqlengine"
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
// another request's control counters.
const injectedEncoding = "missing' OR 1=1 --"

func TestSQLiteTableOldestUpdateUTCBindsRequestEncoding(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`CREATE TABLE probe (modified text, encoded text, iql_generation_id int, iql_session_id int, iql_txn_id int, iql_insert_id int);
INSERT INTO probe VALUES ('2026-10-01T00:00:00', 'other-request', 1, 2, 3, 4)`)
	if err != nil {
		t.Fatal(err)
	}
	sys := &sqLiteSystem{
		sqlEngine:         queryOnlyEngine{db: db},
		controlAttributes: sqlcontrol.GetControlAttributes("standard"),
	}
	if _, hit := sys.TableOldestUpdateUTC("probe", "other-request", "modified", "encoded"); hit == nil {
		t.Fatal("identical encoding must hit the cache")
	}
	if _, miss := sys.TableOldestUpdateUTC("probe", "missing", "modified", "encoded"); miss != nil {
		t.Fatal("unknown encoding must miss the cache")
	}
	if _, injected := sys.TableOldestUpdateUTC("probe", injectedEncoding, "modified", "encoded"); injected != nil {
		t.Fatal("injected encoding must not match another request's counters")
	}
}

func TestPostgresTableOldestUpdateUTCBindsRequestEncoding(t *testing.T) {
	eng := &recordingEngine{}
	sys := &postgresSystem{
		sqlEngine:         eng,
		tableSchema:       "stackql_raw",
		controlAttributes: sqlcontrol.GetControlAttributes("standard"),
	}
	sys.TableOldestUpdateUTC("probe", injectedEncoding, "modified", "encoded")
	if !strings.Contains(eng.query, "WHERE encoded = $1") || strings.Contains(eng.query, injectedEncoding) {
		t.Fatalf("encoding must be bound, not interpolated: %s", eng.query)
	}
	if len(eng.args) != 1 || eng.args[0] != injectedEncoding {
		t.Fatalf("unexpected bound args: %v", eng.args)
	}
}
