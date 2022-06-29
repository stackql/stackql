package primitivebuilder

import (
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
)

type MultipleAcquireAndSelect struct {
	graph           *primitivegraph.PrimitiveGraph
	acquireBuilders []Builder
	selectBuilder   Builder
}

func NewMultipleAcquireAndSelect(graph *primitivegraph.PrimitiveGraph, acquireBuilders []Builder, selectBuilder Builder) Builder {
	return &MultipleAcquireAndSelect{
		graph:           graph,
		acquireBuilders: acquireBuilders,
		selectBuilder:   selectBuilder,
	}
}

func (ss *MultipleAcquireAndSelect) GetRoot() primitivegraph.PrimitiveNode {
	return ss.acquireBuilders[0].GetRoot()
}

func (ss *MultipleAcquireAndSelect) GetTail() primitivegraph.PrimitiveNode {
	return ss.selectBuilder.GetTail()
}

func (ss *MultipleAcquireAndSelect) Build() error {
	err := ss.selectBuilder.Build()
	if err != nil {
		return err
	}
	for _, acbBld := range ss.acquireBuilders {
		err = acbBld.Build()
		if err != nil {
			return err
		}
		graph := ss.graph
		graph.NewDependency(acbBld.GetTail(), ss.selectBuilder.GetRoot(), 1.0)
	}
	return nil
}
