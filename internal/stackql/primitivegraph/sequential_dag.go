package primitivegraph

import (
	"sync"

	"github.com/stackql/stackql/internal/stackql/acid/operation"
	"github.com/stackql/stackql/internal/stackql/primitive"
)

var (
	_ PrimitiveGraph       = (*sequentialPrimitiveGraph)(nil)
	_ primitive.IPrimitive = (*sequentialPrimitiveGraph)(nil)
)

type sequentialPrimitiveGraph struct {
	standardBasePrimitiveGraph
	mutex sync.Mutex
	root  PrimitiveNode
	tail  PrimitiveNode
}

func (pg *sequentialPrimitiveGraph) CreatePrimitiveNode(pr primitive.IPrimitive) PrimitiveNode {
	pg.mutex.Lock()
	defer pg.mutex.Unlock()
	nn := pg.g.NewNode()
	node := &standardPrimitiveNode{
		op:     operation.NewReversibleOperation(pr),
		id:     nn.ID(),
		isDone: make(chan bool, 1),
	}
	var isRoot bool
	if pg.g.Nodes().Len() == 0 {
		pg.root = node
		isRoot = true
	}
	existingTail := pg.tail
	pg.tail = node
	pg.g.AddNode(node)
	if !isRoot {
		pg.NewDependency(existingTail, node, 1.0)
	}
	return node
}

func newSequentialPrimitiveGraph(concurrencyLimit int) PrimitiveGraph {
	baseGraph := newBasePrimitiveGraph(concurrencyLimit)
	return &sequentialPrimitiveGraph{
		standardBasePrimitiveGraph: baseGraph,
	}
}

func (pg *sequentialPrimitiveGraph) NewDependency(from PrimitiveNode, to PrimitiveNode, weight float64) {
	e := pg.g.NewWeightedEdge(from, to, weight)
	pg.g.SetWeightedEdge(e)
}
