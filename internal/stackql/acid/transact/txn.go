package transact

import (
	"fmt"
	"sync"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/acid/binlog"
	"github.com/stackql/stackql/internal/stackql/acid/txn_context"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
)

//nolint:gochecknoglobals // singleton pattern
var (
	coordinatorOnce      sync.Once
	coordinatorSingleton Coordinator
	_                    Coordinator = &standardCoordinator{}
	_                    Manager     = &basicTransactionManager{}
)

const (
	defaultMaxStackDepth = 1
)

// The transaction coordinator is singleton
// that orchestrates transaction managers.
type Coordinator interface {
	// Create a new transaction manager.
	NewTxnManager() (Manager, error)
}

type standardCoordinator struct {
	ctx txn_context.ITransactionCoordinatorContext
}

func (c *standardCoordinator) NewTxnManager() (Manager, error) {
	maxTxnDepth := defaultMaxStackDepth
	if c.ctx != nil {
		maxTxnDepth = c.ctx.GetMaxStackDepth()
	}
	return NewManager(maxTxnDepth), nil
}

func GetCoordinatorInstance(ctx txn_context.ITransactionCoordinatorContext) (Coordinator, error) {
	var err error
	coordinatorOnce.Do(func() {
		if err != nil {
			return
		}
		coordinatorSingleton = &standardCoordinator{
			ctx: ctx,
		}
	})
	return coordinatorSingleton, err
}

// The transaction manager ensures
// that undo and redo logs are kept
// and that 2PC is performed.
type Manager interface {
	Statement
	// Begin a new transaction.
	Begin() (Manager, error)
	// Commit the current transaction.
	Commit() ([]internaldto.ExecutorOutput, error)
	// Rollback the current transaction.
	Rollback() error
	// Enqueue a transaction operation.
	// This method will return an error
	// in the case that the transaction
	// context disallows a particular
	// operation or type of operation.
	Enqueue(Statement) error
	// Get the depth of transaction nesting.
	Depth() int
	// Get the parent transaction manager.
	GetParent() (Manager, bool)
	//
	IsRoot() bool
}

type basicTransactionManager struct {
	parent            Manager
	statementSequence []Statement
	undoLogs          []binlog.LogEntry
	redoLogs          []binlog.LogEntry
	maxTxnDepth       int
	outputs           []internaldto.ExecutorOutput
}

func newBasicTransactionManager(parent Manager, maxTxnDepth int) Manager {
	return &basicTransactionManager{
		parent:      parent,
		maxTxnDepth: maxTxnDepth,
	}
}

func NewManager(maxTxnDepth int) Manager {
	return newBasicTransactionManager(nil, maxTxnDepth)
}

func (m *basicTransactionManager) IsReadOnly() bool {
	for _, statement := range m.statementSequence {
		if !statement.IsReadOnly() {
			return false
		}
	}
	return true
}

func (m *basicTransactionManager) GetAST() (sqlparser.Statement, bool) {
	return nil, false
}

func (m *basicTransactionManager) GetParent() (Manager, bool) {
	return m.parent, m.parent != nil
}

func (m *basicTransactionManager) SetRedoLog(log binlog.LogEntry) {
	m.redoLogs = []binlog.LogEntry{log}
}

func (m *basicTransactionManager) IsBegin() bool {
	return false
}

func (m *basicTransactionManager) IsCommit() bool {
	return false
}

func (m *basicTransactionManager) IsRollback() bool {
	return false
}

func (m *basicTransactionManager) SetUndoLog(log binlog.LogEntry) {
	m.undoLogs = []binlog.LogEntry{log}
}

func (m *basicTransactionManager) GetUndoLog() (binlog.LogEntry, bool) {
	if len(m.undoLogs) == 0 {
		return nil, false
	}
	initialUndoLog := m.undoLogs[len(m.undoLogs)-1]
	rv := initialUndoLog.Clone()
	for i := len(m.undoLogs) - 2; i >= 0; i-- { //nolint:gomnd // magic number second from last
		currentLog := m.undoLogs[i]
		if currentLog != nil {
			rv.AppendHumanReadable(currentLog.GetHumanReadable())
			rv.AppendRaw(currentLog.GetRaw())
		}
	}
	return rv, true
}

func (m *basicTransactionManager) GetRedoLog() (binlog.LogEntry, bool) {
	return nil, false
}

func (m *basicTransactionManager) Prepare() error {
	var err error
	for _, stmt := range m.statementSequence {
		err = stmt.Prepare()
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *basicTransactionManager) Execute() internaldto.ExecutorOutput {
	outputs, err := m.execute()
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

func (m *basicTransactionManager) execute() ([]internaldto.ExecutorOutput, error) {
	var rv []internaldto.ExecutorOutput
	for _, stmt := range m.statementSequence {
		coDomain := stmt.Execute()
		rv = append(rv, coDomain)
		err := coDomain.GetError()
		undoLog, undoLogExists := stmt.GetUndoLog()
		redoLog, redoLogExists := stmt.GetRedoLog()
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

func (m *basicTransactionManager) Begin() (Manager, error) {
	if m.maxTxnDepth >= 0 && m.Depth() >= m.maxTxnDepth {
		return nil, fmt.Errorf("cannot begin nested transaction of depth = %d", m.Depth()+1)
	}
	return newBasicTransactionManager(m, m.maxTxnDepth), nil
}

func (m *basicTransactionManager) Commit() ([]internaldto.ExecutorOutput, error) {
	return m.execute()
}

// Rollback is a no-op for now.
// The redo logs will simply be
// displayed to the user.
func (m *basicTransactionManager) Rollback() error {
	return nil
}

func (m *basicTransactionManager) Enqueue(stmt Statement) error {
	m.statementSequence = append(m.statementSequence, stmt)
	return nil
}

func (m *basicTransactionManager) Depth() int {
	return m.depth()
}

func (m *basicTransactionManager) IsRoot() bool {
	return m.parent == nil
}

func (m *basicTransactionManager) depth() int {
	if m.parent != nil {
		return m.parent.Depth() + 1
	}
	return 0
}
