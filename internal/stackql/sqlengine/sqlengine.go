package sqlengine

import (
	"database/sql"
	"time"

	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/sqlcontrol"
)

type SQLEngine interface {
	GetDB() (*sql.DB, error)
	Exec(string, ...interface{}) (sql.Result, error)
	Query(string, ...interface{}) (*sql.Rows, error)
	ExecFileLocal(string) error
	ExecFile(string) error
	ExecInTxn(queries []string) error
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
	IsMemory() bool
	IsTablePresent(string, string, string) bool
	TableOldestUpdateUTC(string, string, string, string) (time.Time, *dto.TxnControlCounters)
}

func NewSQLEngine(cfg SQLEngineConfig, controlAttributes sqlcontrol.ControlAttributes) (SQLEngine, error) {
	return newSQLiteEngine(cfg, controlAttributes)
}
