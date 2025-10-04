package tsm_physio //nolint:revive,stylecheck // prefer this nomenclature

import (
	"sync"

	"github.com/stackql/stackql/internal/stackql/acid/tsm"
	"github.com/stackql/stackql/internal/stackql/acid/txn_context"
	"github.com/stackql/stackql/internal/stackql/handler"
)

//nolint:gochecknoglobals // singleton pattern
var (
	providerOnce      sync.Once
	providerSingleton Provider
	_                 Provider = &standardProvider{}
	noParentMessage   string   = "no parent transaction manager available" //nolint:gochecknoglobals,revive,lll // permissable
)

const (
	defaultMaxStackDepth = 1
)

// The transaction provider is singleton
// that orchestrates transaction managers.
type Provider interface {
	// Create a new transaction manager.
	getOrchestrator(handler.HandlerContext) (Orchestrator, error)
	GetTSM(handlerCtx handler.HandlerContext) (tsm.TSM, error)
}

type standardProvider struct {
	ctx txn_context.ITransactionCoordinatorContext
}

func (sp *standardProvider) getOrchestrator(handlerCtx handler.HandlerContext) (Orchestrator, error) {
	tsmInstance, walError := GetTSM(handlerCtx)
	if walError != nil {
		return nil, walError
	}
	txnCoordinator := newTxnCoordinator(tsmInstance, handlerCtx, sp.ctx)
	orc, err := newTxnOrchestrator(tsmInstance, handlerCtx, txnCoordinator)
	return orc, err
}

func (sp *standardProvider) GetTSM(handlerCtx handler.HandlerContext) (tsm.TSM, error) {
	return GetTSM(handlerCtx)
}

func newTxnCoordinator(tsmInstance tsm.TSM, handlerCtx handler.HandlerContext,
	ctx txn_context.ITransactionCoordinatorContext) Coordinator {
	maxTxnDepth := defaultMaxStackDepth
	if ctx != nil {
		maxTxnDepth = ctx.GetMaxStackDepth()
	}
	return newCoordinator(tsmInstance, handlerCtx, maxTxnDepth)
}

func GetProviderInstance(ctx txn_context.ITransactionCoordinatorContext) (Provider, error) {
	var err error
	providerOnce.Do(func() {
		if err != nil {
			return
		}
		providerSingleton = &standardProvider{
			ctx: ctx,
		}
	})
	return providerSingleton, err
}

func NewOrchestrator(handlerCtx handler.HandlerContext) (Orchestrator, error) {
	txnProvider, txnProviderErr := GetProviderInstance(
		handlerCtx.GetTxnCoordinatorCtx())
	if txnProviderErr != nil {
		return nil, txnProviderErr
	}
	return txnProvider.getOrchestrator(handlerCtx)
}
