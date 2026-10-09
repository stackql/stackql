package omnistaging_test

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"github.com/stackql/any-sdk/pkg/constants"
	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/any-sdk/public/sqlengine"

	. "github.com/stackql/stackql/internal/stackql/omnistaging" //nolint:revive // test reads as package
)

type fixedQueryID int64

func (q fixedQueryID) Value() int64 { return int64(q) }

func (q fixedQueryID) String() string { return strconv.FormatInt(int64(q), 10) }

func sqliteDialect(t *testing.T) Dialect {
	t.Helper()
	d, err := NewDialect(constants.SQLDialectSQLite3, NewConfig(""))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func mustBegin(t *testing.T, begin func(context.Context) (QueryID, error)) QueryID {
	t.Helper()
	id, err := begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestConfig(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{raw: `{"dsn":"file:./stackql.db"}`, want: DefaultQuerySchema},
		{raw: ``, want: DefaultQuerySchema},
		{raw: `{"dbEngine":"postgres_tcp","schemata":{"tableSchema":"t","querySchema":"q"}}`, want: "q"},
	}
	for _, tc := range cases {
		cfg, err := NewConfigFromSQLBackend(tc.raw)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", tc.raw, err)
		}
		if got := cfg.QuerySchema(); got != tc.want {
			t.Fatalf("query schema for %q: got %q want %q", tc.raw, got, tc.want)
		}
	}
}

