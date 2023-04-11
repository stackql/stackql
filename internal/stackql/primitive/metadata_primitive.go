package primitive

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/acid/binlog"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/provider"
)

type MetaDataPrimitive struct {
	Provider   provider.IProvider
	Executor   func(pc IPrimitiveCtx) internaldto.ExecutorOutput
	Preparator func() *drm.PreparedStatementCtx
	id         int64
	undoLog    binlog.LogEntry
	redoLog    binlog.LogEntry
}

func (pr *MetaDataPrimitive) SetTxnID(_ int) {
}

func (pr *MetaDataPrimitive) IsReadOnly() bool {
	return true
}

func (pr *MetaDataPrimitive) SetUndoLog(log binlog.LogEntry) {
	pr.undoLog = log
}

func (pr *MetaDataPrimitive) SetRedoLog(log binlog.LogEntry) {
	pr.redoLog = log
}

func (pr *MetaDataPrimitive) GetRedoLog() (binlog.LogEntry, bool) {
	return pr.redoLog, pr.redoLog != nil
}

func (pr *MetaDataPrimitive) GetUndoLog() (binlog.LogEntry, bool) {
	return pr.undoLog, pr.undoLog != nil
}

func (pr *MetaDataPrimitive) IncidentData(_ int64, _ internaldto.ExecutorOutput) error {
	return fmt.Errorf("MetaDataPrimitive cannot handle IncidentData")
}

func (pr *MetaDataPrimitive) SetInputAlias(_ string, _ int64) error {
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
