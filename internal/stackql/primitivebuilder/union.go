package primitivebuilder

import (
	"github.com/stackql/stackql/internal/stackql/data_staging/output_data_staging"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/streaming"
)

type Union struct {
	graph      primitivegraph.PrimitiveGraph
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
		outputter := output_data_staging.NewNaiveOutputter(
			output_data_staging.NewNaivePacketPreparator(
				output_data_staging.NewNaiveSource(
					un.handlerCtx.GetSQLEngine(),
					us,
					un.drmCfg,
				),
				un.unionCtx.GetNonControlColumns(),
				streaming.NewNopMapStream(),
				un.drmCfg,
			),
			un.unionCtx.GetNonControlColumns(),
		)
		return outputter.OutputExecutorResult()
	}
	graph := un.graph
	unionNode := graph.CreatePrimitiveNode(primitive.NewLocalPrimitive(unionEx))
	un.root = unionNode
	un.tail = unionNode
	return nil
}

func NewUnion(graph primitivegraph.PrimitiveGraph, handlerCtx handler.HandlerContext, unionCtx drm.PreparedStatementCtx) Builder {
	return &Union{
		graph:      graph,
		handlerCtx: handlerCtx,
		drmCfg:     handlerCtx.GetDrmConfig(),
		unionCtx:   unionCtx,
	}
}

func (ss *Union) GetRoot() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *Union) GetTail() primitivegraph.PrimitiveNode {
	return ss.tail
}
