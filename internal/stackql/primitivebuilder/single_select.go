package primitivebuilder

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/streaming"
	"github.com/stackql/stackql/internal/stackql/tableinsertioncontainer"
)

type SingleSelect struct {
	graph                      *primitivegraph.PrimitiveGraph
	handlerCtx                 handler.HandlerContext
	drmCfg                     drm.DRMConfig
	selectPreparedStatementCtx drm.PreparedStatementCtx
	insertionContainers        []tableinsertioncontainer.TableInsertionContainer
	txnCtrlCtr                 internaldto.TxnControlCounters
	rowSort                    func(map[string]map[string]interface{}) []string
	root                       primitivegraph.PrimitiveNode
	stream                     streaming.MapStream
}

func NewSingleSelect(
	graph *primitivegraph.PrimitiveGraph,
	handlerCtx handler.HandlerContext,
	selectCtx drm.PreparedStatementCtx,
	insertionContainers []tableinsertioncontainer.TableInsertionContainer,
	rowSort func(map[string]map[string]interface{}) []string,
	stream streaming.MapStream,
) Builder {
	return &SingleSelect{
		graph:                      graph,
		handlerCtx:                 handlerCtx,
		rowSort:                    rowSort,
		drmCfg:                     handlerCtx.GetDrmConfig(),
		selectPreparedStatementCtx: selectCtx,
		insertionContainers:        insertionContainers,
		txnCtrlCtr:                 selectCtx.GetGCCtrlCtrs(),
		stream:                     stream,
	}
}

func (ss *SingleSelect) GetRoot() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *SingleSelect) GetTail() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *SingleSelect) Build() error {

	selectEx := func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {

		// select phase
		logging.GetLogger().Infoln(fmt.Sprintf("running select with control parameters: %v", ss.selectPreparedStatementCtx.GetGCCtrlCtrs()))

		return prepareGolangResult(ss.handlerCtx.GetSQLEngine(), ss.handlerCtx.GetOutErrFile(), drm.NewPreparedStatementParameterized(ss.selectPreparedStatementCtx, nil, true), ss.insertionContainers, ss.selectPreparedStatementCtx.GetNonControlColumns(), ss.drmCfg, ss.stream)
	}
	graph := ss.graph
	selectNode := graph.CreatePrimitiveNode(primitive.NewLocalPrimitive(selectEx))
	ss.root = selectNode

	return nil
}
