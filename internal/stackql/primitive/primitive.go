package primitive

import (
	"io"

	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/stackql/internal/stackql/acid/binlog"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
)

type IPrimitiveCtx interface {
	GetAuthContext(string) (*dto.AuthCtx, error)
	GetWriter() io.Writer
	GetErrWriter() io.Writer
}

type IPrimitive interface {
	Optimise() error

	Execute(IPrimitiveCtx) internaldto.ExecutorOutput

	SetExecutor(func(pc IPrimitiveCtx) internaldto.ExecutorOutput) error

	SetTxnID(int)
	//
	IsReadOnly() bool

	// Get the redo log entry.
	GetRedoLog() (binlog.LogEntry, bool)
	// Get the undo log entry.
	GetUndoLog() (binlog.LogEntry, bool)

	// Get the redo log entry.
	SetRedoLog(binlog.LogEntry)
	// Get the undo log entry.
	SetUndoLog(binlog.LogEntry)

	IncidentData(int64, internaldto.ExecutorOutput) error

	SetInputAlias(string, int64) error

	GetInputFromAlias(string) (internaldto.ExecutorOutput, bool)

	WithDebugName(string) IPrimitive
}
