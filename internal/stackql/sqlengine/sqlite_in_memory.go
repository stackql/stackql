package sqlengine

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/sqlcontrol"
	"github.com/stackql/stackql/internal/stackql/util"

	_ "github.com/stackql/go-sqlite3"
)

var (
	_ SQLEngine = &sqLiteInProcessEngine{}
)

type sqLiteInProcessEngine struct {
	db                *sql.DB
	dsn               string
	controlAttributes sqlcontrol.ControlAttributes
	ctrlMutex         *sync.Mutex
	sessionMutex      *sync.Mutex
	discoveryMutex    *sync.Mutex
}

func (se *sqLiteInProcessEngine) IsMemory() bool {
	return strings.Contains(se.dsn, ":memory:") || strings.Contains(se.dsn, "mode=memory")
}

func (se *sqLiteInProcessEngine) GetDB() (*sql.DB, error) {
	return se.db, nil
}

func newSQLiteInProcessEngine(cfg dto.SQLBackendCfg, controlAttributes sqlcontrol.ControlAttributes) (*sqLiteInProcessEngine, error) {
	dsn := cfg.DSN
	if dsn == "" {
		dsn = "file::memory:?cache=shared"
	}
	db, err := sql.Open("sqlite3", dsn)
	db.SetConnMaxLifetime(-1)
	eng := &sqLiteInProcessEngine{
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
	if err != nil {
		return eng, err
	}
	return eng, err
}

func (eng *sqLiteInProcessEngine) execFileSQLite(fileName string) error {
	fileContents, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	_, err = eng.db.Exec(string(fileContents))
	return err
}

// In SQLite, `DateTime` objects are not properly aware; the zone is not recorded.
// That being said, those fields populated with `DateTime('now')` are UTC.
// As per https://www.sqlite.org/lang_datefunc.html:
//
//	The 'now' argument to date and time functions always returns exactly
//	the same value for multiple invocations within the same sqlite3_step()
//	call. Universal Coordinated Time (UTC) is used.
//
// Therefore, this method will behave correctly provided that the column `colName`
// is populated with `DateTime('now')`.
func (eng *sqLiteInProcessEngine) TableOldestUpdateUTC(tableName string, requestEncoding string, updateColName string, requestEncodingColName string) (time.Time, *dto.TxnControlCounters) {
	genIdColName := eng.controlAttributes.GetControlGenIdColumnName()
	ssnIdColName := eng.controlAttributes.GetControlSsnIdColumnName()
	txnIdColName := eng.controlAttributes.GetControlTxnIdColumnName()
	insIdColName := eng.controlAttributes.GetControlInsIdColumnName()
	rows, err := eng.db.Query(fmt.Sprintf("SELECT strftime('%%Y-%%m-%%dT%%H:%%M:%%S', min(%s)) as oldest_update, %s, %s, %s, %s FROM \"%s\" WHERE %s = '%s';", updateColName, genIdColName, ssnIdColName, txnIdColName, insIdColName, tableName, requestEncodingColName, requestEncoding))
	if err == nil && rows != nil {
		defer rows.Close()
		rowExists := rows.Next()
		if rowExists {
			var oldest string
			tcc := dto.TxnControlCounters{}
			err = rows.Scan(&oldest, &tcc.GenId, &tcc.SessionId, &tcc.TxnId, &tcc.InsertId)
			if err == nil {
				oldestTime, err := time.Parse("2006-01-02T15:04:05", oldest)
				if err == nil {
					tcc.TableName = tableName
					return oldestTime, &tcc
				}
			}
		}
	}
	return time.Time{}, nil
}

func (eng *sqLiteInProcessEngine) IsTablePresent(tableName string, requestEncoding string, colName string) bool {
	rows, err := eng.db.Query(fmt.Sprintf(`SELECT count(*) as ct FROM "%s" WHERE iql_insert_encoded=?;`, tableName), requestEncoding)
	if err == nil && rows != nil {
		defer rows.Close()
		rowExists := rows.Next()
		if rowExists {
			var ct int
			rows.Scan(&ct)
			if ct > 0 {
				return true
			}
		}
	}
	return false
}

func (eng *sqLiteInProcessEngine) execFileLocal(fileName string) error {
	expF, err := util.GetFilePathFromRepositoryRoot(fileName)
	if err != nil {
		return err
	}
	return eng.execFileSQLite(expF)
}

func (eng *sqLiteInProcessEngine) ExecFileLocal(fileName string) error {
	return eng.execFileLocal(fileName)
}

func (eng *sqLiteInProcessEngine) ExecFile(fileName string) error {
	return eng.execFileSQLite(fileName)
}

func (se sqLiteInProcessEngine) Exec(query string, varArgs ...interface{}) (sql.Result, error) {
	// logging.GetLogger().Infoln(fmt.Sprintf("exec query = %s", query))
	res, err := se.db.Exec(query, varArgs...)
	// logging.GetLogger().Infoln(fmt.Sprintf("res= %v, err = %v", res, err))
	return res, err
}

func (se sqLiteInProcessEngine) ExecInTxn(queries []string) error {
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

func (se sqLiteInProcessEngine) GetNextGenerationId() (int, error) {
	se.ctrlMutex.Lock()
	defer se.ctrlMutex.Unlock()
	return se.getNextGenerationId()
}

func (se sqLiteInProcessEngine) GetCurrentGenerationId() (int, error) {
	se.ctrlMutex.Lock()
	defer se.ctrlMutex.Unlock()
	return se.getCurrentGenerationId()
}

func (se sqLiteInProcessEngine) GetNextDiscoveryGenerationId(discoveryName string) (int, error) {
	se.discoveryMutex.Lock()
	defer se.discoveryMutex.Unlock()
	return se.getNextProviderGenerationId(discoveryName)
}

func (se sqLiteInProcessEngine) GetCurrentDiscoveryGenerationId(discoveryName string) (int, error) {
	se.discoveryMutex.Lock()
	defer se.discoveryMutex.Unlock()
	return se.getCurrentProviderGenerationId(discoveryName)
}

func (se sqLiteInProcessEngine) GetNextSessionId(generationId int) (int, error) {
	se.sessionMutex.Lock()
	defer se.sessionMutex.Unlock()
	return se.getNextSessionId(generationId)
}

func (se sqLiteInProcessEngine) GetCurrentSessionId(generationId int) (int, error) {
	se.sessionMutex.Lock()
	defer se.sessionMutex.Unlock()
	return se.getCurrentSessionId(generationId)
}

func (se sqLiteInProcessEngine) getCurrentGenerationId() (int, error) {
	var retVal int
	res := se.db.QueryRow(`SELECT lhs.iql_generation_id FROM "__iql__.control.generation" lhs INNER JOIN (SELECT max(created_dttm) AS max_dttm FROM "__iql__.control.generation" WHERE collected_dttm IS null) rhs ON  lhs.created_dttm = rhs.max_dttm WHERE lhs.collected_dttm IS null`)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se sqLiteInProcessEngine) GetCurrentTable(tableHeirarchyIDs *dto.HeirarchyIdentifiers) (dto.DBTable, error) {
	return se.getCurrentTable(tableHeirarchyIDs)
}

func (se sqLiteInProcessEngine) getCurrentTable(tableHeirarchyIDs *dto.HeirarchyIdentifiers) (dto.DBTable, error) {
	var tableName string
	var discoID int
	tableNamePattern := fmt.Sprintf("%s.generation_%%", tableHeirarchyIDs.GetTableName())
	tableNameLHSRemove := fmt.Sprintf("%s.generation_", tableHeirarchyIDs.GetTableName())
	res := se.db.QueryRow(`select name, CAST(REPLACE(name, ?, '') AS INTEGER) from sqlite_schema where type = 'table' and name like ? ORDER BY name DESC limit 1`, tableNameLHSRemove, tableNamePattern)
	err := res.Scan(&tableName, &discoID)
	if err != nil {
		logging.GetLogger().Errorln(fmt.Sprintf("err = %v for tableNamePattern = '%s' and tableNameLHSRemove = '%s'", err, tableNamePattern, tableNameLHSRemove))
	}
	return dto.NewDBTable(tableName, discoID, tableHeirarchyIDs), err
}

func (se sqLiteInProcessEngine) getNextGenerationId() (int, error) {
	var retVal int
	res := se.db.QueryRow(`INSERT INTO "__iql__.control.generation" (generation_description, created_dttm) VALUES ('', strftime('%s', 'now')) RETURNING iql_generation_id`)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se sqLiteInProcessEngine) getCurrentProviderGenerationId(providerName string) (int, error) {
	var retVal int
	res := se.db.QueryRow(`SELECT lhs.iql_discovery_generation_id FROM "__iql__.control.discovery_generation" lhs INNER JOIN (SELECT discovery_name, max(created_dttm) AS max_dttm FROM "__iql__.control.discovery_generation" WHERE collected_dttm IS null GROUP BY discovery_name) rhs ON  lhs.created_dttm = rhs.max_dttm AND lhs.discovery_name = rhs.discovery_name WHERE lhs.collected_dttm IS null AND lhs.discovery_name = ?`, providerName)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se sqLiteInProcessEngine) getNextProviderGenerationId(providerName string) (int, error) {
	var retVal int
	res := se.db.QueryRow(`INSERT INTO "__iql__.control.discovery_generation" (discovery_name, created_dttm) VALUES (?, strftime('%s', 'now')) RETURNING iql_discovery_generation_id`, providerName)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se sqLiteInProcessEngine) getCurrentSessionId(generationId int) (int, error) {
	var retVal int
	res := se.db.QueryRow(`SELECT lhs.iql_session_id FROM "__iql__.control.session" lhs INNER JOIN (SELECT max(created_dttm) AS max_dttm FROM "__iql__.control.session" WHERE collected_dttm IS null) rhs ON  lhs.created_dttm = rhs.max_dttm AND lhs.iql_genration_id = rhs.iql_generation_id WHERE lhs.iql_generation_id = ? AND lhs.collected_dttm IS null`, generationId)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se sqLiteInProcessEngine) getNextSessionId(generationId int) (int, error) {
	var retVal int
	res := se.db.QueryRow(`INSERT INTO "__iql__.control.session" (iql_generation_id, created_dttm) VALUES (?, strftime('%s', 'now')) RETURNING iql_session_id`, generationId)
	err := res.Scan(&retVal)
	logging.GetLogger().Infoln(fmt.Sprintf("getNextSessionId(): generation id = %d, session id = %d", generationId, retVal))
	return retVal, err
}

func (se sqLiteInProcessEngine) CacheStoreGet(key string) ([]byte, error) {
	var retVal []byte
	res := se.db.QueryRow(`SELECT v FROM "__iql__.cache.key_val" WHERE k = ?`, key)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se sqLiteInProcessEngine) CacheStoreGetAll() ([]dto.KeyVal, error) {
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

func (se sqLiteInProcessEngine) CacheStorePut(key string, val []byte, tablespace string, tablespaceID int) error {
	txn, err := se.db.Begin()
	if err != nil {
		return err
	}
	_, err = txn.Exec(`DELETE FROM "__iql__.cache.key_val" WHERE k = ?`, key)
	if err != nil {
		txn.Rollback()
		return err
	}
	_, err = txn.Exec(`INSERT INTO "__iql__.cache.key_val" (k, v, tablespace, tablespace_id) VALUES(?, ?, ?, ?)`, key, val, tablespace, tablespaceID)
	if err != nil {
		txn.Rollback()
		return err
	}
	err = txn.Commit()
	return err
}

func (se sqLiteInProcessEngine) Query(query string, varArgs ...interface{}) (*sql.Rows, error) {
	return se.query(query, varArgs...)
}

func (se sqLiteInProcessEngine) query(query string, varArgs ...interface{}) (*sql.Rows, error) {
	// logging.GetLogger().Infoln(fmt.Sprintf("query = %s", query))
	res, err := se.db.Query(query, varArgs...)
	// logging.GetLogger().Infoln(fmt.Sprintf("res= %v, err = %v", res, err))
	return res, err
}
