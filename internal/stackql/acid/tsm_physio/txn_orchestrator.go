package tsm_physio //nolint:stylecheck // prefer this nomenclature

import (
	"fmt"
	"strings"

	"github.com/stackql/any-sdk/pkg/constants"
	"github.com/stackql/stackql/internal/stackql/acid/tsm"
	"github.com/stackql/stackql/internal/stackql/acid/txn_context"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
)

type Orchestrator interface {
	ProcessQueryOrQueries(
		handlerCtx handler.HandlerContext,
	) ([]internaldto.ExecutorOutput, bool)
}

func newTxnOrchestrator(
	tsmInstance tsm.TSM,
	handlerCtx handler.HandlerContext,
	txnCoordinator Coordinator) (Orchestrator, error) {
	rollbackType := handlerCtx.GetRollbackType()
	switch rollbackType {
	case constants.NopRollback:
		return newStdTxnOrchestrator(tsmInstance, handlerCtx, txnCoordinator)
	case constants.EagerRollback:
		return newBestEffortTxnOrchestrator(tsmInstance, handlerCtx, txnCoordinator)
	default:
		return newStdTxnOrchestrator(tsmInstance, handlerCtx, txnCoordinator)
	}
}

func newStdTxnOrchestrator(
	tsmInstance tsm.TSM,
	_ handler.HandlerContext,
	txnCoordinator Coordinator) (Orchestrator, error) {
	return &standardOrchestrator{
		tsmInstance:    tsmInstance,
		txnCoordinator: txnCoordinator,
	}, nil
}

func newBestEffortTxnOrchestrator(
	tsmInstance tsm.TSM,
	_ handler.HandlerContext,
	txnCoordinator Coordinator) (Orchestrator, error) {
	return &bestEffortOrchestrator{
		tsmInstance:    tsmInstance,
		txnCoordinator: txnCoordinator,
	}, nil
}

type standardOrchestrator struct {
	tsmInstance    tsm.TSM
	txnCoordinator Coordinator
}

func (orc *standardOrchestrator) ProcessQueryOrQueries(
	handlerCtx handler.HandlerContext,
) ([]internaldto.ExecutorOutput, bool) {
	return orc.processQueryOrQueries(handlerCtx)
}

func (orc *standardOrchestrator) processQueryOrQueries(
	handlerCtx handler.HandlerContext,
) ([]internaldto.ExecutorOutput, bool) {
	var retVal []internaldto.ExecutorOutput
	cmdString := handlerCtx.GetRawQuery()
	for _, s := range strings.Split(cmdString, ";") {
		response, hasResponse := orc.processQuery(handlerCtx, s)
		if hasResponse {
			retVal = append(retVal, response...)
		}
	}
	return retVal, len(retVal) > 0
}

//nolint:gocognit // TODO: review
func (orc *standardOrchestrator) processQuery(
	handlerCtx handler.HandlerContext,
	query string,
) ([]internaldto.ExecutorOutput, bool) {
	if query == "" {
		return nil, false
	}
	clonedCtx := handlerCtx.Clone()
	clonedCtx.SetQuery(query)
	transactStatement := NewStatement(query, clonedCtx, txn_context.NewTransactionContext(orc.txnCoordinator.Depth()))
	prepareErr := transactStatement.Prepare()
	if prepareErr != nil {
		return []internaldto.ExecutorOutput{
			internaldto.NewErroneousExecutorOutput(prepareErr),
		}, true
	}
	isReadOnly := transactStatement.IsReadOnly()
	// TODO: implement eager execution for non-mutating statements
	//       and lazy execution for mutating statements.
	// TODO: implement transaction stack.
	if transactStatement.IsBegin() { //nolint:gocritic,nestif // TODO: review
		txnCoordinator, beginErr := orc.txnCoordinator.Begin()
		if beginErr != nil {
			return []internaldto.ExecutorOutput{
				internaldto.NewErroneousExecutorOutput(beginErr),
			}, true
		}
		orc.txnCoordinator = txnCoordinator
		return []internaldto.ExecutorOutput{
			internaldto.NewNopEmptyExecutorOutput([]string{"OK"}),
		}, true
	} else if transactStatement.IsCommit() {
		commitCoDomain := orc.txnCoordinator.Commit()
		commitErr, commitErrExists := commitCoDomain.GetError()
		if commitErrExists {
			retVal := []internaldto.ExecutorOutput{
				internaldto.NewErroneousExecutorOutput(commitErr),
			}
			undoLog, undoLogExists := commitCoDomain.GetUndoLog()
			if undoLogExists && undoLog != nil {
				humanReadable := undoLog.GetHumanReadable()
				if len(humanReadable) > 0 {
					displayUndoLogs := make([]string, len(humanReadable))
					for i, h := range humanReadable {
						displayUndoLogs[i] = fmt.Sprintf("UNDO required: %s", h)
					}
					retVal = append(retVal, internaldto.NewNopEmptyExecutorOutput(displayUndoLogs))
				}
			}
			return retVal, true
		}
		retVal := commitCoDomain.GetExecutorOutput()
		parent, hasParent := orc.txnCoordinator.GetParent()
		if hasParent {
			orc.txnCoordinator = parent
			retVal = append(retVal, internaldto.NewNopEmptyExecutorOutput([]string{"OK"}))
			return retVal, true
		}
		noParentErr := fmt.Errorf(noParentMessage)
		retVal = append(retVal, internaldto.NewErroneousExecutorOutput(noParentErr))
		return retVal, true
	} else if transactStatement.IsRollback() {
		var retVal []internaldto.ExecutorOutput
		rollbackREsponse := orc.txnCoordinator.Rollback()
		rollbackErr, rollbackErrExists := rollbackREsponse.GetError()
		if rollbackErrExists {
			retVal = append(retVal, internaldto.NewErroneousExecutorOutput(rollbackErr))
		}
		parent, hasParent := orc.txnCoordinator.GetParent()
		if hasParent {
			orc.txnCoordinator = parent
			retVal = append(retVal, internaldto.NewNopEmptyExecutorOutput([]string{"Rollback OK"}))
			return retVal, true
		}
		retVal = append(
			retVal,
			internaldto.NewErroneousExecutorOutput(
				fmt.Errorf(noParentMessage)),
		)
		return retVal, true
	}
	if isReadOnly || orc.txnCoordinator.IsRoot() {
		stmtOutput := transactStatement.Execute()
		return []internaldto.ExecutorOutput{
			stmtOutput,
		}, true
	}
	orc.txnCoordinator.Enqueue(transactStatement) //nolint:errcheck // TODO: investigate
	return []internaldto.ExecutorOutput{
		internaldto.NewNopEmptyExecutorOutput([]string{"mutating statement queued"}),
	}, true
}
