package sqlengine

import (
	"github.com/stackql/stackql/internal/stackql/dto"
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
