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
	"github.com/stackql/stackql/internal/stackql/util"

	_ "github.com/stackql/go-sqlite3"
)

var (
	_ SQLEngine = &sqLiteEngine{}
)

type sqLiteEngine struct {
	db             *sql.DB
	fileName       string
	ctrlMutex      *sync.Mutex
	sessionMutex   *sync.Mutex
	discoveryMutex *sync.Mutex
}

func (se *sqLiteEngine) IsMemory() bool {
	return strings.Contains(se.fileName, ":memory:") || strings.Contains(se.fileName, "mode=memory")
}

func (se *sqLiteEngine) GetDB() (*sql.DB, error) {
	return se.db, nil
}

func newSQLiteEngine(cfg SQLEngineConfig) (*sqLiteEngine, error) {
	fileName := cfg.fileName
	if fileName == "" {
		fileName = "file::memory:?cache=shared"
	}
	db, err := sql.Open("sqlite3", fileName)
	db.SetConnMaxLifetime(-1)
	eng := &sqLiteEngine{
		db:             db,
		fileName:       fileName,
		ctrlMutex:      &sync.Mutex{},
		sessionMutex:   &sync.Mutex{},
		discoveryMutex: &sync.Mutex{},
	}
	if err != nil {
		return eng, err
	}
	if cfg.initFileName != "" {
		err = eng.execFileSQLite(cfg.initFileName)
	}
	if err != nil {
		return eng, err
	}
	logging.GetLogger().Infoln(fmt.Sprintf("opened db with file = '%s' and err  = '%v'", fileName, err))
	if err != nil {
		return eng, err
	}
	err = eng.initSQLiteEngine()
	return eng, err
}

