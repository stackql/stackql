package sqlengine

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/util"

	log "github.com/sirupsen/logrus"
	_ "github.com/stackql/go-sqlite3"
)

type SQLEngineConfig struct {
	fileName     string
	initFileName string
	dbEngine     string
}

func NewSQLEngineConfig(runctimeCtx dto.RuntimeCtx) SQLEngineConfig {
	return SQLEngineConfig{
		fileName:     runctimeCtx.DbFilePath,
		initFileName: runctimeCtx.DbInitFilePath,
		dbEngine:     runctimeCtx.DbEngine,
	}
}

type SQLEngine interface {
	GetDB() (*sql.DB, error)
	Exec(string, ...interface{}) (sql.Result, error)
	Query(string, ...interface{}) (*sql.Rows, error)
	ExecFileLocal(string) error
	ExecFile(string) error
	GCCollectObsolete(*dto.TxnControlCounters) error
	GCCollectObsoleteAll() error
	GCCollectUnreachable() error
	GCEnactFull() error
	GetCurrentGenerationId() (int, error)
	GetNextGenerationId() (int, error)
	GetCurrentSessionId(int) (int, error)
	GetNextSessionId(int) (int, error)
	GetCurrentTable(*dto.HeirarchyIdentifiers) (dto.DBTable, error)
	GetCurrentDiscoveryGenerationId(discoveryID string) (int, error)
	GetNextDiscoveryGenerationId(discoveryID string) (int, error)
	CacheStoreGet(string) ([]byte, error)
	CacheStoreGetAll() ([]dto.KeyVal, error)
	CacheStorePut(string, []byte, string, int) error
}

type SQLiteEngine struct {
	db             *sql.DB
	fileName       string
	ctrlMutex      *sync.Mutex
	sessionMutex   *sync.Mutex
	discoveryMutex *sync.Mutex
}

func (se *SQLiteEngine) IsMemory() bool {
	return strings.Contains(se.fileName, ":memory:") || strings.Contains(se.fileName, "mode=memory")
}

func (se *SQLiteEngine) GetDB() (*sql.DB, error) {
	return se.db, nil
}

func NewSQLEngine(cfg SQLEngineConfig) (SQLEngine, error) {
	return newSQLiteEngine(cfg)
}

