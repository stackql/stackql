package primitive

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/provider"
)

type MetaDataPrimitive struct {
	Provider   provider.IProvider
	Executor   func(pc IPrimitiveCtx) dto.ExecutorOutput
	Preparator func() *drm.PreparedStatementCtx
	id         int64
}

func (pr *MetaDataPrimitive) SetTxnId(id int) {
}

func (pr *MetaDataPrimitive) IncidentData(fromId int64, input dto.ExecutorOutput) error {
	return fmt.Errorf("MetaDataPrimitive cannot handle IncidentData")
}

func (pr *MetaDataPrimitive) SetInputAlias(alias string, id int64) error {
	return nil
}

func (pr *MetaDataPrimitive) Optimise() error {
	return nil
}

func (pr *MetaDataPrimitive) GetInputFromAlias(string) (dto.ExecutorOutput, bool) {
	var rv dto.ExecutorOutput
	return rv, false
}

func (pr *MetaDataPrimitive) SetExecutor(ex func(pc IPrimitiveCtx) dto.ExecutorOutput) error {
	pr.Executor = ex
	return nil
}

func (pr *MetaDataPrimitive) ID() int64 {
	return pr.id
}

func (pr *MetaDataPrimitive) Execute(pc IPrimitiveCtx) dto.ExecutorOutput {
	if pr.Executor != nil {
		return pr.Executor(pc)
	}
	return dto.NewExecutorOutput(nil, nil, nil, nil, nil)
}

func NewMetaDataPrimitive(provider provider.IProvider, executor func(pc IPrimitiveCtx) dto.ExecutorOutput) IPrimitive {
	return &MetaDataPrimitive{
		Provider: provider,
		Executor: executor,
	}
}
