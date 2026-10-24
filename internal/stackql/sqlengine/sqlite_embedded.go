package sqlengine

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/stackql/stackql/internal/stackql/logging"
	sqlcontrol "github.com/stackql/stackql/internal/stackql/sqlengine/control"
	"github.com/stackql/stackql/internal/stackql/sqlengine/internal/sqlfuncs"

	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

const (
	// sqliteDriverName publishes the modernc.org/sqlite driver with the
	// StackQL extension functions registered.
	sqliteDriverName = "stackql-sqlite"
	sqliteDefaultDSN = "file::memory:?cache=shared"
	// sqliteDefaultBusyTimeoutMillis preserves the retired cgo driver's
	// per-connection default; modernc defaults to 0.
	sqliteDefaultBusyTimeoutMillis = "5000"
)

//nolint:gochecknoglobals // database/sql driver registration must occur once per process.
var (
	sqliteDriverRegisterOnce    sync.Once
	errSQLiteDriverRegistration error
)

// registerSQLiteDriver idempotently publishes the sqlfuncs-equipped driver
// under sqliteDriverName, wrapped for mattn-parity boolean decltype
// conversion; a registration error surfaces on every attempt.
func registerSQLiteDriver() error {
	sqliteDriverRegisterOnce.Do(func() {
		drv := &sqlite.Driver{}
		if err := sqlfuncs.Register(drv); err != nil {
			errSQLiteDriverRegistration = err
			return
		}
		sql.Register(sqliteDriverName, newBoolDecltypeDriver(drv))
	})
	return errSQLiteDriverRegistration
}

var (
	_ SQLEngine = &sqLiteEmbeddedEngine{}
)

type sqLiteEmbeddedEngine struct {
	db                *sql.DB
	dsn               string
	controlAttributes sqlcontrol.ControlAttributes
	ctrlMutex         *sync.Mutex
	sessionMutex      *sync.Mutex
	discoveryMutex    *sync.Mutex
}

func (se *sqLiteEmbeddedEngine) IsMemory() bool {
	return strings.Contains(se.dsn, ":memory:") || strings.Contains(se.dsn, "mode=memory")
}

func (se *sqLiteEmbeddedEngine) GetDB() (*sql.DB, error) {
	return se.db, nil
}

func (se *sqLiteEmbeddedEngine) GetTx() (*sql.Tx, error) {
	return se.db.Begin()
}

func newSQLiteEmbeddedEngine(
	cfg Config,
	controlAttributes sqlcontrol.ControlAttributes,
) (*sqLiteEmbeddedEngine, error) {
	dsn, expectedPragmas, err := buildDSN(cfg.DSN)
	eng := &sqLiteEmbeddedEngine{
		dsn:               dsn,
		controlAttributes: controlAttributes,
		ctrlMutex:         &sync.Mutex{},
		sessionMutex:      &sync.Mutex{},
		discoveryMutex:    &sync.Mutex{},
	}
	if err != nil {
		return eng, err
	}
	if err = registerSQLiteDriver(); err != nil {
		return eng, err
	}
	db, err := sql.Open(sqliteDriverName, dsn)
	db.SetConnMaxLifetime(-1)
	eng.db = db
	if err != nil {
		return eng, err
	}
	if eng.IsMemory() && !strings.Contains(dsn, "cache=shared") {
		// each pooled connection to a non-shared in-memory DSN would be a
		// separate database
		db.SetMaxOpenConns(1)
	}
	if err = eng.assertPragmas(expectedPragmas); err != nil {
		return eng, err
	}
	if cfg.DbInitFilePath != "" {
		err = eng.execFileSQLite(cfg.DbInitFilePath)
	}
	if err != nil {
		return eng, err
	}
	logging.GetLogger().Infoln(fmt.Sprintf("opened db with file = '%s' and err  = '%v'", dsn, err))
	return eng, err
}

func (se *sqLiteEmbeddedEngine) execFileSQLite(fileName string) error {
	fileContents, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}
	_, err = se.db.Exec(string(fileContents))
	return err
}

func (se *sqLiteEmbeddedEngine) execFileLocal(fileName string) error {
	expF, err := getFilePathFromRepositoryRoot(fileName)
	if err != nil {
		return err
	}
	return se.execFileSQLite(expF)
}

func (se *sqLiteEmbeddedEngine) ExecFileLocal(fileName string) error {
	return se.execFileLocal(fileName)
}

