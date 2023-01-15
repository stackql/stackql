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

	_ "github.com/snowflakedb/gosnowflake"
)

var (
	_ SQLEngine = &snowflakeTcpEngine{}
)

type snowflakeTcpEngine struct {
	db                *sql.DB
	dsn               string
	controlAttributes sqlcontrol.ControlAttributes
	ctrlMutex         *sync.Mutex
	sessionMutex      *sync.Mutex
	discoveryMutex    *sync.Mutex
}

func (se *snowflakeTcpEngine) IsMemory() bool {
	return false
}

func (se *snowflakeTcpEngine) GetDB() (*sql.DB, error) {
	return se.db, nil
}

func (se *snowflakeTcpEngine) GetTx() (*sql.Tx, error) {
	return se.db.Begin()
}

func newSnowflakeTcpEngine(cfg dto.SQLBackendCfg, controlAttributes sqlcontrol.ControlAttributes) (*snowflakeTcpEngine, error) {
	dsn := cfg.GetDSN()
	if dsn == "" {
		return nil, fmt.Errorf("cannot init snowflake from empty dsn")
	}
	db, err := db_util.GetDB("snowflake", "snowflake", cfg)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxLifetime(-1)
	eng := &snowflakeTcpEngine{
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
		err = eng.execFile(cfg.DbInitFilePath)
	}
	if err != nil {
		return eng, err
	}
	if err != nil {
		return eng, err
	}
	return eng, err
}

func (eng *snowflakeTcpEngine) execFile(fileName string) error {
	fileContents, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	_, err = eng.db.Exec(string(fileContents))
	if err != nil {
		return fmt.Errorf("stackql snowflake db exec file error: %s", err.Error())
	}
	return nil
}

func (eng *snowflakeTcpEngine) execFileLocal(fileName string) error {
	expF, err := util.GetFilePathFromRepositoryRoot(fileName)
	if err != nil {
		return err
	}
	return eng.execFile(expF)
}

func (eng *snowflakeTcpEngine) ExecFileLocal(fileName string) error {
	return eng.execFileLocal(fileName)
}

func (eng *snowflakeTcpEngine) ExecFile(fileName string) error {
	return eng.execFile(fileName)
}

func (se snowflakeTcpEngine) Exec(query string, varArgs ...interface{}) (sql.Result, error) {
	res, err := se.db.Exec(query, varArgs...)
	return res, err
}

func (se snowflakeTcpEngine) QueryRow(query string, varArgs ...interface{}) *sql.Row {
	res := se.db.QueryRow(query, varArgs...)
	return res
}

func (se snowflakeTcpEngine) ExecInTxn(queries []string) error {
	txn, err := se.db.Begin()
	if err != nil {
		return err
	}
	for _, query := range queries {
		_, err = txn.Exec(query)
		if err != nil {
			txn.Rollback()
			return err
		}
	}
	err = txn.Commit()
	return err
}

func (se snowflakeTcpEngine) GetNextGenerationId() (int, error) {
	se.ctrlMutex.Lock()
	defer se.ctrlMutex.Unlock()
	return se.getNextGenerationId()
}

func (se snowflakeTcpEngine) GetCurrentGenerationId() (int, error) {
	se.ctrlMutex.Lock()
	defer se.ctrlMutex.Unlock()
	return se.getCurrentGenerationId()
}

func (se snowflakeTcpEngine) GetNextDiscoveryGenerationId(discoveryName string) (int, error) {
	se.discoveryMutex.Lock()
	defer se.discoveryMutex.Unlock()
	return se.getNextProviderGenerationId(discoveryName)
}

func (se snowflakeTcpEngine) GetCurrentDiscoveryGenerationId(discoveryName string) (int, error) {
	se.discoveryMutex.Lock()
	defer se.discoveryMutex.Unlock()
	return se.getCurrentProviderGenerationId(discoveryName)
}

func (se snowflakeTcpEngine) GetNextSessionId(generationId int) (int, error) {
	se.sessionMutex.Lock()
	defer se.sessionMutex.Unlock()
	return se.getNextSessionId(generationId)
}

func (se snowflakeTcpEngine) GetCurrentSessionId(generationId int) (int, error) {
	se.sessionMutex.Lock()
	defer se.sessionMutex.Unlock()
	return se.getCurrentSessionId(generationId)
}

