package primitive

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/sqldialect"
)

type PassThroughPrimitive struct {
	Inputs                 map[int64]dto.ExecutorOutput
	id                     int64
	sqlDialect             sqldialect.SQLDialect
	shouldCollectGarbage   bool
	txnControlCounterSlice []dto.TxnControlCounters
}

func NewPassThroughPrimitive(sqlDialect sqldialect.SQLDialect, txnControlCounterSlice []dto.TxnControlCounters, shouldCollectGarbage bool) IPrimitive {
	return &PassThroughPrimitive{
		Inputs:                 make(map[int64]dto.ExecutorOutput),
		sqlDialect:             sqlDialect,
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
	// placeholder
}

func (pt *PassThroughPrimitive) Execute(pc IPrimitiveCtx) dto.ExecutorOutput {
	defer pt.collectGarbage()
	for _, input := range pt.Inputs {
		return input
	}
	return dto.ExecutorOutput{}
}
