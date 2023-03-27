//nolint:dupl,nolintlint //TODO: fix this
package sqlengine

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/stackql/stackql/internal/stackql/db_util"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/sqlcontrol"
	"github.com/stackql/stackql/internal/stackql/util"

	_ "github.com/jackc/pgx/v5/stdlib" //nolint:revive,nolintlint // anonymous import is a pattern for SQL drivers
)

var (
	_ SQLEngine = &postgresTCPEngine{}
)

type postgresTCPEngine struct {
	db                *sql.DB
	dsn               string
	controlAttributes sqlcontrol.ControlAttributes
	ctrlMutex         *sync.Mutex
	sessionMutex      *sync.Mutex
	discoveryMutex    *sync.Mutex
}

func (se *postgresTCPEngine) IsMemory() bool {
	return false
}

func (se *postgresTCPEngine) GetDB() (*sql.DB, error) {
	return se.db, nil
}

func (se *postgresTCPEngine) GetTx() (*sql.Tx, error) {
	return se.db.Begin()
}

func newPostgresTCPEngine(
	cfg dto.SQLBackendCfg,
	controlAttributes sqlcontrol.ControlAttributes,
) (*postgresTCPEngine, error) {
	dsn := cfg.GetDSN()
	if dsn == "" {
		return nil, fmt.Errorf("cannot init postgres from empty dsn")
	}
	db, err := db_util.GetDB("pgx", "postgres", cfg)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxLifetime(-1)
	eng := &postgresTCPEngine{
		db:                db,
		dsn:               dsn,
		controlAttributes: controlAttributes,
		ctrlMutex:         &sync.Mutex{},
		sessionMutex:      &sync.Mutex{},
		discoveryMutex:    &sync.Mutex{},
	}
	if cfg.DbInitFilePath != "" {
		err = eng.execFile(cfg.DbInitFilePath)
	}
	if err != nil {
		return eng, err
	}
	return eng, err
}

func (se *postgresTCPEngine) execFile(fileName string) error {
	fileContents, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	_, err = se.db.Exec(string(fileContents))
	if err != nil {
		return fmt.Errorf("stackql postgres db exec file error: %w", err)
	}
	return nil
}

func (se *postgresTCPEngine) execFileLocal(fileName string) error {
	expF, err := util.GetFilePathFromRepositoryRoot(fileName)
	if err != nil {
		return err
	}
	return se.execFile(expF)
}

func (se *postgresTCPEngine) ExecFileLocal(fileName string) error {
	return se.execFileLocal(fileName)
}

func (se *postgresTCPEngine) ExecFile(fileName string) error {
	return se.execFile(fileName)
}

func (se postgresTCPEngine) Exec(query string, varArgs ...interface{}) (sql.Result, error) {
	res, err := se.db.Exec(query, varArgs...)
	return res, err
}

func (se postgresTCPEngine) QueryRow(query string, varArgs ...interface{}) *sql.Row {
	res := se.db.QueryRow(query, varArgs...)
	return res
}

