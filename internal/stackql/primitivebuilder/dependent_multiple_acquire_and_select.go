package primitivebuilder

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/primitivegraph"
)

type DependentMultipleAcquireAndSelect struct {
	graph           primitivegraph.PrimitiveGraphHolder
	acquireBuilders []Builder
	selectBuilder   Builder
	dataflowToEdges map[int][]int
}

func NewDependentMultipleAcquireAndSelect(
	graph primitivegraph.PrimitiveGraphHolder,
	acquireBuilders []Builder,
	selectBuilder Builder,
	dataflowToEdges map[int][]int,
) Builder {
	return &DependentMultipleAcquireAndSelect{
		graph:           graph,
		acquireBuilders: acquireBuilders,
		selectBuilder:   selectBuilder,
		dataflowToEdges: dataflowToEdges,
	}
}

// Cache queries may not have acquire builders.
func (ss *DependentMultipleAcquireAndSelect) GetRoot() primitivegraph.PrimitiveNode {
	if len(ss.acquireBuilders) > 0 {
		return ss.acquireBuilders[0].GetRoot()
	}
	return ss.selectBuilder.GetRoot()
}

func (ss *DependentMultipleAcquireAndSelect) GetTail() primitivegraph.PrimitiveNode {
	return ss.selectBuilder.GetTail()
}

func (ss *DependentMultipleAcquireAndSelect) Build() error {
	err := ss.selectBuilder.Build()
	if err != nil {
		return err
	}
	tails := make(map[int]primitivegraph.PrimitiveNode)
	for i, acbBld := range ss.acquireBuilders {
		err = acbBld.Build()
		if err != nil {
			return err
		}
		tail := acbBld.GetTail()
		tails[i] = tail
		graph := ss.graph
		graph.NewDependency(tail, ss.selectBuilder.GetRoot(), 1.0)
		toEdges := ss.dataflowToEdges[i]
		for _, toEdge := range toEdges {
			predecssorTail, ok := tails[toEdge]
			if !ok {
				return fmt.Errorf("unknown predecessor tail")
			}
			graph.NewDependency(predecssorTail, acbBld.GetRoot(), 1.0)
		}
	}
	return nil
}
