package primitivebuilder

import (
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/util"
)

type NopBuilder struct {
	graph      primitivegraph.PrimitiveGraph
	handlerCtx handler.HandlerContext
	root       primitivegraph.PrimitiveNode
	sqlEngine  sqlengine.SQLEngine
}

func NewNopBuilder(
	graph primitivegraph.PrimitiveGraph,
	txnControlCounters internaldto.TxnControlCounters, //nolint:revive // future proofing
	handlerCtx handler.HandlerContext,
	sqlEngine sqlengine.SQLEngine,
) Builder {
	return &NopBuilder{
		graph:      graph,
		handlerCtx: handlerCtx,
		sqlEngine:  sqlEngine,
	}
}

func (nb *NopBuilder) Build() error {
	pr := primitive.NewLocalPrimitive(
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			return util.PrepareResultSet(
				internaldto.NewPrepareResultSetPlusRawDTO(
					nil,
					nil,
					nil,
					nil,
					nil,
					internaldto.NewBackendMessages([]string{"nop completed"}), nil),
			)
		},
	)
	nb.root = nb.graph.CreatePrimitiveNode(pr)
	return nil
}

func (nb *NopBuilder) GetRoot() primitivegraph.PrimitiveNode {
	return nb.root
}

func (nb *NopBuilder) GetTail() primitivegraph.PrimitiveNode {
	return nb.root
}