func newSQLiteEngine(cfg SQLEngineConfig) (*SQLiteEngine, error) {
	fileName := cfg.fileName
	if fileName == "" {
		fileName = "file::memory:?cache=shared"
	}
	db, err := sql.Open("sqlite3", fileName)
	db.SetConnMaxLifetime(-1)
	eng := &SQLiteEngine{
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
	log.Infoln(fmt.Sprintf("opened db with file = '%s' and err  = '%v'", fileName, err))
	if err != nil {
		return eng, err
	}
	err = eng.initSQLiteEngine()
	return eng, err
}

func (eng *SQLiteEngine) execFileSQLite(fileName string) error {
	fileContents, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	_, err = eng.db.Exec(string(fileContents))
	return err
}

func (eng *SQLiteEngine) execFileLocal(fileName string) error {
	expF, err := util.GetFilePathFromRepositoryRoot(fileName)
	if err != nil {
		return err
	}
	return eng.execFileSQLite(expF)
}

func (eng *SQLiteEngine) ExecFileLocal(fileName string) error {
	return eng.execFileLocal(fileName)
}

func (eng *SQLiteEngine) ExecFile(fileName string) error {
	return eng.execFileSQLite(fileName)
}

func (eng *SQLiteEngine) initSQLiteEngine() error {
	_, err := eng.db.Exec(sqlEngineSetupDDL)
	return err
}

func (se SQLiteEngine) Exec(query string, varArgs ...interface{}) (sql.Result, error) {
	// log.Infoln(fmt.Sprintf("exec query = %s", query))
	res, err := se.db.Exec(query, varArgs...)
	// log.Infoln(fmt.Sprintf("res= %v, err = %v", res, err))
	return res, err
}

func (se SQLiteEngine) GetNextGenerationId() (int, error) {
	se.ctrlMutex.Lock()
	defer se.ctrlMutex.Unlock()
	return se.getNextGenerationId()
}

func (se SQLiteEngine) GetCurrentGenerationId() (int, error) {
	se.ctrlMutex.Lock()
	defer se.ctrlMutex.Unlock()
	return se.getCurrentGenerationId()
}

func (se SQLiteEngine) GetNextDiscoveryGenerationId(discoveryName string) (int, error) {
	se.discoveryMutex.Lock()
	defer se.discoveryMutex.Unlock()
	return se.getNextProviderGenerationId(discoveryName)
}

func (se SQLiteEngine) GetCurrentDiscoveryGenerationId(discoveryName string) (int, error) {
	se.discoveryMutex.Lock()
	defer se.discoveryMutex.Unlock()
	return se.getCurrentProviderGenerationId(discoveryName)
}

func (se SQLiteEngine) GetNextSessionId(generationId int) (int, error) {
	se.sessionMutex.Lock()
	defer se.sessionMutex.Unlock()
	return se.getNextSessionId(generationId)
}

func (se SQLiteEngine) GetCurrentSessionId(generationId int) (int, error) {
	se.sessionMutex.Lock()
	defer se.sessionMutex.Unlock()
	return se.getCurrentSessionId(generationId)
}

func (se SQLiteEngine) getCurrentGenerationId() (int, error) {
	var retVal int
	res := se.db.QueryRow(`SELECT lhs.iql_generation_id FROM "__iql__.control.generation" lhs INNER JOIN (SELECT max(created_dttm) AS max_dttm FROM "__iql__.control.generation" WHERE collected_dttm IS null) rhs ON  lhs.created_dttm = rhs.max_dttm WHERE lhs.collected_dttm IS null`)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se SQLiteEngine) GetCurrentTable(tableHeirarchyIDs *dto.HeirarchyIdentifiers) (dto.DBTable, error) {
	return se.getCurrentTable(tableHeirarchyIDs)
}

func (se SQLiteEngine) getCurrentTable(tableHeirarchyIDs *dto.HeirarchyIdentifiers) (dto.DBTable, error) {
	var tableName string
	var discoID int
	tableNamePattern := fmt.Sprintf("%s.generation_%%", tableHeirarchyIDs.GetTableName())
	tableNameLHSRemove := fmt.Sprintf("%s.generation_", tableHeirarchyIDs.GetTableName())
	res := se.db.QueryRow(`select name, CAST(REPLACE(name, ?, '') AS INTEGER) from sqlite_schema where type = 'table' and name like ? ORDER BY name DESC limit 1`, tableNameLHSRemove, tableNamePattern)
	err := res.Scan(&tableName, &discoID)
	if err != nil {
		log.Errorln(fmt.Sprintf("err = %v for tableNamePattern = '%s' and tableNameLHSRemove = '%s'", err, tableNamePattern, tableNameLHSRemove))
	}
	return dto.NewDBTable(tableName, discoID, tableHeirarchyIDs), err
}

func (se SQLiteEngine) getNextGenerationId() (int, error) {
	var retVal int
	res := se.db.QueryRow(`INSERT INTO "__iql__.control.generation" (generation_description, created_dttm) VALUES ('', strftime('%s', 'now')) RETURNING iql_generation_id`)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se SQLiteEngine) getCurrentProviderGenerationId(providerName string) (int, error) {
	var retVal int
	res := se.db.QueryRow(`SELECT lhs.iql_discovery_generation_id FROM "__iql__.control.discovery_generation" lhs INNER JOIN (SELECT discovery_name, max(created_dttm) AS max_dttm FROM "__iql__.control.discovery_generation" WHERE collected_dttm IS null GROUP BY discovery_name) rhs ON  lhs.created_dttm = rhs.max_dttm AND lhs.discovery_name = rhs.discovery_name WHERE lhs.collected_dttm IS null AND lhs.discovery_name = ?`, providerName)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se SQLiteEngine) getNextProviderGenerationId(providerName string) (int, error) {
	var retVal int
	res := se.db.QueryRow(`INSERT INTO "__iql__.control.discovery_generation" (discovery_name, created_dttm) VALUES (?, strftime('%s', 'now')) RETURNING iql_discovery_generation_id`, providerName)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se SQLiteEngine) getCurrentSessionId(generationId int) (int, error) {
	var retVal int
	res := se.db.QueryRow(`SELECT lhs.iql_session_id FROM "__iql__.control.session" lhs INNER JOIN (SELECT max(created_dttm) AS max_dttm FROM "__iql__.control.session" WHERE collected_dttm IS null) rhs ON  lhs.created_dttm = rhs.max_dttm AND lhs.iql_genration_id = rhs.iql_generation_id WHERE lhs.iql_generation_id = ? AND lhs.collected_dttm IS null`, generationId)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se SQLiteEngine) getNextSessionId(generationId int) (int, error) {
	var retVal int
	res := se.db.QueryRow(`INSERT INTO "__iql__.control.session" (iql_generation_id, created_dttm) VALUES (?, strftime('%s', 'now')) RETURNING iql_session_id`, generationId)
	err := res.Scan(&retVal)
	log.Infoln(fmt.Sprintf("getNextSessionId(): generation id = %d, session id = %d", generationId, retVal))
	return retVal, err
}

func (se SQLiteEngine) CacheStoreGet(key string) ([]byte, error) {
	var retVal []byte
	res := se.db.QueryRow(`SELECT v FROM "__iql__.cache.key_val" WHERE k = ?`, key)
	err := res.Scan(&retVal)
	return retVal, err
}

func (se SQLiteEngine) CacheStoreGetAll() ([]dto.KeyVal, error) {
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

func (se SQLiteEngine) CacheStorePut(key string, val []byte, tablespace string, tablespaceID int) error {
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

func (se SQLiteEngine) GCEnactFull() error {
	err := se.collectObsolete()
	if err != nil {
		return err
	}
	err = se.collectUnreachable()
	return err
}

func (se SQLiteEngine) GCCollectObsoleteAll() error {
	return se.collectObsolete()
}

func (se SQLiteEngine) GCCollectObsolete(tcc *dto.TxnControlCounters) error {
	return se.collectObsoleteQualified(tcc)
}

func (se SQLiteEngine) GCCollectUnreachable() error {
	return se.collectUnreachable()
}

func (se SQLiteEngine) collectUnreachable() error {
	return se.concertedQueryGen(unreachableTablesQuery)
}

func (se SQLiteEngine) collectObsolete() error {
	return se.concertedQueryGen(cleanupObsoleteQuery)
}

func (se SQLiteEngine) collectObsoleteQualified(tcc *dto.TxnControlCounters) error {
	return se.concertedQueryGen(cleanupObsoleteQualifiedQuery, tcc.GenId, tcc.SessionId, tcc.TxnId)
}

func (se SQLiteEngine) concertedQueryGen(generatorQuery string, args ...interface{}) error {
	if se.IsMemory() {
		return nil
	}
	rows, err := se.db.Query(generatorQuery, args...)
	if err != nil {
		log.Infoln(fmt.Sprintf("obsolete compose error: %v", err))
		return err
	}
	txn, err := se.db.Begin()
	if err != nil {
		log.Infoln(fmt.Sprintf("%v", err))
		return err
	}
	amalgam, err := singleColRowsToString(rows)
	if err != nil {
		log.Infoln(fmt.Sprintf("obsolete obtain error: %v", err))
		txn.Rollback()
		return err
	}
	log.Infoln(fmt.Sprintf("amalgam = %s", amalgam))
	_, err = se.db.Exec(amalgam, args...)
	if err != nil {
		log.Infoln(fmt.Sprintf("obsolete exec error: %v", err))
		txn.Rollback()
		return err
	}
	err = txn.Commit()
	return err
}

func (se SQLiteEngine) Query(query string, varArgs ...interface{}) (*sql.Rows, error) {
	return se.query(query, varArgs...)
}

func (se SQLiteEngine) query(query string, varArgs ...interface{}) (*sql.Rows, error) {
	// log.Infoln(fmt.Sprintf("query = %s", query))
	res, err := se.db.Query(query, varArgs...)
	// log.Infoln(fmt.Sprintf("res= %v, err = %v", res, err))
	return res, err
}

func singleColRowsToString(rows *sql.Rows) (string, error) {
	var acc []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return "", fmt.Errorf("could not stringify sql rows: %v", err)
		}
		acc = append(acc, s)
	}
	return strings.Join(acc, " "), nil
}