func (se snowflakeTcpEngine) getCurrentGenerationId() (int, error) {
	var retVal int
	res := se.db.QueryRow(`SELECT lhs.iql_generation_id FROM "__iql__.control.generation" lhs INNER JOIN (SELECT max(created_dttm) AS max_dttm FROM "__iql__.control.generation" WHERE collected_dttm IS null) rhs ON  lhs.created_dttm = rhs.max_dttm WHERE lhs.collected_dttm IS null`)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se snowflakeTcpEngine) getNextGenerationId() (int, error) {
	var retVal int
	res := se.db.QueryRow(`INSERT INTO "__iql__.control.generation" (generation_description, created_dttm) VALUES ('', current_timestamp) RETURNING iql_generation_id`)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se snowflakeTcpEngine) getCurrentProviderGenerationId(providerName string) (int, error) {
	var retVal int
	res := se.db.QueryRow(`SELECT lhs.iql_discovery_generation_id FROM "__iql__.control.discovery_generation" lhs INNER JOIN (SELECT discovery_name, max(created_dttm) AS max_dttm FROM "__iql__.control.discovery_generation" WHERE collected_dttm IS null GROUP BY discovery_name) rhs ON  lhs.created_dttm = rhs.max_dttm AND lhs.discovery_name = rhs.discovery_name WHERE lhs.collected_dttm IS null AND lhs.discovery_name = $1`, providerName)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se snowflakeTcpEngine) getNextProviderGenerationId(providerName string) (int, error) {
	var retVal int
	res := se.db.QueryRow(`INSERT INTO "__iql__.control.discovery_generation" (discovery_name, created_dttm) VALUES ($1, current_timestamp) RETURNING iql_discovery_generation_id`, providerName)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se snowflakeTcpEngine) getCurrentSessionId(generationId int) (int, error) {
	var retVal int
	res := se.db.QueryRow(`SELECT lhs.iql_session_id FROM "__iql__.control.session" lhs INNER JOIN (SELECT max(created_dttm) AS max_dttm FROM "__iql__.control.session" WHERE collected_dttm IS null) rhs ON  lhs.created_dttm = rhs.max_dttm AND lhs.iql_genration_id = rhs.iql_generation_id WHERE lhs.iql_generation_id = $1 AND lhs.collected_dttm IS null`, generationId)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se snowflakeTcpEngine) getNextSessionId(generationId int) (int, error) {
	var retVal int
	res := se.db.QueryRow(`INSERT INTO "__iql__.control.session" (iql_generation_id, created_dttm) VALUES ($1, current_timestamp) RETURNING iql_session_id`, generationId)
	err := res.Scan(&retVal)
	logging.GetLogger().Infoln(fmt.Sprintf("getNextSessionId(): generation id = %d, session id = %d", generationId, retVal))
	return retVal, err
}

func (se snowflakeTcpEngine) CacheStoreGet(key string) ([]byte, error) {
	var retVal []byte
	res := se.db.QueryRow(`SELECT v FROM "__iql__.cache.key_val" WHERE k = $1`, key)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se snowflakeTcpEngine) CacheStoreGetAll() ([]internaldto.KeyVal, error) {
	var retVal []internaldto.KeyVal
	res, err := se.db.Query(`SELECT k, v FROM "__iql__.cache.key_val"`)
	if err != nil {
		return nil, err
	}
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

func (se snowflakeTcpEngine) CacheStorePut(key string, val []byte, tablespace string, tablespaceID int) error {
	txn, err := se.db.Begin()
	if err != nil {
		return err
	}
	_, err = txn.Exec(`DELETE FROM "__iql__.cache.key_val" WHERE k = $1`, key)
	if err != nil {
		txn.Rollback()
		return err
	}
	_, err = txn.Exec(`INSERT INTO "__iql__.cache.key_val" (k, v, tablespace, tablespace_id) VALUES($1, $2, $3, $4)`, key, val, tablespace, tablespaceID)
	if err != nil {
		txn.Rollback()
		return err
	}
	err = txn.Commit()
	return err
}

func (se snowflakeTcpEngine) Query(query string, varArgs ...interface{}) (*sql.Rows, error) {
	return se.query(query, varArgs...)
}

func (se snowflakeTcpEngine) query(query string, varArgs ...interface{}) (*sql.Rows, error) {
	res, err := se.db.Query(query, varArgs...)
	return res, err
}
