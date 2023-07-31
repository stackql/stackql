package primitivebuilder

import (
	"github.com/stackql/stackql/internal/stackql/dataflow"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/util"
)

type DataflowGraphBuilder struct {
	graph         primitivegraph.PrimitiveGraphHolder
	dataflowGraph dataflow.WeaklyConnectedComponent
	handlerCtx    handler.HandlerContext
	root          primitivegraph.PrimitiveNode
	sqlEngine     sqlengine.SQLEngine
}

func NewDataflowGraphBuilder(
	graph primitivegraph.PrimitiveGraphHolder,
	dataflowGraph dataflow.WeaklyConnectedComponent,
	txnControlCounters *internaldto.TxnControlCounters, //nolint:revive // future proofing
	handlerCtx handler.HandlerContext,
	sqlEngine sqlengine.SQLEngine,
) Builder {
	return &DataflowGraphBuilder{
		graph:         graph,
		dataflowGraph: dataflowGraph,
		handlerCtx:    handlerCtx,
		sqlEngine:     sqlEngine,
	}
}

func (nb *DataflowGraphBuilder) Build() error {
	pr := primitive.NewLocalPrimitive(
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			return util.PrepareResultSet(
				internaldto.NewPrepareResultSetPlusRawDTO(
					nil,
					nil,
					nil,
					nil,
					nil,
					internaldto.NewBackendMessages([]string{"nop completed"}), nil,
					nb.handlerCtx.GetTypingConfig()),
			)
		},
	)
	nb.root = nb.graph.CreatePrimitiveNode(pr)
	return nil
}

func (nb *DataflowGraphBuilder) GetRoot() primitivegraph.PrimitiveNode {
	return nb.root
}

func (nb *DataflowGraphBuilder) GetTail() primitivegraph.PrimitiveNode {
	return nb.root
}
