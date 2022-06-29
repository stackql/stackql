package primitive

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
)

type PassThroughPrimitive struct {
	Inputs                 map[int64]dto.ExecutorOutput
	id                     int64
	sqlEngine              sqlengine.SQLEngine
	shouldCollectGarbage   bool
	txnControlCounterSlice []dto.TxnControlCounters
}

func NewPassThroughPrimitive(sqlEngine sqlengine.SQLEngine, txnControlCounterSlice []dto.TxnControlCounters, shouldCollectGarbage bool) IPrimitive {
	return &PassThroughPrimitive{
		Inputs:                 make(map[int64]dto.ExecutorOutput),
		sqlEngine:              sqlEngine,
		txnControlCounterSlice: txnControlCounterSlice,
		shouldCollectGarbage:   shouldCollectGarbage,
	}
}

func (pr *PassThroughPrimitive) SetTxnId(id int) {
}

func (pr *PassThroughPrimitive) SetInputAlias(alias string, id int64) error {
	return nil
}

func (pr *PassThroughPrimitive) Optimise() error {
	return nil
}

func (pr *PassThroughPrimitive) GetInputFromAlias(string) (dto.ExecutorOutput, bool) {
	var rv dto.ExecutorOutput
	return rv, false
}

func (pr *PassThroughPrimitive) SetExecutor(func(pc IPrimitiveCtx) dto.ExecutorOutput) error {
	return fmt.Errorf("pass through primitive does not support SetExecutor()")
}

func (pr *PassThroughPrimitive) IncidentData(fromId int64, input dto.ExecutorOutput) error {
	pr.Inputs[fromId] = input
	return nil
}

func (pt *PassThroughPrimitive) collectGarbage() {
	if pt.shouldCollectGarbage {
		for _, gc := range pt.txnControlCounterSlice {
			pt.sqlEngine.GCCollectObsolete(&gc)
		}
	}
}

func (pt *PassThroughPrimitive) Execute(pc IPrimitiveCtx) dto.ExecutorOutput {
	defer pt.collectGarbage()
	for _, input := range pt.Inputs {
		return input
	}
	return dto.ExecutorOutput{}
}
