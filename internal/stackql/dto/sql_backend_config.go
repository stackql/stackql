package dto

import (
	"github.com/stackql/stackql/internal/stackql/constants"
	"gopkg.in/yaml.v2"
)

type SQLBackendCfg struct {
	DbEngine              string `json:"dbEngine" yaml:"dbEngine"`
	DSN                   string `json:"dsn" yaml:"dsn"`
	DbInitFilePath        string `json:"dbInitFilepath" yaml:"dbInitFilepath"`
	SQLDialect            string `json:"sqlDialect" yaml:"sqlDialect"`
	InitMaxRetries        int    `json:"initMaxRetries" yaml:"initMaxRetries"`
	InitRetryInitialDelay int    `json:"initRetryInitialDelay" yaml:"initRetryInitialDelay"`
}

func GetSQLBackendCfg(s string) (SQLBackendCfg, error) {
	rv := SQLBackendCfg{}
	err := yaml.Unmarshal([]byte(s), &rv)
	if rv.DbEngine == "" {
		rv.DbEngine = constants.DbEngineDefault
	}
	if rv.SQLDialect == "" {
		rv.SQLDialect = constants.SQLDialectDefault
	}
	if rv.InitMaxRetries < 1 {
		rv.InitMaxRetries = 10
	}
	if rv.InitRetryInitialDelay < 1 {
		rv.InitRetryInitialDelay = 10
	}
	return rv, err
}
