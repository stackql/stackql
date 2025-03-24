package primitive

import (
	"github.com/stackql/any-sdk/pkg/logging"
	"github.com/stackql/stackql/internal/stackql/acid/binlog"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/primitive_context"
)

type GenericPrimitive struct {
	Executor      func(pc IPrimitiveCtx) internaldto.ExecutorOutput
	Preparator    func() drm.PreparedStatementCtx
	TxnControlCtr internaldto.TxnControlCounters
	Inputs        map[int64]internaldto.ExecutorOutput
	InputAliases  map[string]int64
	id            int64
	isReadOnly    bool
	undoLog       binlog.LogEntry
	redoLog       binlog.LogEntry
	debugName     string
}

func NewGenericPrimitive(
	executor func(pc IPrimitiveCtx) internaldto.ExecutorOutput,
	preparator func() drm.PreparedStatementCtx,
	txnCtrlCtr internaldto.TxnControlCounters,
	primitiveCtx primitive_context.IPrimitiveCtx,
) IPrimitive {
	return &GenericPrimitive{
		Executor:      executor,
		Preparator:    preparator,
		TxnControlCtr: txnCtrlCtr,
		Inputs:        make(map[int64]internaldto.ExecutorOutput),
		InputAliases:  make(map[string]int64),
		isReadOnly:    primitiveCtx.IsReadOnly(),
	}
}

func (pr *GenericPrimitive) WithDebugName(name string) IPrimitive {
	pr.debugName = name
	return pr
}

func (pr *GenericPrimitive) SetUndoLog(log binlog.LogEntry) {
	pr.undoLog = log
}

func (pr *GenericPrimitive) SetRedoLog(log binlog.LogEntry) {
	pr.redoLog = log
}

func (pr *GenericPrimitive) GetRedoLog() (binlog.LogEntry, bool) {
	return pr.redoLog, pr.redoLog != nil
}

func (pr *GenericPrimitive) GetUndoLog() (binlog.LogEntry, bool) {
	return pr.undoLog, pr.undoLog != nil
}

func (pr *GenericPrimitive) SetTxnID(id int) {
	if pr.TxnControlCtr != nil {
		pr.TxnControlCtr.SetTxnID(id)
	}
}

func (pr *GenericPrimitive) IsReadOnly() bool {
	return pr.isReadOnly
}

func (pr *GenericPrimitive) IncidentData(fromID int64, input internaldto.ExecutorOutput) error {
	pr.Inputs[fromID] = input
	return nil
}

func (pr *GenericPrimitive) SetInputAlias(alias string, id int64) error {
	pr.InputAliases[alias] = id
	return nil
}

func (pr *GenericPrimitive) Optimise() error {
	return nil
}

func (pr *GenericPrimitive) GetInputFromAlias(alias string) (internaldto.ExecutorOutput, bool) {
	var rv internaldto.ExecutorOutput
	key, keyExists := pr.InputAliases[alias]
	if !keyExists {
		return rv, false
	}
	input, inputExists := pr.Inputs[key]
	if !inputExists {
		return rv, false
	}
	return input, true
}

func (pr *GenericPrimitive) Execute(pc IPrimitiveCtx) internaldto.ExecutorOutput {
	if pr.Executor != nil {
		logging.GetLogger().Debugf("running HTTP rest primitive %s", pr.debugName)
		op := pr.Executor(pc)
		return op
	}
	return internaldto.NewExecutorOutput(nil, nil, nil, nil, nil)
}

func (pr *GenericPrimitive) ID() int64 {
	return pr.id
}

func (pr *GenericPrimitive) SetExecutor(ex func(pc IPrimitiveCtx) internaldto.ExecutorOutput) error {
	pr.Executor = ex
	return nil
}
