package transact

import (
	"fmt"
	"sync"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/acid/acid_dto"
	"github.com/stackql/stackql/internal/stackql/acid/binlog"
	"github.com/stackql/stackql/internal/stackql/acid/txn_context"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
)

//nolint:gochecknoglobals // singleton pattern
var (
	providerOnce      sync.Once
	providerSingleton Provider
	_                 Provider    = &standardProvider{}
	_                 Coordinator = &basicTransactionCoordinator{}
)

const (
	defaultMaxStackDepth = 1
)

// The transaction coordinator is singleton
// that orchestrates transaction managers.
type Provider interface {
	// Create a new transaction manager.
	NewTxnCoordinator() (Coordinator, error)
}

type standardProvider struct {
	ctx txn_context.ITransactionCoordinatorContext
}

func (c *standardProvider) NewTxnCoordinator() (Coordinator, error) {
	maxTxnDepth := defaultMaxStackDepth
	if c.ctx != nil {
		maxTxnDepth = c.ctx.GetMaxStackDepth()
	}
	return NewManager(maxTxnDepth), nil
}

func GetProviderInstance(ctx txn_context.ITransactionCoordinatorContext) (Provider, error) {
	var err error
	providerOnce.Do(func() {
		if err != nil {
			return
		}
		providerSingleton = &standardProvider{
			ctx: ctx,
		}
	})
	return providerSingleton, err
}

// The transaction coordinator ensures
// that undo and redo logs are kept
// and that 2PC is performed.
type Coordinator interface {
	Statement
	// Begin a new transaction.
	Begin() (Coordinator, error)
	// Commit the current transaction.
	Commit() acid_dto.CommitCoDomain
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
	GetParent() (Coordinator, bool)
	//
	IsRoot() bool
}

type basicTransactionCoordinator struct {
	parent            Coordinator
	statementSequence []Statement
	undoLogs          []binlog.LogEntry
	redoLogs          []binlog.LogEntry
	maxTxnDepth       int
	outputs           []internaldto.ExecutorOutput
}

func newBasicTransactionManager(parent Coordinator, maxTxnDepth int) Coordinator {
	return &basicTransactionCoordinator{
		parent:      parent,
		maxTxnDepth: maxTxnDepth,
	}
}

func NewManager(maxTxnDepth int) Coordinator {
	return newBasicTransactionManager(nil, maxTxnDepth)
}

func (m *basicTransactionCoordinator) IsReadOnly() bool {
	for _, statement := range m.statementSequence {
		if !statement.IsReadOnly() {
			return false
		}
	}
	return true
}

func (m *basicTransactionCoordinator) GetAST() (sqlparser.Statement, bool) {
	return nil, false
}

func (m *basicTransactionCoordinator) GetParent() (Coordinator, bool) {
	return m.parent, m.parent != nil
}

func (m *basicTransactionCoordinator) SetRedoLog(log binlog.LogEntry) {
	m.redoLogs = []binlog.LogEntry{log}
}

func (m *basicTransactionCoordinator) IsBegin() bool {
	return false
}

func (m *basicTransactionCoordinator) IsCommit() bool {
	return false
}

func (m *basicTransactionCoordinator) IsRollback() bool {
	return false
}

func (m *basicTransactionCoordinator) SetUndoLog(log binlog.LogEntry) {
	m.undoLogs = []binlog.LogEntry{log}
}

func (m *basicTransactionCoordinator) GetRedoLog() (binlog.LogEntry, bool) {
	return nil, false
}

func (m *basicTransactionCoordinator) GetUndoLog() (binlog.LogEntry, bool) {
	return nil, false
}

func (m *basicTransactionCoordinator) Prepare() error {
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
func (m *basicTransactionCoordinator) Execute() internaldto.ExecutorOutput {
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

func (m *basicTransactionCoordinator) votingPhase() ([]internaldto.ExecutorOutput, error) {
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

func (m *basicTransactionCoordinator) completionPhase() error {
	return nil
}

func (m *basicTransactionCoordinator) Begin() (Coordinator, error) {
	if m.maxTxnDepth >= 0 && m.Depth() >= m.maxTxnDepth {
		return nil, fmt.Errorf("cannot begin nested transaction of depth = %d", m.Depth()+1)
	}
	return newBasicTransactionManager(m, m.maxTxnDepth), nil
}

func (m *basicTransactionCoordinator) Commit() acid_dto.CommitCoDomain {
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
func (m *basicTransactionCoordinator) Rollback() error {
	return nil
}

func (m *basicTransactionCoordinator) Enqueue(stmt Statement) error {
	m.statementSequence = append(m.statementSequence, stmt)
	return nil
}

func (m *basicTransactionCoordinator) Depth() int {
	return m.depth()
}

func (m *basicTransactionCoordinator) IsRoot() bool {
	return m.parent == nil
}

func (m *basicTransactionCoordinator) depth() int {
	if m.parent != nil {
		return m.parent.Depth() + 1
	}
	return 0
}
