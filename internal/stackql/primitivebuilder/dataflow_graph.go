package primitivebuilder

import (
	"github.com/stackql/stackql/internal/stackql/dataflow"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internaldto"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/util"
)

type DataflowGraphBuilder struct {
	graph         primitivegraph.PrimitiveGraph
	dataflowGraph dataflow.DataFlowWeaklyConnectedComponent
	handlerCtx    handler.HandlerContext
	root          primitivegraph.PrimitiveNode
	sqlEngine     sqlengine.SQLEngine
}

func NewDataflowGraphBuilder(
	graph primitivegraph.PrimitiveGraph,
	dataflowGraph dataflow.DataFlowWeaklyConnectedComponent,
	txnControlCounters *internaldto.TxnControlCounters,
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
					&internaldto.BackendMessages{WorkingMessages: []string{"nop completed"}}, nil),
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
