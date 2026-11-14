package primitivegraph

import (
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/primitive"
)

var (
	_ PrimitiveGraphHolder = (*standardPrimitiveGraphHolder)(nil)
)

//nolint:revive // acceptable nomenclature
type PrimitiveGraphHolder interface {
	Blank() error
	AddTxnControlCounters(t internaldto.TxnControlCounters)
	ContainsIndirect() bool
	CreatePrimitiveNode(pr primitive.IPrimitive) PrimitiveNode
	GetPrimitiveGraph() PrimitiveGraph
	GetTxnControlCounterSlice() []internaldto.TxnControlCounters
	NewDependency(from PrimitiveNode, to PrimitiveNode, weight float64)
	SetContainsIndirect(bool)
	SetTxnID(int)
	SetContainsUserManagedRelation(containsUserRelation bool)
	ContainsUserManagedRelation() bool
	SetExecutorOutput(id string, output internaldto.ExecutorOutput) error
	GetExecutorOutput(id string) (internaldto.ExecutorOutput, bool)
	GetRequest(requestType string) (internaldto.ExecutorOutputRequest, error)
}

type standardPrimitiveGraphHolder struct {
	concurrencyLimit int
	pg               PrimitiveGraph
	outputRegister   internaldto.ExecutorOutputRegister
}

func (pgh *standardPrimitiveGraphHolder) GetPrimitiveGraph() PrimitiveGraph {
	return pgh.pg
}

func (pgh *standardPrimitiveGraphHolder) SetTxnID(txnID int) {
	pgh.pg.SetTxnID(txnID)
}

func (pgh *standardPrimitiveGraphHolder) AddTxnControlCounters(t internaldto.TxnControlCounters) {
	pgh.pg.AddTxnControlCounters(t)
}

func (pgh *standardPrimitiveGraphHolder) CreatePrimitiveNode(pr primitive.IPrimitive) PrimitiveNode {
	return pgh.pg.CreatePrimitiveNode(pr)
}

func (pgh *standardPrimitiveGraphHolder) NewDependency(from PrimitiveNode, to PrimitiveNode, weight float64) {
	pgh.pg.NewDependency(from, to, weight)
}

func (pgh *standardPrimitiveGraphHolder) SetContainsIndirect(containsView bool) {
	pgh.pg.SetContainsIndirect(containsView)
}

func (pgh *standardPrimitiveGraphHolder) SetContainsUserManagedRelation(containsUserRelation bool) {
	pgh.pg.SetContainsUserManagedRelation(containsUserRelation)
}

func (pgh *standardPrimitiveGraphHolder) ContainsIndirect() bool {
	return pgh.pg.ContainsIndirect()
}

func (pgh *standardPrimitiveGraphHolder) ContainsUserManagedRelation() bool {
	return pgh.pg.ContainsUserManagedRelation()
}

func (pgh *standardPrimitiveGraphHolder) GetTxnControlCounterSlice() []internaldto.TxnControlCounters {
	return pgh.pg.GetTxnControlCounterSlice()
}

func (pgh *standardPrimitiveGraphHolder) SetExecutorOutput(id string, output internaldto.ExecutorOutput) error {
	return pgh.outputRegister.SetExecutorOutput(id, output)
}

func (pgh *standardPrimitiveGraphHolder) GetExecutorOutput(id string) (internaldto.ExecutorOutput, bool) {
	return pgh.outputRegister.GetExecutorOutput(id)
}

func (pgh *standardPrimitiveGraphHolder) GetRequest(requestType string) (internaldto.ExecutorOutputRequest, error) {
	return pgh.outputRegister.GetRequest(requestType)
}

func NewPrimitiveGraphHolder(concurrencyLimit int) PrimitiveGraphHolder {
	pg := newPrimitiveGraph(concurrencyLimit)
	return &standardPrimitiveGraphHolder{
		concurrencyLimit: concurrencyLimit,
		pg:               pg,
		outputRegister:   internaldto.NewExecutorOutputRegister(),
	}
}

func (pgh *standardPrimitiveGraphHolder) Blank() error {
	pgh.pg = newPrimitiveGraph(pgh.concurrencyLimit)
	return nil
}
