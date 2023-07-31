package primitivegraph

import (
	"github.com/stackql/stackql/internal/stackql/acid/operation"
	"github.com/stackql/stackql/internal/stackql/primitive"
)

var (
	_ PrimitiveGraph       = (*standardPrimitiveGraph)(nil)
	_ primitive.IPrimitive = (*standardPrimitiveGraph)(nil)
)

type standardPrimitiveGraph struct {
	standardBasePrimitiveGraph
}

func newPrimitiveGraph(concurrencyLimit int) PrimitiveGraph {
	baseGraph := newBasePrimitiveGraph(concurrencyLimit)
	return &standardPrimitiveGraph{
		standardBasePrimitiveGraph: baseGraph,
	}
}

func (pg *standardPrimitiveGraph) CreatePrimitiveNode(pr primitive.IPrimitive) PrimitiveNode {
	nn := pg.g.NewNode()
	node := &standardPrimitiveNode{
		op:     operation.NewReversibleOperation(pr),
		id:     nn.ID(),
		isDone: make(chan bool, 1),
	}
	pg.g.AddNode(node)
	return node
}

func (pg *standardPrimitiveGraph) NewDependency(from PrimitiveNode, to PrimitiveNode, weight float64) {
	e := pg.g.NewWeightedEdge(from, to, weight)
	pg.g.SetWeightedEdge(e)
}
