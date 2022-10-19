package dto

import (
	"gopkg.in/yaml.v2"
)

type GCCfg struct {
	IsEager bool `json:"isEager" yaml:"isEager"`
}

func GetGCCfg(s string) (GCCfg, error) {
	rv := GCCfg{}
	err := yaml.Unmarshal([]byte(s), &rv)
	return rv, err
}
