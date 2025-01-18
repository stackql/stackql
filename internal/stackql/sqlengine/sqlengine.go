package sqlengine

import (
	"database/sql"
	"fmt"

	"github.com/stackql/any-sdk/pkg/constants"
	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/sqlcontrol"
)

type SQLEngine interface {
	GetDB() (*sql.DB, error)
	GetTx() (*sql.Tx, error)
	Exec(string, ...interface{}) (sql.Result, error)
	Query(string, ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	ExecFileLocal(string) error
	ExecFile(string) error
	ExecInTxn(queries []string) error
	GetCurrentGenerationID() (int, error)
	GetNextGenerationID() (int, error)
	GetCurrentSessionID(int) (int, error)
	GetNextSessionID(int) (int, error)
	GetCurrentDiscoveryGenerationID(discoveryID string) (int, error)
	GetNextDiscoveryGenerationID(discoveryID string) (int, error)
	CacheStoreGet(string) ([]byte, error)
	CacheStoreGetAll() ([]internaldto.KeyVal, error)
	CacheStorePut(string, []byte, string, int) error
	IsMemory() bool
}

func NewSQLEngine(cfg dto.SQLBackendCfg, controlAttributes sqlcontrol.ControlAttributes) (SQLEngine, error) {
	switch cfg.DBEngine {
	case constants.DBEngineSQLite3Embedded:
		return newSQLiteEmbeddedEngine(cfg, controlAttributes)
	case constants.DBEnginePostgresTCP:
		return newPostgresTCPEngine(cfg, controlAttributes)
	case constants.SQLDialectSnowflake:
		return newSnowflakeTCPEngine(cfg, controlAttributes)
	default:
		return nil, fmt.Errorf(`SQL backend DB Engine of type '%s' is not permitted`, cfg.DBEngine)
	}
}
