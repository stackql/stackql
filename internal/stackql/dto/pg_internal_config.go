package dto

import (
	"gopkg.in/yaml.v2"
)

type DBMSInternalCfg struct {
	ShowRegex   string `json:"showRegex" yaml:"showRegex"`
	TableRegex  string `json:"tableRegex" yaml:"tableRegex"`
	SchemaRegex string `json:"schemaRegex" yaml:"schemaRegex"`
	FuncRegex   string `json:"funcRegex" yaml:"funcRegex"`
}

func GetDBMSInternalCfg(s string) (DBMSInternalCfg, error) {
	rv := DBMSInternalCfg{}
	err := yaml.Unmarshal([]byte(s), &rv)
	return rv, err
}
