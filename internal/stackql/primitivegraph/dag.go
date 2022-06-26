package primitivegraph

import (
	"context"
	"fmt"

	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/primitive"

	"gonum.org/v1/gonum/graph"

	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"

	"golang.org/x/sync/errgroup"
)

type PrimitiveGraph struct {
	g                      *simple.WeightedDirectedGraph
	sorted                 []graph.Node
	txnControlCounterSlice []dto.TxnControlCounters
	errGroup               *errgroup.Group
	errGroupCtx            context.Context
}

func (pg *PrimitiveGraph) AddTxnControlCounters(t dto.TxnControlCounters) {
	pg.txnControlCounterSlice = append(pg.txnControlCounterSlice, t)
}

func (pg *PrimitiveGraph) GetTxnControlCounterSlice() []dto.TxnControlCounters {
	return pg.txnControlCounterSlice
}

func (pg *PrimitiveGraph) SetExecutor(func(pc primitive.IPrimitiveCtx) dto.ExecutorOutput) error {
	return fmt.Errorf("pass through primitive does not support SetExecutor()")
}

func (pr *PrimitiveGraph) GetInputFromAlias(string) (dto.ExecutorOutput, bool) {
	var rv dto.ExecutorOutput
	return rv, false
}

func (pg *PrimitiveGraph) Execute(ctx primitive.IPrimitiveCtx) dto.ExecutorOutput {
	var output dto.ExecutorOutput = dto.NewExecutorOutput(nil, nil, nil, nil, fmt.Errorf("empty execution graph"))
	for _, node := range pg.sorted {
		outChan := make(chan dto.ExecutorOutput, 1)
		switch node := node.(type) {
		case PrimitiveNode:
			pg.errGroup.Go(
				func() error {
					output := node.Primitive.Execute(ctx)
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
				case PrimitiveNode:
					fromNode.Primitive.IncidentData(node.ID(), output)
				}
			}
		default:
			dto.NewExecutorOutput(nil, nil, nil, nil, fmt.Errorf("unknown execution primitive type: '%T'", node))
		}
	}
	if err := pg.errGroup.Wait(); err != nil {
		return dto.NewExecutorOutput(nil, nil, nil, nil, err)
	}
	return output
}

func (pg *PrimitiveGraph) GetPreparedStatementContext() *drm.PreparedStatementCtx {
	return nil
}

func (pg *PrimitiveGraph) SetTxnId(id int) {
	nodes := pg.g.Nodes()
	for {
		if !nodes.Next() {
			return
		}
		node := nodes.Node()
		switch node := node.(type) {
		case PrimitiveNode:
			node.Primitive.SetTxnId(id)
		}
	}
}

func (pg *PrimitiveGraph) Optimise() error {
	var err error
	pg.sorted, err = topo.Sort(pg.g)
	return err
}

func (pg *PrimitiveGraph) IncidentData(fromId int64, input dto.ExecutorOutput) error {
	return nil
}

func (pr *PrimitiveGraph) SetInputAlias(alias string, id int64) error {
	return nil
}

func SortPlan(g *PrimitiveGraph) (sorted []graph.Node, err error) {
	return topo.Sort(g.g)
}

type PrimitiveNode struct {
	Primitive primitive.IPrimitive
	id        int64
}

func (pg *PrimitiveGraph) CreatePrimitiveNode(pr primitive.IPrimitive) PrimitiveNode {
	nn := pg.g.NewNode()
	node := PrimitiveNode{
		Primitive: pr,
		id:        nn.ID(),
	}
	pg.g.AddNode(node)
	return node
}

func (pn PrimitiveNode) ID() int64 {
	return pn.id
}

func (pn PrimitiveNode) SetInputAlias(alias string, id int64) error {
	return pn.Primitive.SetInputAlias(alias, id)
}

func NewPrimitiveGraph(concurrencyLimit int) *PrimitiveGraph {
	eg, egCtx := errgroup.WithContext(context.Background())
	eg.SetLimit(concurrencyLimit)
	return &PrimitiveGraph{
		g:           simple.NewWeightedDirectedGraph(0.0, 0.0),
		errGroup:    eg,
		errGroupCtx: egCtx,
	}
}

func (pg *PrimitiveGraph) NewDependency(from PrimitiveNode, to PrimitiveNode, weight float64) {
	e := pg.g.NewWeightedEdge(from, to, weight)
	pg.g.SetWeightedEdge(e)
}