func (se *sqLiteEmbeddedEngine) ExecFile(fileName string) error {
	return se.execFileSQLite(fileName)
}

func (se sqLiteEmbeddedEngine) Exec(query string, varArgs ...interface{}) (sql.Result, error) {
	// logging.GetLogger().Infoln(fmt.Sprintf("exec query = %s", query))
	res, err := se.db.Exec(query, varArgs...)
	classifySQLiteError(err)
	// logging.GetLogger().Infoln(fmt.Sprintf("res= %v, err = %v", res, err))
	return res, err
}

func (se sqLiteEmbeddedEngine) ExecInTxn(queries []string) error {
	txn, err := se.db.Begin()
	if err != nil {
		classifySQLiteError(err)
		return err
	}
	for _, query := range queries {
		_, err = txn.Exec(query)
		if err != nil {
			classifySQLiteError(err)
			//nolint:errcheck // intentionally ignoring error TODO: publish variadic error(s)
			txn.Rollback()
			return err
		}
	}
	err = txn.Commit()
	classifySQLiteError(err)
	return err
}

func (se sqLiteEmbeddedEngine) GetNextGenerationID() (int, error) {
	se.ctrlMutex.Lock()
	defer se.ctrlMutex.Unlock()
	return se.getNextGenerationID()
}

func (se sqLiteEmbeddedEngine) GetCurrentGenerationID() (int, error) {
	se.ctrlMutex.Lock()
	defer se.ctrlMutex.Unlock()
	return se.getCurrentGenerationID()
}

func (se sqLiteEmbeddedEngine) GetNextDiscoveryGenerationID(discoveryName string) (int, error) {
	se.discoveryMutex.Lock()
	defer se.discoveryMutex.Unlock()
	return se.getNextProviderGenerationID(discoveryName)
}

func (se sqLiteEmbeddedEngine) GetCurrentDiscoveryGenerationID(discoveryName string) (int, error) {
	se.discoveryMutex.Lock()
	defer se.discoveryMutex.Unlock()
	return se.getCurrentProviderGenerationID(discoveryName)
}

func (se sqLiteEmbeddedEngine) GetNextSessionID(generationID int) (int, error) {
	se.sessionMutex.Lock()
	defer se.sessionMutex.Unlock()
	return se.getNextSessionID(generationID)
}

func (se sqLiteEmbeddedEngine) GetCurrentSessionID(generationID int) (int, error) {
	se.sessionMutex.Lock()
	defer se.sessionMutex.Unlock()
	return se.getCurrentSessionID(generationID)
}

func (se sqLiteEmbeddedEngine) getCurrentGenerationID() (int, error) {
	var retVal int
	//nolint:lll // long SQL query
	res := se.db.QueryRow(`SELECT lhs.iql_generation_id FROM "__iql__.control.generation" lhs INNER JOIN (SELECT max(created_dttm) AS max_dttm FROM "__iql__.control.generation" WHERE collected_dttm IS null) rhs ON  lhs.created_dttm = rhs.max_dttm WHERE lhs.collected_dttm IS null`)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se sqLiteEmbeddedEngine) QueryRow(query string, varArgs ...interface{}) *sql.Row {
	res := se.db.QueryRow(query, varArgs...)
	return res
}

