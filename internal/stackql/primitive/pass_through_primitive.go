package primitive

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/sql_system"
)

type PassThroughPrimitive struct {
	Inputs                 map[int64]internaldto.ExecutorOutput
	id                     int64
	sqlSystem              sql_system.SQLSystem
	shouldCollectGarbage   bool
	txnControlCounterSlice []internaldto.TxnControlCounters
}

func NewPassThroughPrimitive(sqlSystem sql_system.SQLSystem, txnControlCounterSlice []internaldto.TxnControlCounters, shouldCollectGarbage bool) IPrimitive {
	return &PassThroughPrimitive{
		Inputs:                 make(map[int64]internaldto.ExecutorOutput),
		sqlSystem:              sqlSystem,
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

func (pr *PassThroughPrimitive) GetInputFromAlias(string) (internaldto.ExecutorOutput, bool) {
	var rv internaldto.ExecutorOutput
	return rv, false
}

func (pr *PassThroughPrimitive) SetExecutor(func(pc IPrimitiveCtx) internaldto.ExecutorOutput) error {
	return fmt.Errorf("pass through primitive does not support SetExecutor()")
}

func (pr *PassThroughPrimitive) IncidentData(fromId int64, input internaldto.ExecutorOutput) error {
	pr.Inputs[fromId] = input
	return nil
}

func (pt *PassThroughPrimitive) collectGarbage() {
	// placeholder
}

func (pt *PassThroughPrimitive) Execute(pc IPrimitiveCtx) internaldto.ExecutorOutput {
	defer pt.collectGarbage()
	for _, input := range pt.Inputs {
		return input
	}
	return internaldto.ExecutorOutput{}
}
