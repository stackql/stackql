package primitivebuilder

import (
	"time"

	"github.com/stackql/any-sdk/public/sqlengine"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/util"
)

var (
	defaultConcludeExplainMessages []string = []string{"OK"} //nolint:revive,gochecknoglobals // prefer declarative
)

type ExplainBuilder struct {
	graph          primitivegraph.PrimitiveGraphHolder
	handlerCtx     handler.HandlerContext
	root           primitivegraph.PrimitiveNode
	sqlEngine      sqlengine.SQLEngine
	messages       []string
	instructionErr error
}

func NewExplainBuilder(
	graph primitivegraph.PrimitiveGraphHolder,
	txnControlCounters internaldto.TxnControlCounters, //nolint:revive // future proofing
	handlerCtx handler.HandlerContext,
	sqlEngine sqlengine.SQLEngine,
	messages []string,
	instructionErr error,
) Builder {
	if len(messages) == 0 {
		messages = []string{}
	}
	messages = append(messages, defaultConcludeExplainMessages...)
	return &ExplainBuilder{
		graph:          graph,
		handlerCtx:     handlerCtx,
		sqlEngine:      sqlEngine,
		messages:       messages,
		instructionErr: instructionErr,
	}
}

func (nb *ExplainBuilder) Build() error {
	pr := primitive.NewLocalPrimitive(
		//nolint:revive // no big deal
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			if nb.instructionErr != nil {
				return internaldto.NewErroneousExecutorOutput(nb.instructionErr)
			}
			oneLinerOutput := time.Now().Format("2006-01-02T15:04:05-07:00 MST")
			resultMap := map[string]any{
				"query":     nb.handlerCtx.GetQuery(),
				"timestamp": oneLinerOutput,
				"result":    "OK",
				"valid":     true,
			}
			columns := []string{"query", "result", "timestamp", "valid"}
			return util.PrepareResultSet(
				internaldto.NewPrepareResultSetPlusRawDTO(
					nil,
					map[string]map[string]any{"0": resultMap},
					columns,
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

func (nb *ExplainBuilder) GetRoot() primitivegraph.PrimitiveNode {
	return nb.root
}

func (nb *ExplainBuilder) GetTail() primitivegraph.PrimitiveNode {
	return nb.root
}
