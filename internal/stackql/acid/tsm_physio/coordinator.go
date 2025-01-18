package tsm_physio //nolint:revive,stylecheck // prefer this nomenclature

import (
	"github.com/stackql/any-sdk/pkg/constants"
	"github.com/stackql/stackql/internal/stackql/acid/acid_dto"
	"github.com/stackql/stackql/internal/stackql/acid/tsm"
	"github.com/stackql/stackql/internal/stackql/handler"
)

func newCoordinator(tsmInstance tsm.TSM, handlerCtx handler.HandlerContext, maxTxnDepth int) Coordinator {
	rollbackType := handlerCtx.GetRollbackType()
	switch rollbackType {
	case constants.NopRollback:
		return newBasicLazyTransactionCoordinator(tsmInstance, nil, maxTxnDepth)
	case constants.EagerRollback:
		return newBasicBestEffortTransactionCoordinator(tsmInstance, handlerCtx, nil, maxTxnDepth)
	default:
		return newBasicLazyTransactionCoordinator(tsmInstance, nil, maxTxnDepth)
	}
}

// The transaction coordinator ensures
// that undo and redo logs are kept
// and that 2PC is performed.
type Coordinator interface {
	Statement
	// Begin a new transaction.
	Begin() (Coordinator, error)
	// Commit the current transaction.
	Commit() acid_dto.CommitCoDomain
	// Rollback the current transaction.
	Rollback() acid_dto.CommitCoDomain
	// Enqueue a transaction operation.
	// This method will return an error
	// in the case that the transaction
	// context disallows a particular
	// operation or type of operation.
	Enqueue(Statement) error
	// Get the depth of transaction nesting.
	Depth() int
	// Get the parent transaction manager.
	GetParent() (Coordinator, bool)
	//
	IsRoot() bool
}
