package dto

import (
	"gopkg.in/yaml.v2"
)

const (
	defaultMaxTxnDepth = -1
)

type TxnCoordinatorCfg struct {
	MaxTxnDepth *int `json:"maxTransactionDepth" yaml:"maxTransactionDepth"`
}

func (t TxnCoordinatorCfg) GetMaxTxnDepth() int {
	if t.MaxTxnDepth == nil {
		return defaultMaxTxnDepth
	}
	return *t.MaxTxnDepth
}

func GetTxnCoordinatorCfgCfg(s string) (TxnCoordinatorCfg, error) {
	rv := TxnCoordinatorCfg{}
	err := yaml.Unmarshal([]byte(s), &rv)
	return rv, err
}
