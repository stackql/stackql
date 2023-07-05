package primitivebuilder //nolint:dupl // TODO: fix

import (
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
)

type DependentMultipleAcquireAndSelect struct {
	graph           primitivegraph.PrimitiveGraphHolder
	acquireBuilders []Builder
	selectBuilder   Builder
}

func NewDependentMultipleAcquireAndSelect(
	graph primitivegraph.PrimitiveGraphHolder,
	acquireBuilders []Builder,
	selectBuilder Builder,
) Builder {
	return &DependentMultipleAcquireAndSelect{
		graph:           graph,
		acquireBuilders: acquireBuilders,
		selectBuilder:   selectBuilder,
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
