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
	AddInverseTxnControlCounters(t internaldto.TxnControlCounters)
	AddTxnControlCounters(t internaldto.TxnControlCounters)
	ContainsIndirect() bool
	CreateInversePrimitiveNode(pr primitive.IPrimitive) PrimitiveNode
	CreatePrimitiveNode(pr primitive.IPrimitive) PrimitiveNode
	GetInversePrimitiveGraph() PrimitiveGraph
	GetInverseTxnControlCounterSlice() []internaldto.TxnControlCounters
	GetPrimitiveGraph() PrimitiveGraph
	GetTxnControlCounterSlice() []internaldto.TxnControlCounters
	InverseContainsIndirect() bool
	NewInverseDependency(from PrimitiveNode, to PrimitiveNode, weight float64)
	NewDependency(from PrimitiveNode, to PrimitiveNode, weight float64)
	SetContainsIndirect(bool)
	SetInverseContainsIndirect(bool)
	SetTxnID(int)
	SetInverseTxnID(int)
	SetInverseContainsUserManagedRelation(containsUserRelation bool)
	SetContainsUserManagedRelation(containsUserRelation bool)
	InverseContainsUserManagedRelation() bool
	ContainsUserManagedRelation() bool
}

type standardPrimitiveGraphHolder struct {
	concurrencyLimit int
	pg               PrimitiveGraph
	ipg              PrimitiveGraph
}

func (pgh *standardPrimitiveGraphHolder) GetPrimitiveGraph() PrimitiveGraph {
	return pgh.pg
}

func (pgh *standardPrimitiveGraphHolder) SetTxnID(txnID int) {
	pgh.pg.SetTxnID(txnID)
}

func (pgh *standardPrimitiveGraphHolder) SetInverseTxnID(txnID int) {
	pgh.ipg.SetTxnID(txnID)
}

func (pgh *standardPrimitiveGraphHolder) GetInversePrimitiveGraph() PrimitiveGraph {
	return pgh.ipg
}

func (pgh *standardPrimitiveGraphHolder) AddTxnControlCounters(t internaldto.TxnControlCounters) {
	pgh.pg.AddTxnControlCounters(t)
}

func (pgh *standardPrimitiveGraphHolder) AddInverseTxnControlCounters(t internaldto.TxnControlCounters) {
	pgh.ipg.AddTxnControlCounters(t)
}

func (pgh *standardPrimitiveGraphHolder) CreatePrimitiveNode(pr primitive.IPrimitive) PrimitiveNode {
	return pgh.pg.CreatePrimitiveNode(pr)
}

func (pgh *standardPrimitiveGraphHolder) CreateInversePrimitiveNode(pr primitive.IPrimitive) PrimitiveNode {
	return pgh.ipg.CreatePrimitiveNode(pr)
}

func (pgh *standardPrimitiveGraphHolder) NewDependency(from PrimitiveNode, to PrimitiveNode, weight float64) {
	pgh.pg.NewDependency(from, to, weight)
}

func (pgh *standardPrimitiveGraphHolder) NewInverseDependency(from PrimitiveNode, to PrimitiveNode, weight float64) {
	pgh.ipg.NewDependency(to, from, weight)
}

func (pgh *standardPrimitiveGraphHolder) SetContainsIndirect(containsView bool) {
	pgh.pg.SetContainsIndirect(containsView)
}

func (pgh *standardPrimitiveGraphHolder) SetContainsUserManagedRelation(containsUserRelation bool) {
	pgh.pg.SetContainsUserManagedRelation(containsUserRelation)
}

func (pgh *standardPrimitiveGraphHolder) SetInverseContainsUserManagedRelation(containsUserRelation bool) {
	pgh.ipg.SetContainsUserManagedRelation(containsUserRelation)
}

func (pgh *standardPrimitiveGraphHolder) ContainsIndirect() bool {
	return pgh.pg.ContainsIndirect()
}

func (pgh *standardPrimitiveGraphHolder) InverseContainsIndirect() bool {
	return pgh.ipg.ContainsIndirect()
}

func (pgh *standardPrimitiveGraphHolder) ContainsUserManagedRelation() bool {
	return pgh.pg.ContainsUserManagedRelation()
}

func (pgh *standardPrimitiveGraphHolder) InverseContainsUserManagedRelation() bool {
	return pgh.ipg.ContainsUserManagedRelation()
}

func (pgh *standardPrimitiveGraphHolder) GetTxnControlCounterSlice() []internaldto.TxnControlCounters {
	return pgh.pg.GetTxnControlCounterSlice()
}

func (pgh *standardPrimitiveGraphHolder) GetInverseTxnControlCounterSlice() []internaldto.TxnControlCounters {
	return pgh.ipg.GetTxnControlCounterSlice()
}

func (pgh *standardPrimitiveGraphHolder) SetInverseContainsIndirect(containsView bool) {
	pgh.pg.SetContainsIndirect(containsView)
}

func NewPrimitiveGraphHolder(concurrencyLimit int) PrimitiveGraphHolder {
	pg := newPrimitiveGraph(concurrencyLimit)
	ipg := newSequentialPrimitiveGraph(concurrencyLimit)
	return &standardPrimitiveGraphHolder{
		concurrencyLimit: concurrencyLimit,
		pg:               pg,
		ipg:              ipg,
	}
}

func (pgh *standardPrimitiveGraphHolder) Blank() error {
	pgh.pg = newPrimitiveGraph(pgh.concurrencyLimit)
	pgh.ipg = newSequentialPrimitiveGraph(pgh.concurrencyLimit)
	return nil
}
