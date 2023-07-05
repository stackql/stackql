package primitivegraph

import (
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/primitive"

	"gonum.org/v1/gonum/graph"
)

type BasePrimitiveGraph interface {
	primitive.IPrimitive
	AddTxnControlCounters(t internaldto.TxnControlCounters)
	ContainsIndirect() bool
	Execute(ctx primitive.IPrimitiveCtx) internaldto.ExecutorOutput
	GetInputFromAlias(string) (internaldto.ExecutorOutput, bool)
	IncidentData(fromID int64, input internaldto.ExecutorOutput) error
	GetTxnControlCounterSlice() []internaldto.TxnControlCounters
	Optimise() error
	SetContainsIndirect(containsView bool)
	SetExecutor(func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput) error
	SetInputAlias(alias string, id int64) error
	SetTxnID(id int)
	Sort() (sorted []graph.Node, err error)
	Size() int
}

type PrimitiveGraph interface {
	primitive.IPrimitive
	BasePrimitiveGraph
	CreatePrimitiveNode(pr primitive.IPrimitive) PrimitiveNode
	NewDependency(from PrimitiveNode, to PrimitiveNode, weight float64)
}

func SortPlan(pg PrimitiveGraph) ([]graph.Node, error) {
	return pg.Sort()
}
