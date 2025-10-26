package tsm_physio //nolint:revive,stylecheck // prefer this nomenclature

import (
	"fmt"
	"strings"

	"github.com/stackql/stackql/internal/stackql/acid/binlog"
	"github.com/stackql/stackql/internal/stackql/acid/tsm"
	"github.com/stackql/stackql/internal/stackql/acid/txn_context"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
)

var (
	_ Orchestrator = &bestEffortOrchestrator{}
)

// This orchestrator:
//   - Supports a simple reversibility semantic.
//   - In cases of network partitioning or other failures,
//     it will simply spew suggested undo logs.
//
//nolint:unused // TODO: fix this.
type bestEffortOrchestrator struct {
	tsmInstance    tsm.TSM
	txnCoordinator Coordinator
	undoLogs       []binlog.LogEntry
	redoLogs       []binlog.LogEntry
	redoGraphs     []primitivegraph.PrimitiveGraph
	undoGraphs     []primitivegraph.PrimitiveGraph
}

func (orc *bestEffortOrchestrator) ProcessQueryOrQueries(
	handlerCtx handler.HandlerContext,
) ([]internaldto.ExecutorOutput, bool) {
	return orc.processQueryOrQueries(handlerCtx)
}

func (orc *bestEffortOrchestrator) processQueryOrQueries(
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

func (orc *bestEffortOrchestrator) undo(precedingMessages []string) ([]internaldto.ExecutorOutput, bool) {
	rollbackREsponse := orc.txnCoordinator.Rollback()
	rollbackErr, rollbackErrExists := rollbackREsponse.GetError()
	if rollbackErrExists {
		return []internaldto.ExecutorOutput{
			internaldto.NewNopEmptyExecutorOutput(
				precedingMessages,
			),
			internaldto.NewErroneousExecutorOutput(rollbackErr),
		}, true
	}
	return []internaldto.ExecutorOutput{
		internaldto.NewNopEmptyExecutorOutput(
			precedingMessages,
		),
		internaldto.NewNopEmptyExecutorOutput(
			[]string{
				"rollback successful",
			},
		),
	}, true
}

//nolint:gocognit,funlen // TODO: review
func (orc *bestEffortOrchestrator) processQuery(
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
			return orc.undo([]string{
				commitErr.Error(),
			})
		}
		retVal := commitCoDomain.GetExecutorOutput()
		parent, hasParent := orc.txnCoordinator.GetParent()
		if hasParent {
			orc.txnCoordinator = parent
			retVal = append(retVal, internaldto.NewNopEmptyExecutorOutput([]string{"OK"}))
			return retVal, true
		}
		noParentErr := fmt.Errorf("%s", noParentMessage)
		retVal = append(retVal, internaldto.NewErroneousExecutorOutput(noParentErr))
		return retVal, true
	} else if transactStatement.IsRollback() {
		var retVal []internaldto.ExecutorOutput
		rollbackREsponse := orc.txnCoordinator.Rollback()
		rollbackErr, rollbackErrExists := rollbackREsponse.GetError()
		if rollbackErrExists {
			retVal = append(retVal, internaldto.NewErroneousExecutorOutput(rollbackErr))
			retVal = append(retVal, internaldto.NewErroneousExecutorOutput(
				fmt.Errorf("Rollback failed")))
			return retVal, true
		}
		parent, hasParent := orc.txnCoordinator.GetParent()
		if hasParent {
			orc.txnCoordinator = parent
			for _, g := range orc.undoGraphs {
				undoOutput := g.Execute(nil)
				if undoOutput.GetError() != nil {
					retVal = append(retVal, internaldto.NewErroneousExecutorOutput(undoOutput.GetError()))
					retVal = append(retVal, internaldto.NewErroneousExecutorOutput(
						fmt.Errorf("Rollback failed")))
					return retVal, true
				}
			}
			retVal = append(retVal, internaldto.NewNopEmptyExecutorOutput([]string{"Rollback OK"}))
			return retVal, true
		}
		retVal = append(
			retVal,
			internaldto.NewErroneousExecutorOutput(
				fmt.Errorf("%s", noParentMessage)),
		)
		return retVal, true
	}
	if isReadOnly || orc.txnCoordinator.IsRoot() {
		stmtOutput := transactStatement.Execute()
		return []internaldto.ExecutorOutput{
			stmtOutput,
		}, true
	}

	primitiveGraphHolder, primitiveGraphHolderExists := transactStatement.GetPrimitiveGraphHolder()
	if !primitiveGraphHolderExists {
		// bail
		undoMessage := "primitive graph holder does not exist"
		if query != "" {
			undoMessage = fmt.Sprintf("primitive graph holder does not exist for query: '%s'", query)
		}
		return orc.undo([]string{
			undoMessage,
		})
	}

	enqueueError := orc.txnCoordinator.Enqueue(transactStatement)

	// Before bailing on eager execution error,
	// first assemble undo graph.
	undoGraphSize := primitiveGraphHolder.GetInversePrimitiveGraph().Size()
	if undoGraphSize == 0 {
		// bail
		undoMessage := "undo graph does not exist"
		if query != "" {
			undoMessage = fmt.Sprintf("undo graph does not exist for query: '%s'", query)
		}
		return orc.undo([]string{
			undoMessage,
		})
	}

	redoLog, redoLogExists := transactStatement.GetRedoLog()
	if redoLogExists {
		orc.redoLogs = append(orc.redoLogs, redoLog)
	}

	if enqueueError != nil {
		// bail
		return orc.undo([]string{
			enqueueError.Error(),
		})
	}
	output := transactStatement.Execute()
	if output.GetError() != nil {
		// bail
		return orc.undo([]string{
			output.GetError().Error(),
		})
	}
	return []internaldto.ExecutorOutput{output}, true
}
