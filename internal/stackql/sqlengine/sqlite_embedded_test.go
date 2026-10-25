package sqlengine //nolint:testpackage // Exercises private driver and DSN helpers.

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func newTestEngine(t *testing.T, dsn string) *sqLiteEmbeddedEngine {
	t.Helper()
	eng, err := newSQLiteEmbeddedEngine(Config{DSN: dsn})
	if err != nil {
		t.Fatalf("cannot construct embedded engine for dsn %q: %v", dsn, err)
	}
	t.Cleanup(func() {
		if eng.db != nil {
			_ = eng.db.Close()
		}
	})
	return eng
}

func TestBuildDSNDefault(t *testing.T) {
	dsn, pragmas, err := buildDSN("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(dsn, "file::memory:?") {
		t.Fatalf("default DSN base wrong: %q", dsn)
	}
	if !strings.Contains(dsn, "cache=shared") {
		t.Fatalf("default DSN must preserve cache=shared: %q", dsn)
	}
	if pragmas["busy_timeout"] != "5000" {
		t.Fatalf("default busy_timeout not injected: %v", pragmas)
	}
}

//nolint:gocognit // Preserve the migrated backend regression scenarios.
func TestBuildDSNTranslations(t *testing.T) {
	cases := []struct {
		name            string
		in              string
		wantPragmas     map[string]string
		wantParamSubstr []string
	}{
		{
			name: "legacy shorthands",
			in:   "file:test.db?_busy_timeout=1234&_journal_mode=WAL&_synchronous=NORMAL&_foreign_keys=on",
			wantPragmas: map[string]string{
				"busy_timeout": "1234",
				"journal_mode": "WAL",
				"synchronous":  "NORMAL",
				"foreign_keys": "on",
			},
			wantParamSubstr: []string{
				"busy_timeout(1234)", "journal_mode(WAL)", "synchronous(NORMAL)", "foreign_keys(on)",
			},
		},
		{
			name: "aliases win over primary forms",
			in:   "file:test.db?_busy_timeout=1&_timeout=9&_journal_mode=DELETE&_journal=WAL",
			wantPragmas: map[string]string{
				"busy_timeout": "9",
				"journal_mode": "WAL",
			},
			wantParamSubstr: []string{"busy_timeout(9)", "journal_mode(WAL)"},
		},
		{
			name: "native pragma directives pass through",
			in:   "file:test.db?_pragma=busy_timeout(2500)&_pragma=journal_mode(memory)",
			wantPragmas: map[string]string{
				"busy_timeout": "2500",
				"journal_mode": "memory",
			},
			wantParamSubstr: []string{"busy_timeout(2500)", "journal_mode(memory)"},
		},
		{
			name:        "uri params pass through and busy_timeout injected",
			in:          "file::memory:?cache=shared&mode=memory",
			wantPragmas: map[string]string{"busy_timeout": "5000"},
			wantParamSubstr: []string{
				"cache=shared", "mode=memory", "busy_timeout(5000)",
			},
		},
		{
			name:        "extended legacy pragmas translate",
			in:          "file:test.db?_cache_size=-2000&_recursive_triggers=true&_secure_delete=FAST",
			wantPragmas: map[string]string{"busy_timeout": "5000", "cache_size": "-2000", "recursive_triggers": "true", "secure_delete": "FAST"},
			wantParamSubstr: []string{
				"cache_size(-2000)", "recursive_triggers(true)", "secure_delete(FAST)",
			},
		},
		{
			name:            "loc translates to timezone",
			in:              "file:test.db?_loc=auto",
			wantPragmas:     map[string]string{"busy_timeout": "5000"},
			wantParamSubstr: []string{"_timezone=Local"},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dsn, pragmas, err := buildDSN(tc.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for name, want := range tc.wantPragmas {
				if got := pragmas[name]; got != want {
					t.Fatalf("expected pragma %q=%q, got %q (map %v)", name, want, got, pragmas)
				}
			}
			decoded, decodeErr := url.QueryUnescape(dsn)
			if decodeErr != nil {
				t.Fatalf("cannot decode generated DSN %q: %v", dsn, decodeErr)
			}
			for _, want := range tc.wantParamSubstr {
				if !strings.Contains(decoded, want) {
					t.Fatalf("generated DSN %q lacks %q", decoded, want)
				}
			}
		})
	}
}

