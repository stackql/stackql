package primitivebuilder //nolint:dupl // TODO: fix

import (
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
)

type DependencySubDAGBuilder struct {
	graph              primitivegraph.PrimitiveGraph
	dependencyBuilders []Builder
	dependentBuilder   Builder
}

func NewDependencySubDAGBuilder(
	graph primitivegraph.PrimitiveGraph,
	dependencyBuilders []Builder,
	dependentBuilder Builder,
) Builder {
	return &DependencySubDAGBuilder{
		graph:              graph,
		dependencyBuilders: dependencyBuilders,
		dependentBuilder:   dependentBuilder,
	}
}

func (ss *DependencySubDAGBuilder) GetRoot() primitivegraph.PrimitiveNode {
	if len(ss.dependencyBuilders) > 0 {
		return ss.dependencyBuilders[0].GetRoot()
	}
	return ss.dependentBuilder.GetRoot()
}

func (ss *DependencySubDAGBuilder) GetTail() primitivegraph.PrimitiveNode {
	return ss.dependentBuilder.GetTail()
}

func (ss *DependencySubDAGBuilder) Build() error {
	err := ss.dependentBuilder.Build()
	if err != nil {
		return err
	}
	for i, acbBld := range ss.dependencyBuilders {
		err = acbBld.Build()
		if err != nil {
			return err
		}
		graph := ss.graph
		if i > 0 {
			graph.NewDependency(ss.dependencyBuilders[i-1].GetTail(), acbBld.GetRoot(), 1.0)
		}
		if i == len(ss.dependencyBuilders)-1 {
			graph.NewDependency(acbBld.GetTail(), ss.dependentBuilder.GetRoot(), 1.0)
		}
	}
	return nil
}

func (ss *DependencySubDAGBuilder) SetWriteOnly(_ bool) {
}

func (ss *DependencySubDAGBuilder) IsWriteOnly() bool {
	return false
}
