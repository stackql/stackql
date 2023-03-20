package primitivebuilder

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/data_staging/output_data_staging"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/streaming"
	"github.com/stackql/stackql/internal/stackql/tableinsertioncontainer"
)

type SingleSelect struct {
	graph                      primitivegraph.PrimitiveGraph
	handlerCtx                 handler.HandlerContext
	drmCfg                     drm.Config
	selectPreparedStatementCtx drm.PreparedStatementCtx
	insertionContainers        []tableinsertioncontainer.TableInsertionContainer
	txnCtrlCtr                 internaldto.TxnControlCounters
	rowSort                    func(map[string]map[string]interface{}) []string
	root                       primitivegraph.PrimitiveNode
	stream                     streaming.MapStream
}

func NewSingleSelect(
	graph primitivegraph.PrimitiveGraph,
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
		logging.GetLogger().Infoln(
			fmt.Sprintf(
				"running select with control parameters: %v",
				ss.selectPreparedStatementCtx.GetGCCtrlCtrs(),
			),
		)

		outputter := output_data_staging.NewNaiveOutputter(
			output_data_staging.NewNaivePacketPreparator(
				output_data_staging.NewNaiveSource(
					ss.handlerCtx.GetSQLEngine(),
					drm.NewPreparedStatementParameterized(ss.selectPreparedStatementCtx, nil, true),
					ss.drmCfg,
				),
				ss.selectPreparedStatementCtx.GetNonControlColumns(),
				ss.stream,
				ss.drmCfg,
			),
			ss.selectPreparedStatementCtx.GetNonControlColumns(),
		)
		return outputter.OutputExecutorResult()
	}
	graph := ss.graph
	selectNode := graph.CreatePrimitiveNode(primitive.NewLocalPrimitive(selectEx))
	ss.root = selectNode

	return nil
}
