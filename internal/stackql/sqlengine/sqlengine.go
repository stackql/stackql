package sqlengine

import (
	"database/sql"
	"fmt"

	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/sqlcontrol"
)

type SQLEngine interface {
	GetDB() (*sql.DB, error)
	Exec(string, ...interface{}) (sql.Result, error)
	Query(string, ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	ExecFileLocal(string) error
	ExecFile(string) error
	ExecInTxn(queries []string) error
	GetCurrentGenerationId() (int, error)
	GetNextGenerationId() (int, error)
	GetCurrentSessionId(int) (int, error)
	GetNextSessionId(int) (int, error)
	GetCurrentDiscoveryGenerationId(discoveryID string) (int, error)
	GetNextDiscoveryGenerationId(discoveryID string) (int, error)
	CacheStoreGet(string) ([]byte, error)
	CacheStoreGetAll() ([]dto.KeyVal, error)
	CacheStorePut(string, []byte, string, int) error
	IsMemory() bool
}

func NewSQLEngine(cfg dto.SQLBackendCfg, controlAttributes sqlcontrol.ControlAttributes) (SQLEngine, error) {
	switch cfg.DbEngine {
	case constants.DbEngineSQLite3Embedded:
		return newSQLiteInProcessEngine(cfg, controlAttributes)
	case constants.DbEnginePostgresTCP:
		return newPostgresTcpEngine(cfg, controlAttributes)
	default:
		return nil, fmt.Errorf(`SQL backend DB Engine of type '%s' is not permitted`, cfg.DbEngine)
	}
}
