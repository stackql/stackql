package primitivebuilder

import (
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
)

type SingleAcquireAndSelect struct {
	graph          *primitivegraph.PrimitiveGraph
	acquireBuilder Builder
	selectBuilder  Builder
}

func NewSingleAcquireAndSelect(graph *primitivegraph.PrimitiveGraph, txnControlCounters *dto.TxnControlCounters, handlerCtx *handler.HandlerContext, tableMeta *taxonomy.ExtendedTableMetadata, insertCtx *drm.PreparedStatementCtx, selectCtx *drm.PreparedStatementCtx, rowSort func(map[string]map[string]interface{}) []string) Builder {
	return &SingleAcquireAndSelect{
		graph:          graph,
		acquireBuilder: NewSingleSelectAcquire(graph, handlerCtx, tableMeta, insertCtx, rowSort, nil),
		selectBuilder:  NewSingleSelect(graph, handlerCtx, selectCtx, rowSort),
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
