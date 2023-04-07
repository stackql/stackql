package primitive

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/acid/binlog"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/sql_system"
)

type PassThroughPrimitive struct {
	Inputs                 map[int64]internaldto.ExecutorOutput
	sqlSystem              sql_system.SQLSystem
	shouldCollectGarbage   bool
	txnControlCounterSlice []internaldto.TxnControlCounters
}

func NewPassThroughPrimitive(
	sqlSystem sql_system.SQLSystem,
	txnControlCounterSlice []internaldto.TxnControlCounters,
	shouldCollectGarbage bool) IPrimitive {
	return &PassThroughPrimitive{
		Inputs:                 make(map[int64]internaldto.ExecutorOutput),
		sqlSystem:              sqlSystem,
		txnControlCounterSlice: txnControlCounterSlice,
		shouldCollectGarbage:   shouldCollectGarbage,
	}
}

func (pr *PassThroughPrimitive) SetTxnID(_ int) {
}

func (pr *PassThroughPrimitive) IsReadOnly() bool {
	return true
}

func (pr *PassThroughPrimitive) GetRedoLog() (binlog.LogEntry, bool) {
	return nil, false
}

func (pr *PassThroughPrimitive) GetUndoLog() (binlog.LogEntry, bool) {
	return nil, false
}

func (pr *PassThroughPrimitive) SetInputAlias(_ string, _ int64) error {
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

func (pr *PassThroughPrimitive) IncidentData(fromID int64, input internaldto.ExecutorOutput) error {
	pr.Inputs[fromID] = input
	return nil
}

func (pr *PassThroughPrimitive) collectGarbage() {
	// placeholder
}

func (pr *PassThroughPrimitive) Execute(_ IPrimitiveCtx) internaldto.ExecutorOutput {
	defer pr.collectGarbage()
	for _, input := range pr.Inputs {
		return input
	}
	return internaldto.NewEmptyExecutorOutput()
}