func TestDialectStatements(t *testing.T) {
	columns := []Column{NewColumn("vpc_id", "text"), NewColumn(`we"ird`, "integer")}
	id := fixedQueryID(42)
	sqlite, err := NewDialect(constants.SQLDialectSQLite3, NewConfig(""))
	if err != nil {
		t.Fatal(err)
	}
	postgres, err := NewDialect(constants.SQLDialectPostgres, NewConfig("q"))
	if err != nil {
		t.Fatal(err)
	}
	mustString := func(s string, err error) string {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	got := []string{
		sqlite.TableName(id),
		mustString(sqlite.CreateTableStatement(id, columns)),
		mustString(sqlite.InsertStatement(id, columns, 2)),
		sqlite.DropTableStatement(id),
		sqlite.NextQueryIDStatement(),
		postgres.TableName(id),
		mustString(postgres.CreateTableStatement(id, columns)),
		mustString(postgres.InsertStatement(id, columns, 2)),
		postgres.DropTableStatement(id),
		postgres.NextQueryIDStatement(),
	}
	want := []string{
		`"__iql__.queries.42"`,
		`CREATE TABLE IF NOT EXISTS "__iql__.queries.42" ("vpc_id" text, "we""ird" integer)`,
		`INSERT INTO "__iql__.queries.42" ("vpc_id", "we""ird") VALUES (?, ?), (?, ?)`,
		`DROP TABLE IF EXISTS "__iql__.queries.42"`,
		`INSERT INTO "__iql__.query_id_seq" DEFAULT VALUES RETURNING id`,
		`"q"."42"`,
		`CREATE TABLE IF NOT EXISTS "q"."42" ("vpc_id" text, "we""ird" integer)`,
		`INSERT INTO "q"."42" ("vpc_id", "we""ird") VALUES ($1, $2), ($3, $4)`,
		`DROP TABLE IF EXISTS "q"."42"`,
		`SELECT nextval('"q"."query_id_seq"')`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("statements:\n got %q\nwant %q", got, want)
	}
	if !reflect.DeepEqual(postgres.SetupStatements(), []string{
		`CREATE SCHEMA IF NOT EXISTS "q"`,
		`CREATE SEQUENCE IF NOT EXISTS "q"."query_id_seq"`,
	}) {
		t.Fatalf("postgres setup: %q", postgres.SetupStatements())
	}
	if _, err = sqlite.CreateTableStatement(id, nil); err == nil {
		t.Fatal("expected error creating table without columns")
	}
	if _, err = NewDialect(constants.SQLDialectSnowflake, NewConfig("")); err == nil {
		t.Fatal("expected unsupported dialect error")
	}
}

func openSQLite(t *testing.T, path string) *sql.DB {
	t.Helper()
	eng, err := sqlengine.NewSQLEngine(dto.SQLBackendCfg{
		DBEngine:  constants.DBEngineSQLite3Embedded,
		SQLSystem: constants.SQLDialectSQLite3,
		DSN:       "file:" + path,
	}, nil)
	if err != nil {
		t.Fatalf("cannot open sqlite engine: %v", err)
	}
	db, err := eng.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func newSQLiteManager(t *testing.T, db *sql.DB, dialect Dialect) Manager {
	t.Helper()
	m, err := NewManager(context.Background(), db, dialect)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func stagingTables(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.Query(
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name LIKE '\_\_iql\_\_.queries.%' ESCAPE '\' ORDER BY name`,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	return names
}

func stagedPairs(t *testing.T, db *sql.DB, table string) [][2]sql.NullString {
	t.Helper()
	rows, err := db.Query("SELECT vpc_id, subnet_id FROM " + table + " ORDER BY rowid")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got [][2]sql.NullString
	for rows.Next() {
		var r [2]sql.NullString
		if err = rows.Scan(&r[0], &r[1]); err != nil {
			t.Fatal(err)
		}
		got = append(got, r)
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	return got
}

// sliceSource yields fixed batches and records how many rows had been
// written to the staging table each time a batch was requested.
type sliceSource struct {
	batches  [][][]any
	next     int
	observe  func() int64
	observed []int64
}

func (s *sliceSource) Next(context.Context) ([][]any, error) {
	if s.observe != nil {
		s.observed = append(s.observed, s.observe())
	}
	if s.next == len(s.batches) {
		return nil, io.EOF
	}
	b := s.batches[s.next]
	s.next++
	return b, nil
}

func countRows(t *testing.T, db *sql.DB, table string) int64 {
	t.Helper()
	var n int64
	if err := db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestQueryIDsCollisionFreeAcrossManagers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stackql.db")
	ctx := context.Background()
	first := newSQLiteManager(t, openSQLite(t, path), sqliteDialect(t))
	second := newSQLiteManager(t, openSQLite(t, path), sqliteDialect(t))
	var got []int64
	for _, m := range []Manager{first, second, first, second} {
		id, err := m.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, id.Value())
	}
	if want := []int64{1, 2, 3, 4}; !reflect.DeepEqual(got, want) {
		t.Fatalf("query ids: got %v want %v", got, want)
	}
}

func TestStageAndRelease(t *testing.T) {
	ctx := context.Background()
	db := openSQLite(t, filepath.Join(t.TempDir(), "stackql.db"))
	m := newSQLiteManager(t, db, sqliteDialect(t))

	streamOnly := mustBegin(t, m.Begin)
	parent := mustBegin(t, m.Begin)
	child := mustBegin(t, func(ctx context.Context) (QueryID, error) { return m.BeginChild(ctx, parent) })
	if p, ok := m.Parent(child); !ok || p.Value() != parent.Value() {
		t.Fatalf("parent of %s: got %v", child, p)
	}

	columns := []Column{NewColumn("vpc_id", "text"), NewColumn("subnet_id", "text")}
	source := &sliceSource{
		batches: [][][]any{
			{{"vpc-1", "subnet-a"}, {"vpc-1", "subnet-a"}},
			{{"vpc-2", nil}},
			{},
			{{"vpc-3", "subnet-c"}},
		},
		observe: func() int64 { return countRows(t, db, m.TableName(parent)) },
	}
	written, err := m.Stage(ctx, parent, columns, source)
	if err != nil {
		t.Fatal(err)
	}
	if written != 4 {
		t.Fatalf("written: got %d want 4", written)
	}
	// Each batch is requested only after the previous one is inserted.
	if want := []int64{0, 2, 3, 3, 4}; !reflect.DeepEqual(source.observed, want) {
		t.Fatalf("rows staged at each pull: got %v want %v", source.observed, want)
	}
	if _, err = m.Stage(ctx, child, columns, &sliceSource{batches: [][][]any{{{"vpc-9", "subnet-z"}}}}); err != nil {
		t.Fatal(err)
	}

	got := stagedPairs(t, db, m.TableName(parent))
	ns := func(s string) sql.NullString { return sql.NullString{String: s, Valid: true} }
	want := [][2]sql.NullString{
		{ns("vpc-1"), ns("subnet-a")},
		{ns("vpc-1"), ns("subnet-a")},
		{ns("vpc-2"), {}},
		{ns("vpc-3"), ns("subnet-c")},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("staged rows: got %v want %v", got, want)
	}

	if tables := stagingTables(t, db); !reflect.DeepEqual(tables, []string{
		"__iql__.queries." + parent.String(),
		"__iql__.queries." + child.String(),
	}) {
		t.Fatalf("tables before release: %v", tables)
	}
	if owned := m.Owned(parent); len(owned) != 2 || owned[0].Value() != child.Value() || owned[1].Value() != parent.Value() {
		t.Fatalf("owned by parent: %v", owned)
	}
	if owned := m.Owned(streamOnly); len(owned) != 0 {
		t.Fatalf("stream-only query owns tables: %v", owned)
	}

	if err = m.Release(ctx, parent); err != nil {
		t.Fatal(err)
	}
	if err = m.Release(ctx, parent); err != nil {
		t.Fatalf("second release must be a no-op: %v", err)
	}
	if err = m.Release(ctx, streamOnly); err != nil {
		t.Fatal(err)
	}
	if tables := stagingTables(t, db); len(tables) != 0 {
		t.Fatalf("tables after release: %v", tables)
	}
	if _, ok := m.Parent(child); ok {
		t.Fatal("child must be released with parent")
	}
	if _, err = m.Stage(ctx, parent, columns, &sliceSource{}); err == nil {
		t.Fatal("expected error staging a released query")
	}
}

type smallBindDialect struct {
	Dialect
}

func (d *smallBindDialect) MaxBindParameters() int {
	return 4
}

func TestInsertChunksByBindParameterLimit(t *testing.T) {
	ctx := context.Background()
	db := openSQLite(t, filepath.Join(t.TempDir(), "stackql.db"))
	m := newSQLiteManager(t, db, &smallBindDialect{Dialect: sqliteDialect(t)})
	id := mustBegin(t, m.Begin)
	columns := []Column{NewColumn("a", "integer"), NewColumn("b", "integer")}
	batch := [][]any{{1, 2}, {3, 4}, {5, 6}, {7, 8}, {9, 10}}
	if _, err := m.Stage(ctx, id, columns, &sliceSource{batches: [][][]any{batch}}); err != nil {
		t.Fatal(err)
	}
	var sum int64
	if err := db.QueryRow("SELECT sum(a) + sum(b) FROM " + m.TableName(id)).Scan(&sum); err != nil {
		t.Fatal(err)
	}
	if countRows(t, db, m.TableName(id)) != 5 || sum != 55 {
		t.Fatalf("chunked insert: rows %d sum %d", countRows(t, db, m.TableName(id)), sum)
	}
	if _, err := m.Stage(ctx, id, columns, &sliceSource{batches: [][][]any{{{1}}}}); err == nil {
		t.Fatal("expected row width error")
	}
}

type failingSource struct {
	cancel context.CancelFunc
}

func (s *failingSource) Next(context.Context) ([][]any, error) {
	s.cancel()
	return [][]any{{"x"}}, nil
}

func TestCancelledStageIsReleased(t *testing.T) {
	db := openSQLite(t, filepath.Join(t.TempDir(), "stackql.db"))
	m := newSQLiteManager(t, db, sqliteDialect(t))
	id := mustBegin(t, m.Begin)
	ctx, cancel := context.WithCancel(context.Background())
	_, err := m.Stage(ctx, id, []Column{NewColumn("a", "text")}, &failingSource{cancel: cancel})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if len(m.Owned(id)) != 1 {
		t.Fatal("cancelled stage must still be owned for release")
	}
	if err = m.Release(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if tables := stagingTables(t, db); len(tables) != 0 {
		t.Fatalf("tables after release: %v", tables)
	}
}