func TestBuildDSNRejectsUnknownUnderscoreParam(t *testing.T) {
	for _, dsn := range []string{
		"file:test.db?_mutex=full",
		"file:test.db?_auth_user=admin",
		"file:test.db?_no_such_param=1",
	} {
		if _, _, err := buildDSN(dsn); err == nil {
			t.Fatalf("buildDSN(%q) must reject unknown underscore parameters", dsn)
		}
	}
}

func TestSQLiteEmbeddedExtensionFunctions(t *testing.T) {
	eng := newTestEngine(t, "")
	assertQuery := func(query string, want any) {
		t.Helper()
		var got any
		if err := eng.QueryRow(query).Scan(&got); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
		if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", want) {
			t.Fatalf("%s = %#v, want %#v", query, got, want)
		}
	}
	assertQuery(`SELECT split_part('a,b,c', ',', 2)`, "b")
	assertQuery(`SELECT regexp_like('abc123', '\d+')`, int64(1))
	assertQuery(`SELECT 'the year is 2021' REGEXP '[0-9]+'`, int64(1))
	assertQuery(`SELECT regexp_substr('abc123def', '\d+')`, "123")
	assertQuery(`SELECT regexp_replace('abc123', '\d', 'X')`, "abcXXX")
	assertQuery(`SELECT json_equal('{"a":1}', '{ "a" : 1.0 }')`, int64(1))
	assertQuery(
		`SELECT aws_policy_equal('{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["s3:GetObject"],"Resource":"*"}]}', '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:GetObject","Resource":"*"}]}')`,
		int64(1),
	)
}

func TestSQLiteEmbeddedPragmaApplication(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "pragma.db")
	eng := newTestEngine(t,
		"file:"+dbPath+"?_busy_timeout=1234&_journal_mode=WAL&_synchronous=NORMAL&_foreign_keys=on")
	assertPragma := func(name, want string) {
		t.Helper()
		var got any
		if err := eng.QueryRow("PRAGMA " + name).Scan(&got); err != nil {
			t.Fatalf("PRAGMA %s: %v", name, err)
		}
		if canon := canonicalizePragmaValue(name, pragmaValueToString(got)); canon != want {
			t.Fatalf("PRAGMA %s = %q, want %q", name, canon, want)
		}
	}
	assertPragma("busy_timeout", "1234")
	assertPragma("journal_mode", "wal")
	assertPragma("synchronous", "1")
	assertPragma("foreign_keys", "1")
}

func TestSQLiteEmbeddedPragmaAssertionFailure(t *testing.T) {
	eng := newTestEngine(t, "")
	err := eng.assertPragmas(map[string]string{"busy_timeout": "9999"})
	if err == nil {
		t.Fatal("assertPragmas must fail fast on a pragma mismatch")
	}
	if !strings.Contains(err.Error(), "pragma assertion failed") {
		t.Fatalf("unexpected assertion error: %v", err)
	}
}

func TestSQLiteEmbeddedPlainMemoryPoolCap(t *testing.T) {
	eng := newTestEngine(t, ":memory:")
	if !eng.IsMemory() {
		t.Fatal("plain :memory: engine must report IsMemory")
	}
	if got := eng.db.Stats().MaxOpenConnections; got != 1 {
		t.Fatalf("plain :memory: pool must be capped at 1 connection, got %d", got)
	}
	if _, err := eng.Exec(`CREATE TABLE cap_probe (v TEXT)`); err != nil {
		t.Fatalf("create: %v", err)
	}
	var wg sync.WaitGroup
	errCh := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var c int
			if err := eng.QueryRow(`SELECT count(*) FROM cap_probe`).Scan(&c); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent query on capped memory pool failed: %v", err)
	}
}

func TestSQLiteEmbeddedSharedMemoryMultiConn(t *testing.T) {
	eng := newTestEngine(t, "")
	if !eng.IsMemory() {
		t.Fatal("default engine must report IsMemory")
	}
	if got := eng.db.Stats().MaxOpenConnections; got != 0 {
		t.Fatalf("shared-cache memory pool must stay uncapped, got %d", got)
	}
}

