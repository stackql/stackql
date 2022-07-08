package primitivebuilder

import (
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
)

type DependentMultipleAcquireAndSelect struct {
	graph           *primitivegraph.PrimitiveGraph
	acquireBuilders []Builder
	selectBuilder   Builder
}

func NewDependentMultipleAcquireAndSelect(graph *primitivegraph.PrimitiveGraph, acquireBuilders []Builder, selectBuilder Builder) Builder {
	return &DependentMultipleAcquireAndSelect{
		graph:           graph,
		acquireBuilders: acquireBuilders,
		selectBuilder:   selectBuilder,
	}
}

func (ss *DependentMultipleAcquireAndSelect) GetRoot() primitivegraph.PrimitiveNode {
	return ss.acquireBuilders[0].GetRoot()
}

func (ss *DependentMultipleAcquireAndSelect) GetTail() primitivegraph.PrimitiveNode {
	return ss.selectBuilder.GetTail()
}

func (ss *DependentMultipleAcquireAndSelect) Build() error {
	err := ss.selectBuilder.Build()
	if err != nil {
		return err
	}
	for i, acbBld := range ss.acquireBuilders {
		err = acbBld.Build()
		if err != nil {
			return err
		}
		graph := ss.graph
		if i > 0 {
			graph.NewDependency(ss.acquireBuilders[i-1].GetTail(), acbBld.GetRoot(), 1.0)
		}
		if i == len(ss.acquireBuilders)-1 {
			graph.NewDependency(acbBld.GetTail(), ss.selectBuilder.GetRoot(), 1.0)
		}
	}
	return nil
}
