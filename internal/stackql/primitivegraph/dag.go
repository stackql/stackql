package primitivegraph

import (
	"context"
	"fmt"

	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/primitive"

	"gonum.org/v1/gonum/graph"

	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"

	"golang.org/x/sync/errgroup"
)

type PrimitiveGraph interface {
	AddTxnControlCounters(t internaldto.TxnControlCounters)
	ContainsIndirect() bool
	CreatePrimitiveNode(pr primitive.IPrimitive) PrimitiveNode
	Execute(ctx primitive.IPrimitiveCtx) internaldto.ExecutorOutput
	GetInputFromAlias(string) (internaldto.ExecutorOutput, bool)
	IncidentData(fromId int64, input internaldto.ExecutorOutput) error
	GetTxnControlCounterSlice() []internaldto.TxnControlCounters
	NewDependency(from PrimitiveNode, to PrimitiveNode, weight float64)
	Optimise() error
	SetContainsIndirect(containsView bool)
	SetExecutor(func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput) error
	SetInputAlias(alias string, id int64) error
	SetTxnId(id int)
	Sort() (sorted []graph.Node, err error)
}

type standardPrimitiveGraph struct {
	g                      *simple.WeightedDirectedGraph
	sorted                 []graph.Node
	txnControlCounterSlice []internaldto.TxnControlCounters
	errGroup               *errgroup.Group
	errGroupCtx            context.Context
	containsView           bool
}

func (pg *standardPrimitiveGraph) AddTxnControlCounters(t internaldto.TxnControlCounters) {
	pg.txnControlCounterSlice = append(pg.txnControlCounterSlice, t)
}

func (pg *standardPrimitiveGraph) GetTxnControlCounterSlice() []internaldto.TxnControlCounters {
	return pg.txnControlCounterSlice
}

func (pg *standardPrimitiveGraph) SetExecutor(func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput) error {
	return fmt.Errorf("pass through primitive does not support SetExecutor()")
}

func (pg *standardPrimitiveGraph) ContainsIndirect() bool {
	return pg.containsView
}

func (pg *standardPrimitiveGraph) SetContainsIndirect(containsView bool) {
	pg.containsView = containsView
}

func (pr *standardPrimitiveGraph) GetInputFromAlias(string) (internaldto.ExecutorOutput, bool) {
	var rv internaldto.ExecutorOutput
	return rv, false
}

func (pg *standardPrimitiveGraph) Execute(ctx primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
	var output internaldto.ExecutorOutput = internaldto.NewExecutorOutput(nil, nil, nil, nil, fmt.Errorf("empty execution graph"))
	for _, node := range pg.sorted {
		outChan := make(chan internaldto.ExecutorOutput, 1)
		switch node := node.(type) {
		case standardPrimitiveNode:
			pg.errGroup.Go(
				func() error {
					output := node.GetPrimitive().Execute(ctx)
					outChan <- output
					close(outChan)
					return output.Err
				},
			)
			destinationNodes := pg.g.From(node.ID())
			output = <-outChan
			for {
				if !destinationNodes.Next() {
					break
				}
				fromNode := destinationNodes.Node()
				switch fromNode := fromNode.(type) {
				case standardPrimitiveNode:
					fromNode.GetPrimitive().IncidentData(node.ID(), output)
				}
			}
		default:
			internaldto.NewExecutorOutput(nil, nil, nil, nil, fmt.Errorf("unknown execution primitive type: '%T'", node))
		}
	}
	if err := pg.errGroup.Wait(); err != nil {
		return internaldto.NewExecutorOutput(nil, nil, nil, nil, err)
	}
	return output
}

func (pg *standardPrimitiveGraph) SetTxnId(id int) {
	nodes := pg.g.Nodes()
	for {
		if !nodes.Next() {
			return
		}
		node := nodes.Node()
		switch node := node.(type) {
		case standardPrimitiveNode:
			node.GetPrimitive().SetTxnId(id)
		}
	}
}

func (pg *standardPrimitiveGraph) Optimise() error {
	var err error
	pg.sorted, err = topo.Sort(pg.g)
	return err
}

func (pg *standardPrimitiveGraph) IncidentData(fromId int64, input internaldto.ExecutorOutput) error {
	return nil
}

func (pr *standardPrimitiveGraph) SetInputAlias(alias string, id int64) error {
	return nil
}

func (g *standardPrimitiveGraph) Sort() (sorted []graph.Node, err error) {
	return topo.Sort(g.g)
}

func SortPlan(g PrimitiveGraph) (sorted []graph.Node, err error) {
	return g.Sort()
}

type PrimitiveNode interface {
	GetPrimitive() primitive.IPrimitive
	ID() int64
	SetInputAlias(alias string, id int64) error
}

type standardPrimitiveNode struct {
	primitive primitive.IPrimitive
	id        int64
}

func (pg *standardPrimitiveGraph) CreatePrimitiveNode(pr primitive.IPrimitive) PrimitiveNode {
	nn := pg.g.NewNode()
	node := standardPrimitiveNode{
		primitive: pr,
		id:        nn.ID(),
	}
	pg.g.AddNode(node)
	return node
}

func (pn standardPrimitiveNode) ID() int64 {
	return pn.id
}

func (pn standardPrimitiveNode) GetPrimitive() primitive.IPrimitive {
	return pn.primitive
}

func (pn standardPrimitiveNode) SetInputAlias(alias string, id int64) error {
	return pn.GetPrimitive().SetInputAlias(alias, id)
}

func NewPrimitiveGraph(concurrencyLimit int) PrimitiveGraph {
	eg, egCtx := errgroup.WithContext(context.Background())
	eg.SetLimit(concurrencyLimit)
	return &standardPrimitiveGraph{
		g:           simple.NewWeightedDirectedGraph(0.0, 0.0),
		errGroup:    eg,
		errGroupCtx: egCtx,
	}
}

func (pg *standardPrimitiveGraph) NewDependency(from PrimitiveNode, to PrimitiveNode, weight float64) {
	e := pg.g.NewWeightedEdge(from, to, weight)
	pg.g.SetWeightedEdge(e)
}
