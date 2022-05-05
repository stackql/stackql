package primitivebuilder

import (
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/util"
)

func NewNopBuilder(graph *primitivegraph.PrimitiveGraph, txnControlCounters *dto.TxnControlCounters, handlerCtx *handler.HandlerContext, sqlEngine sqlengine.SQLEngine) Builder {
	return &NopBuilder{
		graph:      graph,
		handlerCtx: handlerCtx,
		sqlEngine:  sqlEngine,
	}
}

func (nb *NopBuilder) Build() error {

	pr := NewLocalPrimitive(
		func(pc primitive.IPrimitiveCtx) dto.ExecutorOutput {
			return util.PrepareResultSet(
				dto.NewPrepareResultSetPlusRawDTO(
					nil,
					nil,
					nil,
					nil,
					nil,
					&dto.BackendMessages{WorkingMessages: []string{"nop completed"}}, nil),
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
