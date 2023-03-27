package transact

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/primitive"
)

var (
	_ Operation = &reversibleOperation{}
	_ Operation = &irreversibleOperation{}
)

// The Operation is an abstract
// data type that represents
// a stackql action.
// The operation maps to each of:
//   - an executable action.
//   - a redo log entry.
//   - an undo log entry.
//
// One possible implementation is to
// store a nullable primitive (plan) graph
// node alongside log entries.
type Operation interface {
	// Execute the operation.
	Execute(primitive.IPrimitiveCtx) internaldto.ExecutorOutput
	// Reverse the operation.
	Undo() error
	// Get the redo log entry.
	GetRedoLog() (LogEntry, bool)
	// Get the undo log entry.
	GetUndoLog() (LogEntry, bool)
	//
	IncidentData(int64, internaldto.ExecutorOutput) error
	//
	SetTxnID(id int)
	//
	SetInputAlias(alias string, id int64) error
}

type reversibleOperation struct {
	redoLog LogEntry
	undoLog LogEntry
	pr      primitive.IPrimitive
}

func (op *reversibleOperation) Execute(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
	rv := op.pr.Execute(pc)
	return rv
}

func (op *reversibleOperation) IncidentData(id int64, ou internaldto.ExecutorOutput) error {
	return op.pr.IncidentData(id, ou)
}

func (op *reversibleOperation) SetTxnID(id int) {
	op.pr.SetTxnID(id)
}

func (op *reversibleOperation) SetInputAlias(alias string, id int64) error {
	return op.pr.SetInputAlias(alias, id)
}

func (op *reversibleOperation) Undo() error {
	return nil
}

func (op *reversibleOperation) Redo() error {
	return nil
}

func (op *reversibleOperation) GetRedoLog() (LogEntry, bool) {
	return op.redoLog, true
}

func (op *reversibleOperation) GetUndoLog() (LogEntry, bool) {
	return op.undoLog, true
}

type irreversibleOperation struct {
	redoLog LogEntry
	pr      primitive.IPrimitive
}

func (op *irreversibleOperation) Execute(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
	res := op.pr.Execute(pc)
	return res
}

func (op *irreversibleOperation) IncidentData(id int64, ou internaldto.ExecutorOutput) error {
	return op.pr.IncidentData(id, ou)
}

func (op *irreversibleOperation) SetTxnID(id int) {
	op.pr.SetTxnID(id)
}

func (op *irreversibleOperation) SetInputAlias(alias string, id int64) error {
	return op.pr.SetInputAlias(alias, id)
}

func (op *irreversibleOperation) Undo() error {
	return fmt.Errorf("irreversible operation cannot be undone")
}

func (op *irreversibleOperation) Redo() error {
	return nil
}

func (op *irreversibleOperation) GetRedoLog() (LogEntry, bool) {
	return op.redoLog, true
}

func (op *irreversibleOperation) GetUndoLog() (LogEntry, bool) {
	return nil, false
}

func NewIrreversibleOperation(pr primitive.IPrimitive) Operation {
	return &irreversibleOperation{
		pr: pr,
	}
}