func (se sqLiteEmbeddedEngine) getNextGenerationID() (int, error) {
	var retVal int
	//nolint:lll,execinquery // long SQL query and `execinquery` is DEAD SET RUBBISH for INSERT... RETURNING
	res := se.db.QueryRow(`INSERT INTO "__iql__.control.generation" (generation_description, created_dttm) VALUES ('', strftime('%s', 'now')) RETURNING iql_generation_id`)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se sqLiteEmbeddedEngine) getCurrentProviderGenerationID(providerName string) (int, error) {
	var retVal int
	//nolint:lll // long SQL query
	res := se.db.QueryRow(`SELECT lhs.iql_discovery_generation_id FROM "__iql__.control.discovery_generation" lhs INNER JOIN (SELECT discovery_name, max(created_dttm) AS max_dttm FROM "__iql__.control.discovery_generation" WHERE collected_dttm IS null GROUP BY discovery_name) rhs ON  lhs.created_dttm = rhs.max_dttm AND lhs.discovery_name = rhs.discovery_name WHERE lhs.collected_dttm IS null AND lhs.discovery_name = ?`, providerName)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se sqLiteEmbeddedEngine) getNextProviderGenerationID(providerName string) (int, error) {
	var retVal int
	//nolint:lll,execinquery // long SQL query and `execinquery` is DEAD SET RUBBISH for INSERT... RETURNING
	res := se.db.QueryRow(`INSERT INTO "__iql__.control.discovery_generation" (discovery_name, created_dttm) VALUES (?, strftime('%s', 'now')) RETURNING iql_discovery_generation_id`, providerName)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se sqLiteEmbeddedEngine) getCurrentSessionID(generationID int) (int, error) {
	var retVal int
	//nolint:lll // long SQL query
	res := se.db.QueryRow(`SELECT lhs.iql_session_id FROM "__iql__.control.session" lhs INNER JOIN (SELECT max(created_dttm) AS max_dttm FROM "__iql__.control.session" WHERE collected_dttm IS null) rhs ON  lhs.created_dttm = rhs.max_dttm AND lhs.iql_genration_id = rhs.iql_generation_id WHERE lhs.iql_generation_id = ? AND lhs.collected_dttm IS null`, generationID)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se sqLiteEmbeddedEngine) getNextSessionID(generationID int) (int, error) {
	var retVal int
	//nolint:lll,execinquery // long SQL query and `execinquery` is DEAD SET RUBBISH for INSERT... RETURNING
	res := se.db.QueryRow(`INSERT INTO "__iql__.control.session" (iql_generation_id, created_dttm) VALUES (?, strftime('%s', 'now')) RETURNING iql_session_id`, generationID)
	err := res.Scan(&retVal)
	logging.GetLogger().Infoln(
		fmt.Sprintf(
			"getNextSessionID(): generation id = %d, session id = %d",
			generationID,
			retVal,
		),
	)
	return retVal, err
}

func (se sqLiteEmbeddedEngine) CacheStoreGet(key string) ([]byte, error) {
	var retVal []byte
	res := se.db.QueryRow(`SELECT v FROM "__iql__.cache.key_val" WHERE k = ?`, key)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se sqLiteEmbeddedEngine) CacheStorePut(key string, val []byte, tablespace string, tablespaceID int) error {
	txn, err := se.db.Begin()
	if err != nil {
		return err
	}
	_, err = txn.Exec(`DELETE FROM "__iql__.cache.key_val" WHERE k = ?`, key)
	if err != nil {
		//nolint:errcheck // intentionally ignoring error TODO: publish variadic error(s)
		txn.Rollback()
		return err
	}
	_, err = txn.Exec(
		`INSERT INTO "__iql__.cache.key_val" (k, v, tablespace, tablespace_id) VALUES(?, ?, ?, ?)`,
		key,
		val,
		tablespace,
		tablespaceID,
	)
	if err != nil {
		//nolint:errcheck // intentionally ignoring error TODO: publish variadic error(s)
		txn.Rollback()
		return err
	}
	err = txn.Commit()
	return err
}

func (se sqLiteEmbeddedEngine) Query(query string, varArgs ...interface{}) (*sql.Rows, error) {
	return se.query(query, varArgs...)
}

func (se sqLiteEmbeddedEngine) query(query string, varArgs ...interface{}) (*sql.Rows, error) {
	logging.GetLogger().Debugln(fmt.Sprintf("sqlite embedded raw query = %s, varArgs = %v", query, varArgs))
	res, err := se.db.Query(query, varArgs...)
	// logging.GetLogger().Infoln(fmt.Sprintf("res= %v, err = %v", res, err))
	return res, err
}

// isBusy reports whether err carries primary result code SQLITE_BUSY.
func isBusy(err error) bool {
	return hasSQLitePrimaryCode(err, sqlite3.SQLITE_BUSY)
}

// isConstraintViolation reports whether err carries primary result code
// SQLITE_CONSTRAINT.
func isConstraintViolation(err error) bool {
	return hasSQLitePrimaryCode(err, sqlite3.SQLITE_CONSTRAINT)
}

// hasSQLitePrimaryCode compares the primary (low byte) result code; the
// driver enables extended result codes.
func hasSQLitePrimaryCode(err error, code int) bool {
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		return sqliteErr.Code()&0xff == code
	}
	return false
}

// classifySQLiteError logs whether a write-path error is lock contention or
// a constraint failure.
func classifySQLiteError(err error) {
	if err == nil {
		return
	}
	switch {
	case isBusy(err):
		logging.GetLogger().Debugln(fmt.Sprintf("sqlite embedded: busy: %v", err))
	case isConstraintViolation(err):
		logging.GetLogger().Debugln(fmt.Sprintf("sqlite embedded: constraint violation: %v", err))
	}
}

