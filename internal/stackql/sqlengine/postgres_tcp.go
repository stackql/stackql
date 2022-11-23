package sqlengine

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/sqlcontrol"
	"github.com/stackql/stackql/internal/stackql/util"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	_ SQLEngine = &postgresTcpEngine{}
)

type postgresTcpEngine struct {
	db                *sql.DB
	dsn               string
	controlAttributes sqlcontrol.ControlAttributes
	ctrlMutex         *sync.Mutex
	sessionMutex      *sync.Mutex
	discoveryMutex    *sync.Mutex
}

func (se *postgresTcpEngine) IsMemory() bool {
	return false
}

func (se *postgresTcpEngine) GetDB() (*sql.DB, error) {
	return se.db, nil
}

func newPostgresTcpEngine(cfg dto.SQLBackendCfg, controlAttributes sqlcontrol.ControlAttributes) (*postgresTcpEngine, error) {
	dsn := cfg.DSN
	if dsn == "" {
		return nil, fmt.Errorf("cannot init postgres TCP connection with empty connection string")
	}
	db, err := sql.Open("pgx", dsn)
	retryCount := 0
	for {
		if retryCount >= cfg.InitMaxRetries || err == nil {
			break
		}
		time.Sleep(time.Duration(cfg.InitRetryInitialDelay) * time.Second)
		db, err = sql.Open("pgx", dsn)
		retryCount++
	}
	if err != nil {
		return nil, fmt.Errorf("postgres db object setup error = '%s'", err.Error())
	}
	logging.GetLogger().Debugln(fmt.Sprintf("opened postgres TCP db with connection string = '%s' and err  = '%v'", dsn, err))
	pingErr := db.Ping()
	retryCount = 0
	for {
		if retryCount >= cfg.InitMaxRetries || pingErr == nil {
			break
		}
		time.Sleep(time.Duration(cfg.InitRetryInitialDelay) * time.Second)
		pingErr = db.Ping()
		retryCount++
	}
	if pingErr != nil {
		return nil, fmt.Errorf("postgres connection setup ping error = '%s'", pingErr.Error())
	}
	logging.GetLogger().Debugln(fmt.Sprintf("opened and pinged postgres TCP db with connection string = '%s' and err  = '%v'", dsn, err))
	db.SetConnMaxLifetime(-1)
	eng := &postgresTcpEngine{
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
	// logging.GetLogger().Debugln(fmt.Sprintf("opened postgres TCP db with connection string = '%s'", dsn))
	if err != nil {
		return eng, err
	}
	return eng, err
}

func (eng *postgresTcpEngine) execFile(fileName string) error {
	fileContents, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	_, err = eng.db.Exec(string(fileContents))
	if err != nil {
		return fmt.Errorf("stackql postgres db exec file error: %s", err.Error())
	}
	return nil
}

func (eng *postgresTcpEngine) execFileLocal(fileName string) error {
	expF, err := util.GetFilePathFromRepositoryRoot(fileName)
	if err != nil {
		return err
	}
	return eng.execFile(expF)
}

func (eng *postgresTcpEngine) ExecFileLocal(fileName string) error {
	return eng.execFileLocal(fileName)
}

func (eng *postgresTcpEngine) ExecFile(fileName string) error {
	return eng.execFile(fileName)
}

func (se postgresTcpEngine) Exec(query string, varArgs ...interface{}) (sql.Result, error) {
	res, err := se.db.Exec(query, varArgs...)
	return res, err
}

func (se postgresTcpEngine) ExecInTxn(queries []string) error {
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

func (se postgresTcpEngine) GetNextGenerationId() (int, error) {
	se.ctrlMutex.Lock()
	defer se.ctrlMutex.Unlock()
	return se.getNextGenerationId()
}

func (se postgresTcpEngine) GetCurrentGenerationId() (int, error) {
	se.ctrlMutex.Lock()
	defer se.ctrlMutex.Unlock()
	return se.getCurrentGenerationId()
}

func (se postgresTcpEngine) GetNextDiscoveryGenerationId(discoveryName string) (int, error) {
	se.discoveryMutex.Lock()
	defer se.discoveryMutex.Unlock()
	return se.getNextProviderGenerationId(discoveryName)
}

func (se postgresTcpEngine) GetCurrentDiscoveryGenerationId(discoveryName string) (int, error) {
	se.discoveryMutex.Lock()
	defer se.discoveryMutex.Unlock()
	return se.getCurrentProviderGenerationId(discoveryName)
}

func (se postgresTcpEngine) GetNextSessionId(generationId int) (int, error) {
	se.sessionMutex.Lock()
	defer se.sessionMutex.Unlock()
	return se.getNextSessionId(generationId)
}

func (se postgresTcpEngine) GetCurrentSessionId(generationId int) (int, error) {
	se.sessionMutex.Lock()
	defer se.sessionMutex.Unlock()
	return se.getCurrentSessionId(generationId)
}

func (se postgresTcpEngine) getCurrentGenerationId() (int, error) {
	var retVal int
	res := se.db.QueryRow(`SELECT lhs.iql_generation_id FROM "__iql__.control.generation" lhs INNER JOIN (SELECT max(created_dttm) AS max_dttm FROM "__iql__.control.generation" WHERE collected_dttm IS null) rhs ON  lhs.created_dttm = rhs.max_dttm WHERE lhs.collected_dttm IS null`)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se postgresTcpEngine) GetCurrentTable(tableHeirarchyIDs *dto.HeirarchyIdentifiers) (dto.DBTable, error) {
	return se.getCurrentTable(tableHeirarchyIDs)
}

func (se postgresTcpEngine) getCurrentTable(tableHeirarchyIDs *dto.HeirarchyIdentifiers) (dto.DBTable, error) {
	var tableName string
	var discoID int
	tableNamePattern := fmt.Sprintf("%s.generation_%%", tableHeirarchyIDs.GetTableName())
	tableNameLHSRemove := fmt.Sprintf("%s.generation_", tableHeirarchyIDs.GetTableName())
	res := se.db.QueryRow(`
	select 
		table_name, 
		CAST(REPLACE(table_name, $1, '') AS INTEGER) 
	from 
		information_schema.tables 
	where 
		table_type = 'BASE TABLE'
	  and 
		table_name like $2 
	ORDER BY table_name DESC 
	limit 1
	`, tableNameLHSRemove, tableNamePattern)
	err := res.Scan(&tableName, &discoID)
	if err != nil {
		logging.GetLogger().Errorln(fmt.Sprintf("err = %v for tableNamePattern = '%s' and tableNameLHSRemove = '%s'", err, tableNamePattern, tableNameLHSRemove))
	}
	return dto.NewDBTable(tableName, tableHeirarchyIDs.GetTableName(), discoID, tableHeirarchyIDs), err
}

func (se postgresTcpEngine) getNextGenerationId() (int, error) {
	var retVal int
	res := se.db.QueryRow(`INSERT INTO "__iql__.control.generation" (generation_description, created_dttm) VALUES ('', current_timestamp) RETURNING iql_generation_id`)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se postgresTcpEngine) getCurrentProviderGenerationId(providerName string) (int, error) {
	var retVal int
	res := se.db.QueryRow(`SELECT lhs.iql_discovery_generation_id FROM "__iql__.control.discovery_generation" lhs INNER JOIN (SELECT discovery_name, max(created_dttm) AS max_dttm FROM "__iql__.control.discovery_generation" WHERE collected_dttm IS null GROUP BY discovery_name) rhs ON  lhs.created_dttm = rhs.max_dttm AND lhs.discovery_name = rhs.discovery_name WHERE lhs.collected_dttm IS null AND lhs.discovery_name = $1`, providerName)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se postgresTcpEngine) getNextProviderGenerationId(providerName string) (int, error) {
	var retVal int
	res := se.db.QueryRow(`INSERT INTO "__iql__.control.discovery_generation" (discovery_name, created_dttm) VALUES ($1, current_timestamp) RETURNING iql_discovery_generation_id`, providerName)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se postgresTcpEngine) getCurrentSessionId(generationId int) (int, error) {
	var retVal int
	res := se.db.QueryRow(`SELECT lhs.iql_session_id FROM "__iql__.control.session" lhs INNER JOIN (SELECT max(created_dttm) AS max_dttm FROM "__iql__.control.session" WHERE collected_dttm IS null) rhs ON  lhs.created_dttm = rhs.max_dttm AND lhs.iql_genration_id = rhs.iql_generation_id WHERE lhs.iql_generation_id = $1 AND lhs.collected_dttm IS null`, generationId)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se postgresTcpEngine) getNextSessionId(generationId int) (int, error) {
	var retVal int
	res := se.db.QueryRow(`INSERT INTO "__iql__.control.session" (iql_generation_id, created_dttm) VALUES ($1, current_timestamp) RETURNING iql_session_id`, generationId)
	err := res.Scan(&retVal)
	logging.GetLogger().Infoln(fmt.Sprintf("getNextSessionId(): generation id = %d, session id = %d", generationId, retVal))
	return retVal, err
}

func (se postgresTcpEngine) CacheStoreGet(key string) ([]byte, error) {
	var retVal []byte
	res := se.db.QueryRow(`SELECT v FROM "__iql__.cache.key_val" WHERE k = $1`, key)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se postgresTcpEngine) CacheStoreGetAll() ([]dto.KeyVal, error) {
	var retVal []dto.KeyVal
	res, err := se.db.Query(`SELECT k, v FROM "__iql__.cache.key_val"`)
	if err != nil {
		return nil, err
	}
	for res.Next() {
		var kv dto.KeyVal
		err = res.Scan(&kv.K, &kv.V)
		if err != nil {
			return nil, err
		}
		retVal = append(retVal, kv)
	}
	return retVal, err
}

func (se postgresTcpEngine) CacheStorePut(key string, val []byte, tablespace string, tablespaceID int) error {
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

func (se postgresTcpEngine) Query(query string, varArgs ...interface{}) (*sql.Rows, error) {
	return se.query(query, varArgs...)
}

func (se postgresTcpEngine) query(query string, varArgs ...interface{}) (*sql.Rows, error) {
	res, err := se.db.Query(query, varArgs...)
	return res, err
}
