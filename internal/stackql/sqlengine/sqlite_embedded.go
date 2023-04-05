package sqlengine

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/sqlcontrol"
	"github.com/stackql/stackql/internal/stackql/util"

	_ "github.com/stackql/go-sqlite3" //nolint:revive,nolintlint // anonymous import is a pattern for SQL drivers
)

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
	cfg dto.SQLBackendCfg,
	controlAttributes sqlcontrol.ControlAttributes,
) (*sqLiteEmbeddedEngine, error) {
	// SQLite permeits empty DSN and can safely ignore the err
	dsn := cfg.GetDSN()
	if dsn == "" {
		dsn = "file::memory:?cache=shared"
	}
	db, err := sql.Open("sqlite3", dsn)
	db.SetConnMaxLifetime(-1)
	eng := &sqLiteEmbeddedEngine{
		db:                db,
		dsn:               dsn,
		controlAttributes: controlAttributes,
		ctrlMutex:         &sync.Mutex{},
		sessionMutex:      &sync.Mutex{},
		discoveryMutex:    &sync.Mutex{},
	}
	if err != nil {
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
	expF, err := util.GetFilePathFromRepositoryRoot(fileName)
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
	// logging.GetLogger().Infoln(fmt.Sprintf("res= %v, err = %v", res, err))
	return res, err
}

func (se sqLiteEmbeddedEngine) ExecInTxn(queries []string) error {
	txn, err := se.db.Begin()
	if err != nil {
		return err
	}
	for _, query := range queries {
		_, err = txn.Exec(query)
		if err != nil {
			//nolint:errcheck // intentionally ignoring error TODO: publish variadic error(s)
			txn.Rollback()
			return err
		}
	}
	err = txn.Commit()
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

func (se sqLiteEmbeddedEngine) CacheStoreGetAll() ([]internaldto.KeyVal, error) {
	var retVal []internaldto.KeyVal
	//nolint:rowserrcheck // TODO: fix this
	res, err := se.db.Query(`SELECT k, v FROM "__iql__.cache.key_val"`)
	if err != nil {
		return nil, err
	}
	defer res.Close()
	for res.Next() {
		var kv internaldto.KeyVal
		err = res.Scan(&kv.K, &kv.V)
		if err != nil {
			return nil, err
		}
		retVal = append(retVal, kv)
	}
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
	// logging.GetLogger().Infoln(fmt.Sprintf("query = %s", query))
	res, err := se.db.Query(query, varArgs...)
	// logging.GetLogger().Infoln(fmt.Sprintf("res= %v, err = %v", res, err))
	return res, err
}
