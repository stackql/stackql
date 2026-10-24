package dto

import (
	"gopkg.in/yaml.v2"
)

type KStoreCfg struct {
	IsPlaceholder bool `json:"isPlaceholder" yaml:"isPlaceholder"`
}

func GetKStoreCfg(s string) (KStoreCfg, error) {
	rv := KStoreCfg{}
	err := yaml.Unmarshal([]byte(s), &rv)
	return rv, err
}