// In SQLite, `DateTime` objects are not properly aware; the zone is not recorded.
// That being said, those fields populated with `DateTime('now')` are UTC.
// As per https://www.sqlite.org/lang_datefunc.html:
//    The 'now' argument to date and time functions always returns exactly
//    the same value for multiple invocations within the same sqlite3_step()
//    call. Universal Coordinated Time (UTC) is used.
// Therefore, this method will behave correctly provided that the column `colName`
// is populated with `DateTime('now')`.
func (eng *sqLiteEngine) TableOldestUpdateUTC(tableName string, requestEncoding string, updateColName string, requestEncodingColName string) (time.Time, *dto.TxnControlCounters) {
	var gen_id_col_name string = "iql_generation_id"
	var ssn_id_col_name string = "iql_session_id"
	var txn_id_col_name string = "iql_txn_id"
	var ins_id_col_name string = "iql_insert_id"
	rows, err := eng.db.Query(fmt.Sprintf("SELECT strftime('%%Y-%%m-%%dT%%H:%%M:%%S', min(%s)) as oldest_update, %s, %s, %s, %s FROM \"%s\" WHERE %s = '%s';", updateColName, gen_id_col_name, ssn_id_col_name, txn_id_col_name, ins_id_col_name, tableName, requestEncodingColName, requestEncoding))
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

func (eng *sqLiteEngine) execFileSQLite(fileName string) error {
	fileContents, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	_, err = eng.db.Exec(string(fileContents))
	return err
}

func (eng *sqLiteEngine) IsTablePresent(tableName string, requestEncoding string, colName string) bool {
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

func (eng *sqLiteEngine) execFileLocal(fileName string) error {
	expF, err := util.GetFilePathFromRepositoryRoot(fileName)
	if err != nil {
		return err
	}
	return eng.execFileSQLite(expF)
}

func (eng *sqLiteEngine) ExecFileLocal(fileName string) error {
	return eng.execFileLocal(fileName)
}

func (eng *sqLiteEngine) ExecFile(fileName string) error {
	return eng.execFileSQLite(fileName)
}

func (eng *sqLiteEngine) initSQLiteEngine() error {
	_, err := eng.db.Exec(sqlEngineSetupDDL)
	return err
}

func (se sqLiteEngine) Exec(query string, varArgs ...interface{}) (sql.Result, error) {
	// logging.GetLogger().Infoln(fmt.Sprintf("exec query = %s", query))
	res, err := se.db.Exec(query, varArgs...)
	// logging.GetLogger().Infoln(fmt.Sprintf("res= %v, err = %v", res, err))
	return res, err
}

func (se sqLiteEngine) ExecInTxn(queries []string) error {
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

func (se sqLiteEngine) GetNextGenerationId() (int, error) {
	se.ctrlMutex.Lock()
	defer se.ctrlMutex.Unlock()
	return se.getNextGenerationId()
}

func (se sqLiteEngine) GetCurrentGenerationId() (int, error) {
	se.ctrlMutex.Lock()
	defer se.ctrlMutex.Unlock()
	return se.getCurrentGenerationId()
}

func (se sqLiteEngine) GetNextDiscoveryGenerationId(discoveryName string) (int, error) {
	se.discoveryMutex.Lock()
	defer se.discoveryMutex.Unlock()
	return se.getNextProviderGenerationId(discoveryName)
}

func (se sqLiteEngine) GetCurrentDiscoveryGenerationId(discoveryName string) (int, error) {
	se.discoveryMutex.Lock()
	defer se.discoveryMutex.Unlock()
	return se.getCurrentProviderGenerationId(discoveryName)
}

func (se sqLiteEngine) GetNextSessionId(generationId int) (int, error) {
	se.sessionMutex.Lock()
	defer se.sessionMutex.Unlock()
	return se.getNextSessionId(generationId)
}

func (se sqLiteEngine) GetCurrentSessionId(generationId int) (int, error) {
	se.sessionMutex.Lock()
	defer se.sessionMutex.Unlock()
	return se.getCurrentSessionId(generationId)
}

func (se sqLiteEngine) getCurrentGenerationId() (int, error) {
	var retVal int
	res := se.db.QueryRow(`SELECT lhs.iql_generation_id FROM "__iql__.control.generation" lhs INNER JOIN (SELECT max(created_dttm) AS max_dttm FROM "__iql__.control.generation" WHERE collected_dttm IS null) rhs ON  lhs.created_dttm = rhs.max_dttm WHERE lhs.collected_dttm IS null`)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se sqLiteEngine) GetCurrentTable(tableHeirarchyIDs *dto.HeirarchyIdentifiers) (dto.DBTable, error) {
	return se.getCurrentTable(tableHeirarchyIDs)
}

func (se sqLiteEngine) getCurrentTable(tableHeirarchyIDs *dto.HeirarchyIdentifiers) (dto.DBTable, error) {
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

func (se sqLiteEngine) getNextGenerationId() (int, error) {
	var retVal int
	res := se.db.QueryRow(`INSERT INTO "__iql__.control.generation" (generation_description, created_dttm) VALUES ('', strftime('%s', 'now')) RETURNING iql_generation_id`)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se sqLiteEngine) getCurrentProviderGenerationId(providerName string) (int, error) {
	var retVal int
	res := se.db.QueryRow(`SELECT lhs.iql_discovery_generation_id FROM "__iql__.control.discovery_generation" lhs INNER JOIN (SELECT discovery_name, max(created_dttm) AS max_dttm FROM "__iql__.control.discovery_generation" WHERE collected_dttm IS null GROUP BY discovery_name) rhs ON  lhs.created_dttm = rhs.max_dttm AND lhs.discovery_name = rhs.discovery_name WHERE lhs.collected_dttm IS null AND lhs.discovery_name = ?`, providerName)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se sqLiteEngine) getNextProviderGenerationId(providerName string) (int, error) {
	var retVal int
	res := se.db.QueryRow(`INSERT INTO "__iql__.control.discovery_generation" (discovery_name, created_dttm) VALUES (?, strftime('%s', 'now')) RETURNING iql_discovery_generation_id`, providerName)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se sqLiteEngine) getCurrentSessionId(generationId int) (int, error) {
	var retVal int
	res := se.db.QueryRow(`SELECT lhs.iql_session_id FROM "__iql__.control.session" lhs INNER JOIN (SELECT max(created_dttm) AS max_dttm FROM "__iql__.control.session" WHERE collected_dttm IS null) rhs ON  lhs.created_dttm = rhs.max_dttm AND lhs.iql_genration_id = rhs.iql_generation_id WHERE lhs.iql_generation_id = ? AND lhs.collected_dttm IS null`, generationId)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se sqLiteEngine) getNextSessionId(generationId int) (int, error) {
	var retVal int
	res := se.db.QueryRow(`INSERT INTO "__iql__.control.session" (iql_generation_id, created_dttm) VALUES (?, strftime('%s', 'now')) RETURNING iql_session_id`, generationId)
	err := res.Scan(&retVal)
	logging.GetLogger().Infoln(fmt.Sprintf("getNextSessionId(): generation id = %d, session id = %d", generationId, retVal))
	return retVal, err
}

func (se sqLiteEngine) CacheStoreGet(key string) ([]byte, error) {
	var retVal []byte
	res := se.db.QueryRow(`SELECT v FROM "__iql__.cache.key_val" WHERE k = ?`, key)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se sqLiteEngine) CacheStoreGetAll() ([]dto.KeyVal, error) {
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

func (se sqLiteEngine) CacheStorePut(key string, val []byte, tablespace string, tablespaceID int) error {
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

func (se sqLiteEngine) GCEnactFull() error {
	err := se.collectObsolete()
	if err != nil {
		return err
	}
	err = se.collectUnreachable()
	return err
}

func (se sqLiteEngine) GCCollectObsolete(tcc *dto.TxnControlCounters) error {
	return se.collectObsoleteQualified(tcc)
}

func (se sqLiteEngine) GCCollectUnreachable() error {
	return se.collectUnreachable()
}

func (se sqLiteEngine) collectUnreachable() error {
	return se.concertedQueryGen(unreachableTablesQuery)
}

func (se sqLiteEngine) collectObsolete() error {
	return se.concertedQueryGen(cleanupObsoleteQuery)
}

func (se sqLiteEngine) collectObsoleteQualified(tcc *dto.TxnControlCounters) error {
	return se.concertedQueryGen(cleanupObsoleteQualifiedQuery, tcc.GenId, tcc.SessionId, tcc.TxnId)
}

func (se sqLiteEngine) concertedQueryGen(generatorQuery string, args ...interface{}) error {
	if se.IsMemory() {
		return nil
	}
	rows, err := se.db.Query(generatorQuery, args...)
	if err != nil {
		logging.GetLogger().Infoln(fmt.Sprintf("obsolete compose error: %v", err))
		return err
	}
	txn, err := se.db.Begin()
	if err != nil {
		logging.GetLogger().Infoln(fmt.Sprintf("%v", err))
		return err
	}
	amalgam, err := singleColRowsToString(rows)
	if err != nil {
		logging.GetLogger().Infoln(fmt.Sprintf("obsolete obtain error: %v", err))
		txn.Rollback()
		return err
	}
	logging.GetLogger().Infoln(fmt.Sprintf("amalgam = %s", amalgam))
	_, err = se.db.Exec(amalgam, args...)
	if err != nil {
		logging.GetLogger().Infoln(fmt.Sprintf("obsolete exec error: %v", err))
		txn.Rollback()
		return err
	}
	err = txn.Commit()
	return err
}

func (se sqLiteEngine) Query(query string, varArgs ...interface{}) (*sql.Rows, error) {
	return se.query(query, varArgs...)
}

func (se sqLiteEngine) query(query string, varArgs ...interface{}) (*sql.Rows, error) {
	// logging.GetLogger().Infoln(fmt.Sprintf("query = %s", query))
	res, err := se.db.Query(query, varArgs...)
	// logging.GetLogger().Infoln(fmt.Sprintf("res= %v, err = %v", res, err))
	return res, err
}
