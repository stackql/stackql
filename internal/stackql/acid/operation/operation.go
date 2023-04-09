package operation

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/acid/binlog"
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
	GetRedoLog() (binlog.LogEntry, bool)
	// Get the undo log entry.
	GetUndoLog() (binlog.LogEntry, bool)
	//
	IncidentData(int64, internaldto.ExecutorOutput) error
	//
	SetTxnID(id int)
	//
	SetInputAlias(alias string, id int64) error
	//
	IsReadOnly() bool
}

type reversibleOperation struct {
	pr primitive.IPrimitive
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

// TODO: interrogate the primitive
func (op *reversibleOperation) IsReadOnly() bool {
	return op.pr.IsReadOnly()
}

func (op *reversibleOperation) GetRedoLog() (binlog.LogEntry, bool) {
	return op.pr.GetRedoLog()
}

func (op *reversibleOperation) GetUndoLog() (binlog.LogEntry, bool) {
	return op.pr.GetUndoLog()
}

type irreversibleOperation struct {
	pr primitive.IPrimitive
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

// TODO: interrogate the primitive
func (op *irreversibleOperation) IsReadOnly() bool {
	return op.pr.IsReadOnly()
}

func (op *irreversibleOperation) GetRedoLog() (binlog.LogEntry, bool) {
	return op.pr.GetRedoLog()
}

func (op *irreversibleOperation) GetUndoLog() (binlog.LogEntry, bool) {
	return nil, false
}

func NewReversibleOperation(pr primitive.IPrimitive) Operation {
	return &reversibleOperation{
		pr: pr,
	}
}