// legacyParamTranslations maps the retired cgo driver's DSN shorthands to
// pragmas. Order matters: a later alias overrides its primary form, matching
// legacy precedence. modernc silently ignores unknown shorthands, so every
// one is translated explicitly.
//
//nolint:gochecknoglobals // Immutable driver configuration lookup.
var legacyParamTranslations = []struct {
	param  string
	pragma string
}{
	{"_busy_timeout", "busy_timeout"},
	{"_timeout", "busy_timeout"},
	{"_journal_mode", "journal_mode"},
	{"_journal", "journal_mode"},
	{"_synchronous", "synchronous"},
	{"_sync", "synchronous"},
	{"_foreign_keys", "foreign_keys"},
	{"_fk", "foreign_keys"},
	{"_auto_vacuum", "auto_vacuum"},
	{"_vacuum", "auto_vacuum"},
	{"_cache_size", "cache_size"},
	{"_case_sensitive_like", "case_sensitive_like"},
	{"_cslike", "case_sensitive_like"},
	{"_defer_foreign_keys", "defer_foreign_keys"},
	{"_defer_fk", "defer_foreign_keys"},
	{"_ignore_check_constraints", "ignore_check_constraints"},
	{"_query_only", "query_only"},
	{"_recursive_triggers", "recursive_triggers"},
	{"_rt", "recursive_triggers"},
	{"_secure_delete", "secure_delete"},
	{"_writable_schema", "writable_schema"},
}

// moderncNativeParams pass through buildDSN verbatim.
//
//nolint:gochecknoglobals // Immutable driver configuration lookup.
var moderncNativeParams = map[string]struct{}{
	"_pragma":              {},
	"_time_format":         {},
	"_time_integer_format": {},
	"_timezone":            {},
	"_txlock":              {},
	"_inttotime":           {},
	"_texttotime":          {},
	"_defensive":           {},
	"_error_rc":            {},
}

// nonQueryablePragmas cannot be read back, so the startup assertion skips
// them.
//
//nolint:gochecknoglobals // Immutable driver configuration lookup.
var nonQueryablePragmas = map[string]struct{}{
	"case_sensitive_like": {},
}

var pragmaNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// buildDSN is the single place embedded-engine DSNs are constructed. It
// translates legacy `_param=value` shorthands to modernc `_pragma=name(value)`
// directives, passes through modernc-native and plain URI parameters, rejects
// unknown underscore parameters (modernc would silently ignore them), and
// injects the 5000ms legacy busy_timeout default when unset. An empty dsn
// selects the default shared in-memory database. The returned map records
// every pragma the DSN sets, for the startup assertion.
//
//nolint:gocognit,funlen // Preserve the migrated DSN translation rules during backend relocation.
func buildDSN(dsn string) (string, map[string]string, error) {
	if dsn == "" {
		dsn = sqliteDefaultDSN
	}
	base, rawQuery, _ := strings.Cut(dsn, "?")
	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		return "", nil, fmt.Errorf("invalid sqlite DSN query %q: %w", rawQuery, err)
	}
	expectedPragmas := map[string]string{}
	translatedPragmas := map[string]string{}
	outParams := url.Values{}
	for key, vals := range query {
		if len(vals) == 0 {
			continue
		}
		if key == "_pragma" {
			for _, directive := range vals {
				name, value, valued := parsePragmaDirective(directive)
				if valued {
					expectedPragmas[name] = value
				}
			}
			outParams["_pragma"] = append([]string{}, vals...)
			continue
		}
		if _, isNative := moderncNativeParams[key]; isNative {
			outParams[key] = vals
			continue
		}
		if key == "_loc" {
			// legacy time-location parameter; maps to _timezone, "auto"
			// meaning the process-local zone
			if query.Has("_timezone") {
				return "", nil, fmt.Errorf("conflicting sqlite DSN parameters: _loc and _timezone")
			}
			loc := vals[len(vals)-1]
			if strings.EqualFold(loc, "auto") {
				loc = "Local"
			}
			outParams.Set("_timezone", loc)
			continue
		}
		if isLegacyParam(key) {
			continue // handled in declaration order below
		}
		if strings.HasPrefix(key, "_") {
			return "", nil, fmt.Errorf("unsupported sqlite DSN parameter %q: no modernc.org/sqlite translation", key)
		}
		outParams[key] = vals
	}
	// declaration order makes an alias override its primary form
	for _, tr := range legacyParamTranslations {
		if vals := query[tr.param]; len(vals) > 0 {
			translatedPragmas[tr.pragma] = vals[len(vals)-1]
		}
	}
	for name, value := range translatedPragmas {
		expectedPragmas[name] = value
	}
	if _, hasBusy := expectedPragmas["busy_timeout"]; !hasBusy {
		translatedPragmas["busy_timeout"] = sqliteDefaultBusyTimeoutMillis
		expectedPragmas["busy_timeout"] = sqliteDefaultBusyTimeoutMillis
	}
	translatedNames := make([]string, 0, len(translatedPragmas))
	for name := range translatedPragmas {
		translatedNames = append(translatedNames, name)
	}
	sort.Strings(translatedNames)
	directives := make([]string, 0, len(translatedNames)+len(outParams["_pragma"]))
	for _, name := range translatedNames {
		directives = append(directives, fmt.Sprintf("%s(%s)", name, translatedPragmas[name]))
	}
	directives = append(directives, outParams["_pragma"]...)
	outParams["_pragma"] = directives
	return base + "?" + outParams.Encode(), expectedPragmas, nil
}

