package primitive

import (
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/dto"
)

type LocalPrimitive struct {
	Executor   func(pc IPrimitiveCtx) dto.ExecutorOutput
	Preparator func() *drm.PreparedStatementCtx
	Inputs     map[int64]dto.ExecutorOutput
	id         int64
}

func NewLocalPrimitive(executor func(pc IPrimitiveCtx) dto.ExecutorOutput) IPrimitive {
	return &LocalPrimitive{
		Executor: executor,
		Inputs:   make(map[int64]dto.ExecutorOutput),
	}
}

func (pr *LocalPrimitive) SetTxnId(id int) {
}

func (pr *LocalPrimitive) IncidentData(fromId int64, input dto.ExecutorOutput) error {
	pr.Inputs[fromId] = input
	return nil
}

func (pr *LocalPrimitive) SetInputAlias(alias string, id int64) error {
	return nil
}

func (pr *LocalPrimitive) Optimise() error {
	return nil
}

func (pr *LocalPrimitive) GetInputFromAlias(string) (dto.ExecutorOutput, bool) {
	var rv dto.ExecutorOutput
	return rv, false
}

func (pr *LocalPrimitive) SetExecutor(ex func(pc IPrimitiveCtx) dto.ExecutorOutput) error {
	pr.Executor = ex
	return nil
}

func (pr *LocalPrimitive) ID() int64 {
	return pr.id
}

func (pr *LocalPrimitive) Execute(pc IPrimitiveCtx) dto.ExecutorOutput {
	if pr.Executor != nil {
		return pr.Executor(pc)
	}
	return dto.NewExecutorOutput(nil, nil, nil, nil, nil)
}