func (se postgresTCPEngine) ExecInTxn(queries []string) error {
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

func (se postgresTCPEngine) GetNextGenerationID() (int, error) {
	se.ctrlMutex.Lock()
	defer se.ctrlMutex.Unlock()
	return se.getNextGenerationID()
}

func (se postgresTCPEngine) GetCurrentGenerationID() (int, error) {
	se.ctrlMutex.Lock()
	defer se.ctrlMutex.Unlock()
	return se.getCurrentGenerationID()
}

func (se postgresTCPEngine) GetNextDiscoveryGenerationID(discoveryName string) (int, error) {
	se.discoveryMutex.Lock()
	defer se.discoveryMutex.Unlock()
	return se.getNextProviderGenerationID(discoveryName)
}

func (se postgresTCPEngine) GetCurrentDiscoveryGenerationID(discoveryName string) (int, error) {
	se.discoveryMutex.Lock()
	defer se.discoveryMutex.Unlock()
	return se.getCurrentProviderGenerationID(discoveryName)
}

func (se postgresTCPEngine) GetNextSessionID(generationID int) (int, error) {
	se.sessionMutex.Lock()
	defer se.sessionMutex.Unlock()
	return se.getNextSessionID(generationID)
}

func (se postgresTCPEngine) GetCurrentSessionID(generationID int) (int, error) {
	se.sessionMutex.Lock()
	defer se.sessionMutex.Unlock()
	return se.getCurrentSessionID(generationID)
}

func (se postgresTCPEngine) getCurrentGenerationID() (int, error) {
	var retVal int
	//nolint:lll // long SQL query
	res := se.db.QueryRow(`SELECT lhs.iql_generation_id FROM "__iql__.control.generation" lhs INNER JOIN (SELECT max(created_dttm) AS max_dttm FROM "__iql__.control.generation" WHERE collected_dttm IS null) rhs ON  lhs.created_dttm = rhs.max_dttm WHERE lhs.collected_dttm IS null`)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se postgresTCPEngine) getNextGenerationID() (int, error) {
	var retVal int
	//nolint:lll,execinquery // long SQL query and `execinquery` is DEAD SET RUBBISH for INSERT... RETURNING
	res := se.db.QueryRow(`INSERT INTO "__iql__.control.generation" (generation_description, created_dttm) VALUES ('', current_timestamp) RETURNING iql_generation_id`)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se postgresTCPEngine) getCurrentProviderGenerationID(providerName string) (int, error) {
	var retVal int
	//nolint:lll // long SQL query
	res := se.db.QueryRow(`SELECT lhs.iql_discovery_generation_id FROM "__iql__.control.discovery_generation" lhs INNER JOIN (SELECT discovery_name, max(created_dttm) AS max_dttm FROM "__iql__.control.discovery_generation" WHERE collected_dttm IS null GROUP BY discovery_name) rhs ON  lhs.created_dttm = rhs.max_dttm AND lhs.discovery_name = rhs.discovery_name WHERE lhs.collected_dttm IS null AND lhs.discovery_name = $1`, providerName)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se postgresTCPEngine) getNextProviderGenerationID(providerName string) (int, error) {
	var retVal int
	//nolint:lll,execinquery // long SQL query and `execinquery` is DEAD SET RUBBISH for INSERT... RETURNING
	res := se.db.QueryRow(`INSERT INTO "__iql__.control.discovery_generation" (discovery_name, created_dttm) VALUES ($1, current_timestamp) RETURNING iql_discovery_generation_id`, providerName)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se postgresTCPEngine) getCurrentSessionID(generationID int) (int, error) {
	var retVal int
	//nolint:lll // long SQL query
	res := se.db.QueryRow(`SELECT lhs.iql_session_id FROM "__iql__.control.session" lhs INNER JOIN (SELECT max(created_dttm) AS max_dttm FROM "__iql__.control.session" WHERE collected_dttm IS null) rhs ON  lhs.created_dttm = rhs.max_dttm AND lhs.iql_genration_id = rhs.iql_generation_id WHERE lhs.iql_generation_id = $1 AND lhs.collected_dttm IS null`, generationID)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se postgresTCPEngine) getNextSessionID(generationID int) (int, error) {
	var retVal int
	//nolint:lll,execinquery // long SQL query and `execinquery` is DEAD SET RUBBISH for INSERT... RETURNING
	res := se.db.QueryRow(`INSERT INTO "__iql__.control.session" (iql_generation_id, created_dttm) VALUES ($1, current_timestamp) RETURNING iql_session_id`, generationID)
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

func (se postgresTCPEngine) CacheStoreGet(key string) ([]byte, error) {
	var retVal []byte
	res := se.db.QueryRow(`SELECT v FROM "__iql__.cache.key_val" WHERE k = $1`, key)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se postgresTCPEngine) CacheStoreGetAll() ([]internaldto.KeyVal, error) {
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

func (se postgresTCPEngine) CacheStorePut(key string, val []byte, tablespace string, tablespaceID int) error {
	txn, err := se.db.Begin()
	if err != nil {
		return err
	}
	_, err = txn.Exec(`DELETE FROM "__iql__.cache.key_val" WHERE k = $1`, key)
	if err != nil {
		//nolint:errcheck // intentionally ignoring error TODO: publish variadic error(s)
		txn.Rollback()
		return err
	}
	_, err = txn.Exec(
		`INSERT INTO "__iql__.cache.key_val" (k, v, tablespace, tablespace_id) VALUES($1, $2, $3, $4)`,
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

func (se postgresTCPEngine) Query(query string, varArgs ...interface{}) (*sql.Rows, error) {
	return se.query(query, varArgs...)
}

func (se postgresTCPEngine) query(query string, varArgs ...interface{}) (*sql.Rows, error) {
	res, err := se.db.Query(query, varArgs...)
	return res, err
}
