package sqlengine

import (
	"database/sql"

	"github.com/stackql/stackql/internal/stackql/dto"
)

type SQLEngine interface {
	GetDB() (*sql.DB, error)
	Exec(string, ...interface{}) (sql.Result, error)
	Query(string, ...interface{}) (*sql.Rows, error)
	ExecFileLocal(string) error
	ExecFile(string) error
	GCCollectObsolete(*dto.TxnControlCounters) error
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

func NewSQLEngine(cfg SQLEngineConfig) (SQLEngine, error) {
	return newSQLiteEngine(cfg)
}
