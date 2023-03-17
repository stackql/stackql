package primitive

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/provider"
)

type MetaDataPrimitive struct {
	Provider   provider.IProvider
	Executor   func(pc IPrimitiveCtx) internaldto.ExecutorOutput
	Preparator func() *drm.PreparedStatementCtx
	id         int64
}

func (pr *MetaDataPrimitive) SetTxnID(id int) {
}

func (pr *MetaDataPrimitive) IncidentData(fromID int64, input internaldto.ExecutorOutput) error {
	return fmt.Errorf("MetaDataPrimitive cannot handle IncidentData")
}

func (pr *MetaDataPrimitive) SetInputAlias(alias string, id int64) error {
	return nil
}

func (pr *MetaDataPrimitive) Optimise() error {
	return nil
}

func (pr *MetaDataPrimitive) GetInputFromAlias(string) (internaldto.ExecutorOutput, bool) {
	var rv internaldto.ExecutorOutput
	return rv, false
}

func (pr *MetaDataPrimitive) SetExecutor(ex func(pc IPrimitiveCtx) internaldto.ExecutorOutput) error {
	pr.Executor = ex
	return nil
}

func (pr *MetaDataPrimitive) ID() int64 {
	return pr.id
}

func (pr *MetaDataPrimitive) Execute(pc IPrimitiveCtx) internaldto.ExecutorOutput {
	if pr.Executor != nil {
		return pr.Executor(pc)
	}
	return internaldto.NewExecutorOutput(nil, nil, nil, nil, nil)
}

func NewMetaDataPrimitive(
	provider provider.IProvider,
	executor func(pc IPrimitiveCtx) internaldto.ExecutorOutput,
) IPrimitive {
	return &MetaDataPrimitive{
		Provider: provider,
		Executor: executor,
	}
}