//nolint:gocognit // Preserve the migrated backend regression scenarios.
func TestSQLiteEmbeddedConcurrencyStress(t *testing.T) {
	eng := newTestEngine(t, "")
	if _, err := eng.Exec(`CREATE TABLE stress_t (id INTEGER PRIMARY KEY AUTOINCREMENT, v TEXT)`); err != nil {
		t.Fatalf("create: %v", err)
	}
	const (
		writers      = 2
		readers      = 8
		opsPerWorker = 200
	)
	var wg sync.WaitGroup
	errCh := make(chan error, (writers+readers)*opsPerWorker)
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				if _, err := eng.Exec(`INSERT INTO stress_t (v) VALUES (?)`, fmt.Sprintf("w%d-%d", n, i)); err != nil {
					errCh <- fmt.Errorf("writer %d op %d: %w", n, i, err)
				}
			}
		}(w)
	}
	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				var c int
				if err := eng.QueryRow(`SELECT count(*) FROM stress_t`).Scan(&c); err != nil {
					errCh <- fmt.Errorf("reader %d op %d: %w", n, i, err)
				}
			}
		}(r)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if isBusy(err) {
			t.Fatalf("unhandled SQLITE_BUSY surfaced to a caller: %v", err)
		}
		t.Fatalf("concurrency stress error: %v", err)
	}
	var total int
	if err := eng.QueryRow(`SELECT count(*) FROM stress_t`).Scan(&total); err != nil {
		t.Fatalf("final count: %v", err)
	}
	if total != writers*opsPerWorker {
		t.Fatalf("expected %d rows, got %d", writers*opsPerWorker, total)
	}
}

