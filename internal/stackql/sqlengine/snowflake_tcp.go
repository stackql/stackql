//nolint:dupl,nolintlint //TODO: fix this
package sqlengine

import (
	"database/sql"
	"fmt"
	"os"
	"sync"

	"github.com/stackql/stackql/internal/stackql/db_util"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/sqlcontrol"
	"github.com/stackql/stackql/internal/stackql/util"

	_ "github.com/snowflakedb/gosnowflake" //nolint:revive,nolintlint // anonymous import is a pattern for SQL drivers
)

var (
	_ SQLEngine = &snowflakeTCPEngine{}
)

type snowflakeTCPEngine struct {
	db                *sql.DB
	dsn               string
	controlAttributes sqlcontrol.ControlAttributes
	ctrlMutex         *sync.Mutex
	sessionMutex      *sync.Mutex
	discoveryMutex    *sync.Mutex
}

func (se *snowflakeTCPEngine) IsMemory() bool {
	return false
}

func (se *snowflakeTCPEngine) GetDB() (*sql.DB, error) {
	return se.db, nil
}

func (se *snowflakeTCPEngine) GetTx() (*sql.Tx, error) {
	return se.db.Begin()
}

func newSnowflakeTCPEngine(
	cfg dto.SQLBackendCfg,
	controlAttributes sqlcontrol.ControlAttributes,
) (*snowflakeTCPEngine, error) {
	dsn := cfg.GetDSN()
	if dsn == "" {
		return nil, fmt.Errorf("cannot init snowflake from empty dsn")
	}
	db, err := db_util.GetDB("snowflake", "snowflake", cfg)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxLifetime(-1)
	eng := &snowflakeTCPEngine{
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

func (se *snowflakeTCPEngine) execFile(fileName string) error {
	fileContents, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}
	_, err = se.db.Exec(string(fileContents))
	if err != nil {
		return fmt.Errorf("stackql snowflake db exec file error: %w", err)
	}
	return nil
}

func (se *snowflakeTCPEngine) execFileLocal(fileName string) error {
	expF, err := util.GetFilePathFromRepositoryRoot(fileName)
	if err != nil {
		return err
	}
	return se.execFile(expF)
}

func (se *snowflakeTCPEngine) ExecFileLocal(fileName string) error {
	return se.execFileLocal(fileName)
}

func (se *snowflakeTCPEngine) ExecFile(fileName string) error {
	return se.execFile(fileName)
}

func (se snowflakeTCPEngine) Exec(query string, varArgs ...interface{}) (sql.Result, error) {
	res, err := se.db.Exec(query, varArgs...)
	return res, err
}

func (se snowflakeTCPEngine) QueryRow(query string, varArgs ...interface{}) *sql.Row {
	res := se.db.QueryRow(query, varArgs...)
	return res
}

func (se snowflakeTCPEngine) ExecInTxn(queries []string) error {
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

func (se snowflakeTCPEngine) GetNextGenerationID() (int, error) {
	se.ctrlMutex.Lock()
	defer se.ctrlMutex.Unlock()
	return se.getNextGenerationID()
}

func (se snowflakeTCPEngine) GetCurrentGenerationID() (int, error) {
	se.ctrlMutex.Lock()
	defer se.ctrlMutex.Unlock()
	return se.getCurrentGenerationID()
}

func (se snowflakeTCPEngine) GetNextDiscoveryGenerationID(discoveryName string) (int, error) {
	se.discoveryMutex.Lock()
	defer se.discoveryMutex.Unlock()
	return se.getNextProviderGenerationID(discoveryName)
}

func (se snowflakeTCPEngine) GetCurrentDiscoveryGenerationID(discoveryName string) (int, error) {
	se.discoveryMutex.Lock()
	defer se.discoveryMutex.Unlock()
	return se.getCurrentProviderGenerationID(discoveryName)
}

func (se snowflakeTCPEngine) GetNextSessionID(generationID int) (int, error) {
	se.sessionMutex.Lock()
	defer se.sessionMutex.Unlock()
	return se.getNextSessionID(generationID)
}

func (se snowflakeTCPEngine) GetCurrentSessionID(generationID int) (int, error) {
	se.sessionMutex.Lock()
	defer se.sessionMutex.Unlock()
	return se.getCurrentSessionID(generationID)
}

func (se snowflakeTCPEngine) getCurrentGenerationID() (int, error) {
	var retVal int
	//nolint:lll // long SQL query
	res := se.db.QueryRow(`SELECT lhs.iql_generation_id FROM "__iql__.control.generation" lhs INNER JOIN (SELECT max(created_dttm) AS max_dttm FROM "__iql__.control.generation" WHERE collected_dttm IS null) rhs ON  lhs.created_dttm = rhs.max_dttm WHERE lhs.collected_dttm IS null`)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se snowflakeTCPEngine) getNextGenerationID() (int, error) {
	var retVal int
	//nolint:lll,execinquery // long SQL query and `execinquery` is DEAD SET RUBBISH for INSERT... RETURNING
	res := se.db.QueryRow(`INSERT INTO "__iql__.control.generation" (generation_description, created_dttm) VALUES ('', current_timestamp) RETURNING iql_generation_id`)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se snowflakeTCPEngine) getCurrentProviderGenerationID(providerName string) (int, error) {
	var retVal int
	//nolint:lll // long SQL query
	res := se.db.QueryRow(`SELECT lhs.iql_discovery_generation_id FROM "__iql__.control.discovery_generation" lhs INNER JOIN (SELECT discovery_name, max(created_dttm) AS max_dttm FROM "__iql__.control.discovery_generation" WHERE collected_dttm IS null GROUP BY discovery_name) rhs ON  lhs.created_dttm = rhs.max_dttm AND lhs.discovery_name = rhs.discovery_name WHERE lhs.collected_dttm IS null AND lhs.discovery_name = $1`, providerName)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se snowflakeTCPEngine) getNextProviderGenerationID(providerName string) (int, error) {
	var retVal int
	//nolint:lll,execinquery // long SQL query and `execinquery` is DEAD SET RUBBISH for INSERT... RETURNING
	res := se.db.QueryRow(`INSERT INTO "__iql__.control.discovery_generation" (discovery_name, created_dttm) VALUES ($1, current_timestamp) RETURNING iql_discovery_generation_id`, providerName)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se snowflakeTCPEngine) getCurrentSessionID(generationID int) (int, error) {
	var retVal int
	//nolint:lll // long SQL query
	res := se.db.QueryRow(`SELECT lhs.iql_session_id FROM "__iql__.control.session" lhs INNER JOIN (SELECT max(created_dttm) AS max_dttm FROM "__iql__.control.session" WHERE collected_dttm IS null) rhs ON  lhs.created_dttm = rhs.max_dttm AND lhs.iql_genration_id = rhs.iql_generation_id WHERE lhs.iql_generation_id = $1 AND lhs.collected_dttm IS null`, generationID)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se snowflakeTCPEngine) getNextSessionID(generationID int) (int, error) {
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

func (se snowflakeTCPEngine) CacheStoreGet(key string) ([]byte, error) {
	var retVal []byte
	res := se.db.QueryRow(`SELECT v FROM "__iql__.cache.key_val" WHERE k = $1`, key)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se snowflakeTCPEngine) CacheStoreGetAll() ([]internaldto.KeyVal, error) {
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

func (se snowflakeTCPEngine) CacheStorePut(key string, val []byte, tablespace string, tablespaceID int) error {
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

func (se snowflakeTCPEngine) Query(query string, varArgs ...interface{}) (*sql.Rows, error) {
	return se.query(query, varArgs...)
}

func (se snowflakeTCPEngine) query(query string, varArgs ...interface{}) (*sql.Rows, error) {
	res, err := se.db.Query(query, varArgs...)
	return res, err
}