// parsePragmaDirective splits a `name(value)` or bare `name` directive;
// valued reports whether a value was supplied.
func parsePragmaDirective(directive string) (string, string, bool) {
	name, rest, found := strings.Cut(directive, "(")
	name = strings.TrimSpace(name)
	if !found {
		return name, "", false
	}
	value := strings.TrimSuffix(strings.TrimSpace(rest), ")")
	return name, value, true
}

func isLegacyParam(key string) bool {
	for _, tr := range legacyParamTranslations {
		if tr.param == key {
			return true
		}
	}
	return false
}

// canonicalizePragmaValue reduces equivalent pragma value spellings to one
// form so requested and reported values compare.
func canonicalizePragmaValue(name, value string) string {
	v := strings.ToLower(strings.TrimSpace(value))
	switch name {
	case "foreign_keys", "defer_foreign_keys", "ignore_check_constraints",
		"query_only", "recursive_triggers", "writable_schema", "case_sensitive_like":
		switch v {
		case "1", "true", "yes", "on":
			return "1"
		case "0", "false", "no", "off":
			return "0"
		}
	case "synchronous":
		switch v {
		case "off":
			return "0"
		case "normal":
			return "1"
		case "full":
			return "2"
		case "extra":
			return "3"
		}
	case "auto_vacuum":
		switch v {
		case "none":
			return "0"
		case "full":
			return "1"
		case "incremental":
			return "2"
		}
	case "secure_delete":
		switch v {
		case "off", "false":
			return "0"
		case "on", "true":
			return "1"
		case "fast":
			return "2"
		}
	}
	return v
}

// assertPragmas reads back every pragma the DSN set and fails fast on
// mismatch, so a translation that stops taking effect cannot fail silently.
func (se *sqLiteEmbeddedEngine) assertPragmas(expected map[string]string) error {
	names := make([]string, 0, len(expected))
	for name := range expected {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if _, skip := nonQueryablePragmas[name]; skip {
			continue
		}
		if !pragmaNamePattern.MatchString(name) {
			return fmt.Errorf("pragma assertion: invalid pragma name %q", name)
		}
		var raw any
		err := se.db.QueryRow("PRAGMA " + name).Scan(&raw)
		if errors.Is(err, sql.ErrNoRows) {
			// command-style pragmas report nothing
			continue
		}
		if err != nil {
			return fmt.Errorf("pragma assertion: cannot query pragma %q: %w", name, err)
		}
		actual := canonicalizePragmaValue(name, pragmaValueToString(raw))
		want := canonicalizePragmaValue(name, expected[name])
		if actual != want {
			if name == "journal_mode" && se.IsMemory() && actual == "memory" {
				// SQLite coerces in-memory journal mode to "memory"
				continue
			}
			return fmt.Errorf("pragma assertion failed: %q is %q, requested %q", name, actual, want)
		}
	}
	return nil
}

func pragmaValueToString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []byte:
		return string(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return strconv.FormatFloat(t, 'g', -1, 64)
	default:
		return fmt.Sprintf("%v", t)
	}
}
