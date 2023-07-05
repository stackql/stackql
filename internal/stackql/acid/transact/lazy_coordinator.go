package transact

import (
	"fmt"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/acid/acid_dto"
	"github.com/stackql/stackql/internal/stackql/acid/binlog"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
)

var (
	_ Coordinator = &basicLazyTransactionCoordinator{}
)

type basicLazyTransactionCoordinator struct {
	parent            Coordinator
	statementSequence []Statement
	undoLogs          []binlog.LogEntry
	redoLogs          []binlog.LogEntry
	maxTxnDepth       int
	outputs           []internaldto.ExecutorOutput
	isExecuted        bool
}

func newBasicLazyTransactionCoordinator(parent Coordinator, maxTxnDepth int) Coordinator {
	return &basicLazyTransactionCoordinator{
		parent:      parent,
		maxTxnDepth: maxTxnDepth,
	}
}

func (m *basicLazyTransactionCoordinator) GetPrimitiveGraphHolder() (primitivegraph.PrimitiveGraphHolder, bool) {
	return nil, false
}

func (m *basicLazyTransactionCoordinator) IsReadOnly() bool {
	for _, statement := range m.statementSequence {
		if !statement.IsReadOnly() {
			return false
		}
	}
	return true
}

func (m *basicLazyTransactionCoordinator) GetAST() (sqlparser.Statement, bool) {
	return nil, false
}

func (m *basicLazyTransactionCoordinator) GetParent() (Coordinator, bool) {
	return m.parent, m.parent != nil
}

func (m *basicLazyTransactionCoordinator) AppendRedoLog(log binlog.LogEntry) {
	m.redoLogs = append(m.redoLogs, log)
}

func (m *basicLazyTransactionCoordinator) AppendUndoLog(log binlog.LogEntry) {
	m.undoLogs = append(m.undoLogs, log)
}

func (m *basicLazyTransactionCoordinator) IsBegin() bool {
	return false
}

func (m *basicLazyTransactionCoordinator) IsCommit() bool {
	return false
}

func (m *basicLazyTransactionCoordinator) IsRollback() bool {
	return false
}

func (m *basicLazyTransactionCoordinator) SetUndoLog(log binlog.LogEntry) {
	m.undoLogs = []binlog.LogEntry{log}
}

func (m *basicLazyTransactionCoordinator) GetRedoLog() (binlog.LogEntry, bool) {
	rv := binlog.NewSimpleLogEntry(nil, nil)
	if len(m.redoLogs) == 0 {
		return nil, false
	}
	for _, log := range m.redoLogs {
		rv = rv.Concatenate(log)
	}
	return rv, true
}

func (m *basicLazyTransactionCoordinator) GetUndoLog() (binlog.LogEntry, bool) {
	rv := binlog.NewSimpleLogEntry(nil, nil)
	if len(m.undoLogs) == 0 {
		return nil, false
	}
	for _, log := range m.undoLogs {
		rv = rv.Concatenate(log)
	}
	return rv, true
}

func (m *basicLazyTransactionCoordinator) Prepare() error {
	var err error
	for _, stmt := range m.statementSequence {
		err = stmt.Prepare()
		if err != nil {
			return err
		}
	}
	return nil
}

// This is an approximation of 2PC.
func (m *basicLazyTransactionCoordinator) Execute() internaldto.ExecutorOutput {
	outputs, err := m.votingPhase()
	m.outputs = outputs
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	return internaldto.NewExecutorOutput(
		nil,
		nil,
		nil,
		internaldto.NewBackendMessages([]string{"transaction committed"}),
		nil,
	)
}

func (m *basicLazyTransactionCoordinator) IsExecuted() bool {
	return m.outputs != nil
}

func (m *basicLazyTransactionCoordinator) votingPhase() ([]internaldto.ExecutorOutput, error) {
	defer func() {
		m.isExecuted = true
	}()
	var rv []internaldto.ExecutorOutput
	for _, stmt := range m.statementSequence {
		coDomain := stmt.Execute()
		rv = append(rv, coDomain)
		err := coDomain.GetError()
		undoLog, undoLogExists := coDomain.GetUndoLog()
		redoLog, redoLogExists := coDomain.GetRedoLog()
		if undoLogExists {
			m.undoLogs = append(m.undoLogs, undoLog)
		}
		if redoLogExists {
			m.redoLogs = append(m.redoLogs, redoLog)
		}
		if err != nil {
			return rv, err
		}
	}
	return rv, nil
}

func (m *basicLazyTransactionCoordinator) completionPhase() error {
	return nil
}

func (m *basicLazyTransactionCoordinator) GetQuery() string {
	return ""
}

func (m *basicLazyTransactionCoordinator) Begin() (Coordinator, error) {
	if m.maxTxnDepth >= 0 && m.Depth() >= m.maxTxnDepth {
		return nil, fmt.Errorf("cannot begin nested transaction of depth = %d", m.Depth()+1)
	}
	return newBasicLazyTransactionCoordinator(m, m.maxTxnDepth), nil
}

func (m *basicLazyTransactionCoordinator) Commit() acid_dto.CommitCoDomain {
	rv, err := m.votingPhase()
	if err != nil {
		return acid_dto.NewCommitCoDomain(rv, err, nil)
	}
	completionErr := m.completionPhase()
	return acid_dto.NewCommitCoDomain(rv, nil, completionErr)
}

// Rollback is a no-op for now.
// The redo logs will simply be
// displayed to the user.
func (m *basicLazyTransactionCoordinator) Rollback() acid_dto.CommitCoDomain {
	return acid_dto.NewCommitCoDomain(
		nil,
		nil,
		nil,
	)
}

func (m *basicLazyTransactionCoordinator) Enqueue(stmt Statement) error {
	m.statementSequence = append(m.statementSequence, stmt)
	return nil
}

func (m *basicLazyTransactionCoordinator) Depth() int {
	return m.depth()
}

func (m *basicLazyTransactionCoordinator) IsRoot() bool {
	return m.parent == nil
}

func (m *basicLazyTransactionCoordinator) depth() int {
	if m.parent != nil {
		return m.parent.Depth() + 1
	}
	return 0
}