func TestSQLiteEmbeddedTimestampRoundTrip(t *testing.T) {
	eng := newTestEngine(t, "")
	ddl := `CREATE TABLE ts_probe (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_dttm INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
		created_text TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := eng.Exec(ddl); err != nil {
		t.Fatalf("create: %v", err)
	}
	before := time.Now().UTC().Add(-2 * time.Second).Unix()
	if _, err := eng.Exec(`INSERT INTO ts_probe DEFAULT VALUES`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	after := time.Now().UTC().Add(2 * time.Second).Unix()
	var epoch int64
	var text string
	if err := eng.QueryRow(`SELECT created_dttm, created_text FROM ts_probe WHERE id = 1`).Scan(&epoch, &text); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if epoch < before || epoch > after {
		t.Fatalf("epoch timestamp %d outside window [%d, %d]", epoch, before, after)
	}
	parsed, err := time.Parse("2006-01-02 15:04:05", text)
	if err != nil {
		t.Fatalf("CURRENT_TIMESTAMP text %q failed round trip: %v", text, err)
	}
	if u := parsed.UTC().Unix(); u < before || u > after {
		t.Fatalf("text timestamp %q (%d) outside window [%d, %d]", text, u, before, after)
	}
	// Control-plane style INSERT ... RETURNING with strftime, as used by the
	// generation/session bookkeeping queries.
	var id int64
	if err := eng.QueryRow(
		`INSERT INTO ts_probe (created_dttm) VALUES (strftime('%s', 'now')) RETURNING id`,
	).Scan(&id); err != nil {
		t.Fatalf("insert returning: %v", err)
	}
	if id != 2 {
		t.Fatalf("expected returned id 2, got %d", id)
	}
}

//nolint:gocognit // Preserve the migrated backend regression scenarios.
func TestSQLiteEmbeddedBooleanDecltypeRoundTrip(t *testing.T) {
	eng := newTestEngine(t, "")
	ddl := `CREATE TABLE bool_probe (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		flag BOOLEAN,
		flag_lower boolean,
		plain INTEGER
	)`
	if _, err := eng.Exec(ddl); err != nil {
		t.Fatalf("create: %v", err)
	}
	// 2 and -1 pin the retired driver's val > 0 rule.
	if _, err := eng.Exec(
		`INSERT INTO bool_probe (flag, flag_lower, plain) VALUES (1, 1, 1), (0, 0, 0), (NULL, NULL, NULL), (2, 2, 2), (-1, -1, -1)`,
	); err != nil {
		t.Fatalf("insert: %v", err)
	}
	expected := []struct {
		flag  any
		plain any
	}{
		{true, int64(1)},
		{false, int64(0)},
		{nil, nil},
		{true, int64(2)},
		{false, int64(-1)},
	}
	rows, err := eng.Query(`SELECT flag, flag_lower, plain FROM bool_probe ORDER BY id`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer func() { _ = rows.Close() }()
	i := 0
	for rows.Next() {
		var flag, flagLower, plain any
		if err = rows.Scan(&flag, &flagLower, &plain); err != nil {
			t.Fatalf("scan row %d: %v", i, err)
		}
		if i >= len(expected) {
			t.Fatalf("unexpected extra row %d", i)
		}
		if flag != expected[i].flag {
			t.Fatalf("row %d: BOOLEAN column = %#v, want %#v", i, flag, expected[i].flag)
		}
		if flagLower != expected[i].flag {
			t.Fatalf("row %d: boolean (lowercase decltype) column = %#v, want %#v", i, flagLower, expected[i].flag)
		}
		if plain != expected[i].plain {
			t.Fatalf("row %d: plain INTEGER column = %#v, want %#v (must stay untouched)", i, plain, expected[i].plain)
		}
		i++
	}
	if err = rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	if i != len(expected) {
		t.Fatalf("expected %d rows, got %d", len(expected), i)
	}
	// Typed sinks: bool and sql.NullBool round-trip, NULL preserved.
	var b bool
	if err = eng.QueryRow(`SELECT flag FROM bool_probe WHERE id = 1`).Scan(&b); err != nil || !b {
		t.Fatalf("bool scan = (%v, %v), want (true, nil)", b, err)
	}
	var nb sql.NullBool
	if err = eng.QueryRow(`SELECT flag FROM bool_probe WHERE id = 3`).Scan(&nb); err != nil {
		t.Fatalf("NullBool scan: %v", err)
	}
	if nb.Valid {
		t.Fatalf("NULL in a declared-boolean column must stay NULL, got %+v", nb)
	}
	// Prepared-statement path goes through the same conversion.
	stmt, err := eng.db.Prepare(`SELECT flag FROM bool_probe WHERE id = ?`)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	defer func() { _ = stmt.Close() }()
	var prepared any
	if err = stmt.QueryRow(2).Scan(&prepared); err != nil {
		t.Fatalf("prepared scan: %v", err)
	}
	if prepared != false {
		t.Fatalf("prepared statement BOOLEAN column = %#v, want false", prepared)
	}
	// An expression over a boolean column has no decltype and must stay
	// an INTEGER, exactly as under the retired driver.
	var expr any
	if err = eng.QueryRow(`SELECT flag + 0 FROM bool_probe WHERE id = 1`).Scan(&expr); err != nil {
		t.Fatalf("expression scan: %v", err)
	}
	if expr != int64(1) {
		t.Fatalf("expression column = %#v, want int64(1) (must stay untouched)", expr)
	}
}

func TestSQLiteErrorPredicates(t *testing.T) {
	eng := newTestEngine(t, "")
	if _, err := eng.Exec(`CREATE TABLE pred_t (id INTEGER PRIMARY KEY, v TEXT)`); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := eng.Exec(`INSERT INTO pred_t (id, v) VALUES (1, 'a')`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	_, err := eng.Exec(`INSERT INTO pred_t (id, v) VALUES (1, 'dup')`)
	if err == nil {
		t.Fatal("duplicate insert must fail")
	}
	if !isConstraintViolation(err) {
		t.Fatalf("expected constraint violation classification for %v", err)
	}
	if isBusy(err) {
		t.Fatalf("constraint violation misclassified as busy: %v", err)
	}
	plain := errors.New("not a sqlite error")
	if isBusy(plain) || isConstraintViolation(plain) {
		t.Fatal("non-driver errors must not classify as sqlite errors")
	}
}
