package primitivebuilder

import (
	"github.com/stackql/any-sdk/public/sqlengine"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/util"
)

var (
	defaultNopMessages []string = []string{"OK"} //nolint:revive,gochecknoglobals // prefer declarative
)

type NopBuilder struct {
	graph      primitivegraph.PrimitiveGraphHolder
	handlerCtx handler.HandlerContext
	root       primitivegraph.PrimitiveNode
	sqlEngine  sqlengine.SQLEngine
	messages   []string
}

func NewNopBuilder(
	graph primitivegraph.PrimitiveGraphHolder,
	txnControlCounters internaldto.TxnControlCounters, //nolint:revive // future proofing
	handlerCtx handler.HandlerContext,
	sqlEngine sqlengine.SQLEngine,
	messages []string,
) Builder {
	if len(messages) == 0 {
		messages = defaultNopMessages
	}
	return &NopBuilder{
		graph:      graph,
		handlerCtx: handlerCtx,
		sqlEngine:  sqlEngine,
		messages:   messages,
	}
}

func (nb *NopBuilder) Build() error {
	pr := primitive.NewLocalPrimitive(
		//nolint:revive // no big deal
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			return util.PrepareResultSet(
				internaldto.NewPrepareResultSetPlusRawDTO(
					nil,
					nil,
					nil,
					nil,
					nil,
					internaldto.NewBackendMessages(nb.messages), nil,
					nb.handlerCtx.GetTypingConfig()),
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
