package primitivebuilder

import (
	"github.com/stackql/any-sdk/pkg/streaming"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/builder_input"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/tableinsertioncontainer"
)

type SingleAcquireAndSelect struct {
	graph          primitivegraph.PrimitiveGraphHolder
	acquireBuilder Builder
	selectBuilder  Builder
	bldrInput      builder_input.BuilderInput
	root           primitivegraph.PrimitiveNode
}

func NewSingleAcquireAndSelect(
	// graph primitivegraph.PrimitiveGraphHolder,
	// txnControlCounters internaldto.TxnControlCounters, //nolint:revive // future proofing
	// handlerCtx handler.HandlerContext,
	// insertContainer tableinsertioncontainer.TableInsertionContainer,
	bldrInput builder_input.BuilderInput,
	insertCtx drm.PreparedStatementCtx,
	selectCtx drm.PreparedStatementCtx,
	rowSort func(map[string]map[string]interface{}) []string,
) Builder {
	graph, _ := bldrInput.GetGraphHolder()
	// txnControlCounters, _ := bldrInput.GetTxnCtrlCtrs()
	handlerCtx, _ := bldrInput.GetHandlerContext()
	insertContainer, _ := bldrInput.GetTableInsertionContainer()
	return &SingleAcquireAndSelect{
		graph: graph,
		acquireBuilder: NewSingleSelectAcquire(
			graph,
			handlerCtx,
			insertContainer,
			insertCtx,
			rowSort,
			nil,
			bldrInput.IsAwait(),
		),
		selectBuilder: NewSingleSelect(
			graph, handlerCtx, selectCtx,
			[]tableinsertioncontainer.TableInsertionContainer{insertContainer},
			rowSort,
			streaming.NewNopMapStream()),
		bldrInput: bldrInput,
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
	rootNode := ss.acquireBuilder.GetRoot()
	ss.root = rootNode
	dependencyNode, dependencyNodeExists := ss.bldrInput.GetDependencyNode()
	if dependencyNodeExists {
		//nolint:errcheck // TODO: fix this
		rootNode.SetInputAlias("", dependencyNode.ID())
		ss.graph.NewDependency(dependencyNode, rootNode, 1.0)
		// ss.root = dependencyNode // dont think this is needed
	}
	return nil
}
