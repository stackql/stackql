package dto

import (
	"gopkg.in/yaml.v2"
)

type DBMSInternalCfg struct {
	ShowRegex  string `json:"showRegex" yaml:"showRegex"`
	TableRegex string `json:"tableRegex" yaml:"tableRegex"`
}

func GetDBMSInternalCfg(s string) (DBMSInternalCfg, error) {
	rv := DBMSInternalCfg{}
	err := yaml.Unmarshal([]byte(s), &rv)
	return rv, err
}
