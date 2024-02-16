package primitivebuilder

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/builder_input"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/util"
)

type RawNativeExec struct {
	graph       primitivegraph.PrimitiveGraphHolder
	handlerCtx  handler.HandlerContext
	txnCtrlCtr  internaldto.TxnControlCounters
	root        primitivegraph.PrimitiveNode
	nativeQuery string
	bldrInput   builder_input.BuilderInput
}

func NewRawNativeExec(
	graph primitivegraph.PrimitiveGraphHolder,
	handlerCtx handler.HandlerContext,
	txnCtrlCtr internaldto.TxnControlCounters,
	nativeQuery string,
	bldrInput builder_input.BuilderInput,
) Builder {
	return &RawNativeExec{
		graph:       graph,
		handlerCtx:  handlerCtx,
		txnCtrlCtr:  txnCtrlCtr,
		nativeQuery: nativeQuery,
		bldrInput:   bldrInput,
	}
}

func (ss *RawNativeExec) GetRoot() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *RawNativeExec) GetTail() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *RawNativeExec) Build() error {
	dependencyNode, dependencyNodeExists := ss.bldrInput.GetDependencyNode()
	selectEx := func(_ primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
		// select phase
		logging.GetLogger().Infoln(fmt.Sprintf("running native query: '''%s''' ", ss.nativeQuery))

		row, err := ss.handlerCtx.GetSQLEngine().Exec(ss.nativeQuery)

		if row != nil {
			rowsAffected, countErr := row.RowsAffected()
			if countErr == nil {
				logging.GetLogger().Debugf("native exec rows affected = %d\n", rowsAffected)
			} else {
				logging.GetLogger().Infof("native exec affected count error = '%s'\n", countErr.Error())
			}
		}

		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}

		return util.PrepareResultSet(
			internaldto.NewPrepareResultSetPlusRawDTO(
				nil,
				nil,
				nil,
				nil,
				nil,
				internaldto.NewBackendMessages([]string{"exec completed"}), nil,
				ss.handlerCtx.GetTypingConfig()),
		)
	}

	graph := ss.graph
	selectNode := graph.CreatePrimitiveNode(primitive.NewLocalPrimitive(selectEx))
	if dependencyNodeExists {
		graph.NewDependency(dependencyNode, selectNode, 1.0)
	}
	ss.root = selectNode

	return nil
}
