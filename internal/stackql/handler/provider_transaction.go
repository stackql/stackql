package handler

import (
	"context"
	"fmt"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/intrinsic"
)

// ProviderTransactionConfig is host addressing/configuration, not desired state.
// Transaction lifecycle and resource ownership are implemented by Omni.
type ProviderTransactionConfig struct {
	omnisdk.TransactionConfig
	ResumeID string `json:"resume_id,omitempty"`
}

func (hc *standardHandlerContext) configureProviderTransaction(parts []string, rhs interface{}, scope string) error {
	if scope != "" && scope != "session" {
		return fmt.Errorf("transaction configuration requires session scope")
	}
	state := hc.providerSessionState()
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.transactionSession != nil || state.transactionStarting {
		return fmt.Errorf("transaction configuration is frozen until COMMIT or ROLLBACK")
	}
	if len(parts) != 1 {
		return fmt.Errorf("set the complete transaction configuration object")
	}
	if rhs == nil {
		state.transactionConfig = nil
		return nil
	}
	var cfg ProviderTransactionConfig
	if err := decodeSessionObject(rhs, &cfg); err != nil {
		return err
	}
	if _, err := intrinsic.ManagedStateDirectory(hc, cfg.State); err != nil {
		return err
	}
	if cfg.Collection == "" {
		return fmt.Errorf("transaction configuration requires collection")
	}
	state.transactionConfig = &cfg
	return nil
}
func (hc *standardHandlerContext) ProviderTransaction() (*omnisdk.TransactionSession, omnisdk.TransactionHandle, bool) {
	state := hc.providerSessionState()
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.transactionSession, state.transactionHandle, state.transactionSession != nil
}
func (hc *standardHandlerContext) BeginProviderTransaction(ctx context.Context) error {
	state := hc.providerSessionState()
	state.mu.Lock()
	if state.transactionSession != nil || state.transactionStarting {
		state.mu.Unlock()
		return fmt.Errorf("nested provider transactions are unsupported")
	}
	cfg := ProviderTransactionConfig{TransactionConfig: omnisdk.TransactionConfig{
		Collection: "sql", DynamicScope: true,
	}}
	if state.transactionConfig != nil {
		cfg = *state.transactionConfig
	}
	state.transactionStarting = true
	state.mu.Unlock()
	defer func() { state.mu.Lock(); state.transactionStarting = false; state.mu.Unlock() }()
	session, err := omnisdk.NewTransactionSession(hc.GetCaller())
	if err != nil {
		return err
	}
	cfg.State, err = intrinsic.ManagedStateDirectory(hc, cfg.State)
	if err != nil {
		session.Close()
		return err
	}
	if cfg.Mode == "" {
		cfg.Mode = omnisdk.TransactionEager
		if hc.GetRollbackType() == constants.NopRollback {
			cfg.Mode = omnisdk.TransactionLazy
		}
	}
	var handle omnisdk.TransactionHandle
	if cfg.ResumeID != "" {
		handle, err = session.Resume(ctx, cfg.TransactionConfig, cfg.ResumeID)
	} else {
		handle, err = intrinsic.OpenProviderTransaction(hc, session, cfg.TransactionConfig)
	}
	if err != nil {
		session.Close()
		return err
	}
	state.mu.Lock()
	state.transactionSession = session
	state.transactionHandle = handle
	state.mu.Unlock()
	return nil
}

func (hc *standardHandlerContext) ClearProviderTransaction() {
	state := hc.providerSessionState()
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.transactionSession != nil {
		state.transactionSession.Close()
	}
	state.transactionSession = nil
	state.transactionHandle = omnisdk.TransactionHandle{}
}
