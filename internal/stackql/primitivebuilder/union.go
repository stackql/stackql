package primitivebuilder

import (
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internaldto"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/streaming"
)

type Union struct {
	graph      *primitivegraph.PrimitiveGraph
	unionCtx   drm.PreparedStatementCtx
	handlerCtx handler.HandlerContext
	drmCfg     drm.DRMConfig
	lhs        drm.PreparedStatementCtx
	rhs        []drm.PreparedStatementCtx
	root, tail primitivegraph.PrimitiveNode
}

func (un *Union) Build() error {
	unionEx := func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
		us := drm.NewPreparedStatementParameterized(un.unionCtx, nil, false)
		i := 0
		us.AddChild(i, drm.NewPreparedStatementParameterized(un.lhs, nil, true))
		for _, rhsElement := range un.rhs {
			i++
			us.AddChild(i, drm.NewPreparedStatementParameterized(rhsElement, nil, true))
		}
		return prepareGolangResult(un.handlerCtx.GetSQLEngine(), un.handlerCtx.GetOutErrFile(), us, nil, un.lhs.GetNonControlColumns(), un.drmCfg, streaming.NewNopMapStream())
	}
	graph := un.graph
	unionNode := graph.CreatePrimitiveNode(primitive.NewLocalPrimitive(unionEx))
	un.root = unionNode
	return nil
}

func NewUnion(graph *primitivegraph.PrimitiveGraph, handlerCtx handler.HandlerContext, unionCtx drm.PreparedStatementCtx, lhs drm.PreparedStatementCtx, rhs []drm.PreparedStatementCtx) Builder {
	return &Union{
		graph:      graph,
		handlerCtx: handlerCtx,
		drmCfg:     handlerCtx.GetDrmConfig(),
		unionCtx:   unionCtx,
		lhs:        lhs,
		rhs:        rhs,
	}
}

func (ss *Union) GetRoot() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *Union) GetTail() primitivegraph.PrimitiveNode {
	return ss.tail
}
