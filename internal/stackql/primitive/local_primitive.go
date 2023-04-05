package primitive

import (
	"github.com/stackql/stackql/internal/stackql/acid/binlog"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
)

type LocalPrimitive struct {
	Executor   func(pc IPrimitiveCtx) internaldto.ExecutorOutput
	Preparator func() *drm.PreparedStatementCtx
	Inputs     map[int64]internaldto.ExecutorOutput
	id         int64
}

func NewLocalPrimitive(executor func(pc IPrimitiveCtx) internaldto.ExecutorOutput) IPrimitive {
	return &LocalPrimitive{
		Executor: executor,
		Inputs:   make(map[int64]internaldto.ExecutorOutput),
	}
}

func (pr *LocalPrimitive) IsReadOnly() bool {
	return false
}

func (pr *LocalPrimitive) GetRedoLog() (binlog.LogEntry, bool) {
	return nil, false
}

func (pr *LocalPrimitive) GetUndoLog() (binlog.LogEntry, bool) {
	return nil, false
}

func (pr *LocalPrimitive) SetTxnID(_ int) {
}

func (pr *LocalPrimitive) IncidentData(fromID int64, input internaldto.ExecutorOutput) error {
	pr.Inputs[fromID] = input
	return nil
}

func (pr *LocalPrimitive) SetInputAlias(_ string, _ int64) error {
	return nil
}

func (pr *LocalPrimitive) Optimise() error {
	return nil
}

func (pr *LocalPrimitive) GetInputFromAlias(string) (internaldto.ExecutorOutput, bool) {
	var rv internaldto.ExecutorOutput
	return rv, false
}

func (pr *LocalPrimitive) SetExecutor(ex func(pc IPrimitiveCtx) internaldto.ExecutorOutput) error {
	pr.Executor = ex
	return nil
}

func (pr *LocalPrimitive) ID() int64 {
	return pr.id
}

func (pr *LocalPrimitive) Execute(pc IPrimitiveCtx) internaldto.ExecutorOutput {
	if pr.Executor != nil {
		return pr.Executor(pc)
	}
	return internaldto.NewExecutorOutput(nil, nil, nil, nil, nil)
}
