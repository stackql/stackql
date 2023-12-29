package primitivebuilder

import (
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
)

type DependencySubDAGBuilder struct {
	graph              primitivegraph.PrimitiveGraphHolder
	dependencyBuilders []Builder
	dependentBuilder   Builder
}

func NewDependencySubDAGBuilder(
	graph primitivegraph.PrimitiveGraphHolder,
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
	for _, db := range ss.dependencyBuilders {
		acbBld := db
		err = acbBld.Build()
		if err != nil {
			return err
		}
		graph := ss.graph
		graph.NewDependency(acbBld.GetTail(), ss.dependentBuilder.GetRoot(), 1.0)
	}
	return nil
}
