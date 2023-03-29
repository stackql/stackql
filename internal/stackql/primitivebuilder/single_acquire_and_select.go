package primitivebuilder

import (
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/streaming"
	"github.com/stackql/stackql/internal/stackql/tableinsertioncontainer"
)

type SingleAcquireAndSelect struct {
	graph          primitivegraph.PrimitiveGraph
	acquireBuilder Builder
	selectBuilder  Builder
}

func NewSingleAcquireAndSelect(
	graph primitivegraph.PrimitiveGraph,
	txnControlCounters internaldto.TxnControlCounters, //nolint:revive // future proofing
	handlerCtx handler.HandlerContext,
	insertContainer tableinsertioncontainer.TableInsertionContainer,
	insertCtx drm.PreparedStatementCtx,
	selectCtx drm.PreparedStatementCtx,
	rowSort func(map[string]map[string]interface{}) []string,
) Builder {
	return &SingleAcquireAndSelect{
		graph: graph,
		acquireBuilder: NewSingleSelectAcquire(
			graph,
			handlerCtx,
			insertContainer,
			insertCtx,
			rowSort,
			nil),
		selectBuilder: NewSingleSelect(
			graph, handlerCtx, selectCtx,
			[]tableinsertioncontainer.TableInsertionContainer{insertContainer},
			rowSort,
			streaming.NewNopMapStream()),
	}
}

func (ss *SingleAcquireAndSelect) GetRoot() primitivegraph.PrimitiveNode {
	return ss.acquireBuilder.GetRoot()
}

func (ss *SingleAcquireAndSelect) GetTail() primitivegraph.PrimitiveNode {
	return ss.selectBuilder.GetTail()
}

func (ss *SingleAcquireAndSelect) Build() error {
	err := ss.acquireBuilder.Build()
	if err != nil {
		return err
	}
	err = ss.selectBuilder.Build()
	if err != nil {
		return err
	}
	graph := ss.graph
	graph.NewDependency(ss.acquireBuilder.GetTail(), ss.selectBuilder.GetRoot(), 1.0)
	return nil
}
