package primitivegraph

import (
	"context"
	"fmt"

	"github.com/stackql/stackql/internal/stackql/acid/binlog"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/primitive"

	"gonum.org/v1/gonum/graph"

	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"

	"golang.org/x/sync/errgroup"
)

var (
	_ BasePrimitiveGraph = (*standardBasePrimitiveGraph)(nil)
)

type standardBasePrimitiveGraph struct {
	g                      *simple.WeightedDirectedGraph
	sorted                 []graph.Node
	txnControlCounterSlice []internaldto.TxnControlCounters
	errGroup               *errgroup.Group
	errGroupCtx            context.Context
	containsView           bool
}

func (pg *standardBasePrimitiveGraph) Size() int {
	return pg.g.Nodes().Len()
}

func (pg *standardBasePrimitiveGraph) IsReadOnly() bool {
	nodes := pg.g.Nodes()
	for nodes.Next() {
		node := nodes.Node()
		primNode, isPrimNode := node.(PrimitiveNode)
		if !isPrimNode {
			continue
		}
		if !primNode.GetOperation().IsReadOnly() {
			return false
		}
	}
	return true
}

func (pg *standardBasePrimitiveGraph) SetRedoLog(binlog.LogEntry) {
}

func (pg *standardBasePrimitiveGraph) SetUndoLog(binlog.LogEntry) {
}

func (pg *standardBasePrimitiveGraph) GetRedoLog() (binlog.LogEntry, bool) {
	return nil, false
}

func (pg *standardBasePrimitiveGraph) GetUndoLog() (binlog.LogEntry, bool) {
	rv := binlog.NewSimpleLogEntry(nil, nil)
	for _, node := range pg.sorted {
		primNode, isPrimNode := node.(PrimitiveNode)
		if !isPrimNode {
			continue
		}
		op := primNode.GetOperation()
		undoLog, undoLogExists := op.GetUndoLog()
		if undoLogExists && undoLog != nil {
			rv.AppendRaw(undoLog.GetRaw())
			for _, h := range undoLog.GetHumanReadable() {
				rv.AppendHumanReadable(h)
			}
		}
	}
	return nil, false
}

func (pg *standardBasePrimitiveGraph) AddTxnControlCounters(t internaldto.TxnControlCounters) {
	pg.txnControlCounterSlice = append(pg.txnControlCounterSlice, t)
}

func (pg *standardBasePrimitiveGraph) GetTxnControlCounterSlice() []internaldto.TxnControlCounters {
	return pg.txnControlCounterSlice
}

func (pg *standardBasePrimitiveGraph) SetExecutor(func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput) error {
	return fmt.Errorf("pass through primitive does not support SetExecutor()")
}

func (pg *standardBasePrimitiveGraph) ContainsIndirect() bool {
	return pg.containsView
}

func (pg *standardBasePrimitiveGraph) SetContainsIndirect(containsView bool) {
	pg.containsView = containsView
}

func (pg *standardBasePrimitiveGraph) GetInputFromAlias(string) (internaldto.ExecutorOutput, bool) {
	var rv internaldto.ExecutorOutput
	return rv, false
}

// After each query execution, the graph needs to be reset.
// This is so that cached queries can be re-executed.
func (pg *standardBasePrimitiveGraph) reset() {
	for _, node := range pg.sorted {
		switch node := node.(type) { //nolint:gocritic // acceptable
		case PrimitiveNode:
			select {
			case <-node.IsDone():
			default:
			}
		}
	}
}

// Execute() is the entry point for the execution of the graph.
// It is responsible for executing the graph in a topological order.
// This particular implementation:
//   - Uses the errgroup package to execute the graph in parallel.
//   - Blocks on any node that has a dependency that has not been executed.
//
//nolint:gocognit // inherent complexity
func (pg *standardBasePrimitiveGraph) Execute(ctx primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
	// Reset the graph.
	// Absolutely necessary for re-execution
	defer pg.reset()
	//nolint:stylecheck // prefer declarative
	var output internaldto.ExecutorOutput = internaldto.NewExecutorOutput(
		nil, nil, nil, nil, fmt.Errorf("empty execution graph"))
	for _, node := range pg.sorted {
		outChan := make(chan internaldto.ExecutorOutput, 1)
		switch node := node.(type) {
		case PrimitiveNode:
			incidentNodes := pg.g.To(node.ID())
			for {
				hasNext := incidentNodes.Next()
				if !hasNext {
					break
				}
				incidentNode := incidentNodes.Node()
				switch incidentNode := incidentNode.(type) {
				case PrimitiveNode:
					// await completion of the incident node
					// and replenish the IsDone() channel
					incidentNode.SetIsDone(<-incidentNode.IsDone())
				default:
					return internaldto.NewExecutorOutput(
						nil, nil, nil, nil,
						fmt.Errorf("unknown execution primitive type: '%T'", incidentNode))
				}
			}
			pg.errGroup.Go(
				func() error {
					output := node.GetOperation().Execute(ctx) //nolint:govet // intentional
					outChan <- output
					close(outChan)
					return output.GetError()
				},
			)
			destinationNodes := pg.g.From(node.ID())
			output = <-outChan
			for {
				if !destinationNodes.Next() {
					break
				}
				fromNode := destinationNodes.Node()
				switch fromNode := fromNode.(type) { //nolint:gocritic // acceptable
				case PrimitiveNode:
					op := fromNode.GetOperation()
					op.IncidentData(node.ID(), output) //nolint:errcheck // TODO: consider design options
				}
			}
			node.SetIsDone(true)
		default:
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, fmt.Errorf("unknown execution primitive type: '%T'", node))
		}
	}
	if err := pg.errGroup.Wait(); err != nil {
		undoLog, _ := output.GetUndoLog()
		return internaldto.NewExecutorOutput(nil, nil, nil, nil, err).WithUndoLog(undoLog)
	}
	return output
}

func (pg *standardBasePrimitiveGraph) SetTxnID(id int) {
	nodes := pg.g.Nodes()
	for {
		if !nodes.Next() {
			return
		}
		node := nodes.Node()
		switch node := node.(type) { //nolint:gocritic // acceptable
		case PrimitiveNode:
			node.GetOperation().SetTxnID(id)
		}
	}
}

func (pg *standardBasePrimitiveGraph) Optimise() error {
	var err error
	pg.sorted, err = topo.Sort(pg.g)
	return err
}

//nolint:revive // future proofing
func (pg *standardBasePrimitiveGraph) IncidentData(fromID int64, input internaldto.ExecutorOutput) error {
	return nil
}

//nolint:revive // future proofing
func (pg *standardBasePrimitiveGraph) SetInputAlias(alias string, id int64) error {
	return nil
}

func (pg *standardBasePrimitiveGraph) Sort() ([]graph.Node, error) {
	return topo.Sort(pg.g)
}

func newBasePrimitiveGraph(concurrencyLimit int) standardBasePrimitiveGraph {
	eg, egCtx := errgroup.WithContext(context.Background())
	eg.SetLimit(concurrencyLimit)
	return standardBasePrimitiveGraph{
		g:           simple.NewWeightedDirectedGraph(0.0, 0.0),
		errGroup:    eg,
		errGroupCtx: egCtx,
	}
}
