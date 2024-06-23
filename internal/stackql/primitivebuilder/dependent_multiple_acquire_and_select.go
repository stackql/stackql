package primitivebuilder

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/sql_system"
)

type DependentMultipleAcquireAndSelect struct {
	graph           primitivegraph.PrimitiveGraphHolder
	acquireBuilders []Builder
	selectBuilder   Builder
	dataflowToEdges map[int][]int
	sqlSystem       sql_system.SQLSystem
	root            primitivegraph.PrimitiveNode
}

func NewDependentMultipleAcquireAndSelect(
	graph primitivegraph.PrimitiveGraphHolder,
	acquireBuilders []Builder,
	selectBuilder Builder,
	dataflowToEdges map[int][]int,
	sqlSystem sql_system.SQLSystem,
) Builder {
	return &DependentMultipleAcquireAndSelect{
		graph:           graph,
		acquireBuilders: acquireBuilders,
		selectBuilder:   selectBuilder,
		dataflowToEdges: dataflowToEdges,
		sqlSystem:       sqlSystem,
	}
}

// Cache queries may not have acquire builders.
func (ss *DependentMultipleAcquireAndSelect) GetRoot() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *DependentMultipleAcquireAndSelect) GetTail() primitivegraph.PrimitiveNode {
	return ss.selectBuilder.GetTail()
}

func (ss *DependentMultipleAcquireAndSelect) Build() error {
	ss.root = ss.graph.CreatePrimitiveNode(
		primitive.NewPassThroughPrimitive(
			ss.sqlSystem,
			ss.graph.GetTxnControlCounterSlice(),
			false,
		),
	)
	err := ss.selectBuilder.Build()
	if err != nil {
		return err
	}
	ss.graph.NewDependency(ss.root, ss.selectBuilder.GetRoot(), 1.0)
	tails := make(map[int]primitivegraph.PrimitiveNode)
	for i, acbBld := range ss.acquireBuilders {
		err = acbBld.Build()
		if err != nil {
			return err
		}
		tail := acbBld.GetTail()
		tails[i] = tail
		graph := ss.graph
		graph.NewDependency(ss.root, acbBld.GetRoot(), 1.0)
		graph.NewDependency(tail, ss.selectBuilder.GetRoot(), 1.0)
		toEdges := ss.dataflowToEdges[i]
		for _, toEdge := range toEdges {
			predecessorTail, ok := tails[toEdge]
			if !ok {
				return fmt.Errorf("unknown predecessor tail")
			}
			graph.NewDependency(predecessorTail, acbBld.GetRoot(), 1.0)
		}
	}
	return nil
}
