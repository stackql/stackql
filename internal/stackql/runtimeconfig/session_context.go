package dto

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/constants"

	"gopkg.in/yaml.v3"
)

var (
	_ SessionContext = &sessionContext{}
)

type SessionCtxConfig struct {
	IsolationLevel string `json:"isolation_level" yaml:"isolation_level"`
	RollbackType   string `json:"rollback_type" yaml:"rollback_type"`
}

type SessionContext interface {
	Clone() SessionContext
	GetIsolationLevel() constants.IsolationLevel
	UpdateIsolationLevel(string) error
	GetRollbackType() constants.RollbackType
	UpdateRollbackType(string) error
}

type sessionContext struct {
	isolationLevel constants.IsolationLevel
	rollbackType   constants.RollbackType
}

func NewSessionContext(cfgStr string) (SessionContext, error) {
	var cfg SessionCtxConfig
	err := yaml.Unmarshal([]byte(cfgStr), &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal session context config: %w", err)
	}
	isolationLevelStr := cfg.IsolationLevel
	if isolationLevelStr == "" {
		isolationLevelStr = constants.ReadUncommittedStr
	}
	rollbackTypeStr := cfg.RollbackType
	if rollbackTypeStr == "" {
		rollbackTypeStr = constants.NopRollbackStr
	}
	isolationLevel, isolationLevelErr := inferIsolationLevel(isolationLevelStr)
	if isolationLevelErr != nil {
		return nil, fmt.Errorf("failed to infer isolation level: %w", isolationLevelErr)
	}
	rollbackType, rollbackTypeErr := inferRollbackType(rollbackTypeStr)
	if rollbackTypeErr != nil {
		return nil, fmt.Errorf("failed to infer rollback type: %w", rollbackTypeErr)
	}
	return &sessionContext{
		isolationLevel: isolationLevel,
		rollbackType:   rollbackType,
	}, nil
}

func (sc *sessionContext) GetIsolationLevel() constants.IsolationLevel {
	return sc.isolationLevel
}

func inferIsolationLevel(isolationLevelStr string) (constants.IsolationLevel, error) {
	isolationLevel := constants.ReadUncommitted
	switch isolationLevelStr {
	case constants.ReadUncommittedStr:
		isolationLevel = constants.ReadUncommitted
	case constants.ReadCommittedStr:
		isolationLevel = constants.ReadCommitted
	case constants.RepeatableReadStr:
		isolationLevel = constants.RepeatableRead
	case constants.SerializableStr:
		isolationLevel = constants.Serializable
	default:
		return isolationLevel, fmt.Errorf("invalid isolation level: %s", isolationLevelStr)
	}
	return isolationLevel, nil
}

func inferRollbackType(rollbackTypeStr string) (constants.RollbackType, error) {
	rollbackType := constants.NopRollback
	switch rollbackTypeStr {
	case constants.NopRollbackStr:
		rollbackType = constants.NopRollback
	case constants.EagerRollbackStr:
		rollbackType = constants.EagerRollback
	default:
		return rollbackType, fmt.Errorf("invalid rollback type: %s", rollbackTypeStr)
	}
	return rollbackType, nil
}

func (sc *sessionContext) UpdateIsolationLevel(isolationLevelStr string) error {
	isolationLevel, isolationLevelErr := inferIsolationLevel(isolationLevelStr)
	if isolationLevelErr != nil {
		return fmt.Errorf("failed to infer isolation level: %w", isolationLevelErr)
	}
	sc.isolationLevel = isolationLevel
	return nil
}

func (sc *sessionContext) GetRollbackType() constants.RollbackType {
	return sc.rollbackType
}

func (sc *sessionContext) UpdateRollbackType(rollbackTypeStr string) error {
	rollbackType, rollbackTypeErr := inferRollbackType(rollbackTypeStr)
	if rollbackTypeErr != nil {
		return fmt.Errorf("failed to infer rollback type: %w", rollbackTypeErr)
	}
	sc.rollbackType = rollbackType
	return nil
}

func (sc *sessionContext) Clone() SessionContext {
	return &sessionContext{
		isolationLevel: sc.isolationLevel,
		rollbackType:   sc.rollbackType,
	}
}
