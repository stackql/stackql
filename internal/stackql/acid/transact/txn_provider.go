package transact

import (
	"sync"

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
	GetOrchestrator(handler.HandlerContext) (Orchestrator, error)
}

type standardProvider struct {
	ctx txn_context.ITransactionCoordinatorContext
}

func (sp *standardProvider) GetOrchestrator(handlerCtx handler.HandlerContext) (Orchestrator, error) {
	txnCoordinator := newTxnCoordinator(handlerCtx, sp.ctx)
	return newTxnOrchestrator(handlerCtx, txnCoordinator)
}

func newTxnCoordinator(handlerCtx handler.HandlerContext,
	ctx txn_context.ITransactionCoordinatorContext) Coordinator {
	maxTxnDepth := defaultMaxStackDepth
	if ctx != nil {
		maxTxnDepth = ctx.GetMaxStackDepth()
	}
	return NewCoordinator(handlerCtx, maxTxnDepth)
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
