package primitivebuilder

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/util"
)

type RawNativeExec struct {
	graph       *primitivegraph.PrimitiveGraph
	handlerCtx  *handler.HandlerContext
	txnCtrlCtr  dto.TxnControlCounters
	root        primitivegraph.PrimitiveNode
	nativeQuery string
}

func NewRawNativeExec(
	graph *primitivegraph.PrimitiveGraph,
	handlerCtx *handler.HandlerContext,
	txnCtrlCtr dto.TxnControlCounters,
	nativeQuery string,
) Builder {
	return &RawNativeExec{
		graph:       graph,
		handlerCtx:  handlerCtx,
		txnCtrlCtr:  txnCtrlCtr,
		nativeQuery: nativeQuery,
	}
}

func (ss *RawNativeExec) GetRoot() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *RawNativeExec) GetTail() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *RawNativeExec) Build() error {

	selectEx := func(pc primitive.IPrimitiveCtx) dto.ExecutorOutput {

		// select phase
		logging.GetLogger().Infoln(fmt.Sprintf("running native query: '''%s''' ", ss.nativeQuery))

		row, err := ss.handlerCtx.SQLEngine.Exec(ss.nativeQuery)

		if row != nil {
			rowsAffected, countErr := row.RowsAffected()
			if countErr == nil {
				logging.GetLogger().Debugf("native exec rows affected = %d\n", rowsAffected)
			} else {
				logging.GetLogger().Infof("native exec affected count error = '%s'\n", countErr.Error())
			}
		}

		if err != nil {
			return dto.NewErroneousExecutorOutput(err)
		}

		return util.PrepareResultSet(
			dto.NewPrepareResultSetPlusRawDTO(
				nil,
				nil,
				nil,
				nil,
				nil,
				&dto.BackendMessages{WorkingMessages: []string{"exec completed"}}, nil),
		)
	}

	graph := ss.graph
	selectNode := graph.CreatePrimitiveNode(primitive.NewLocalPrimitive(selectEx))
	ss.root = selectNode

	return nil
}
