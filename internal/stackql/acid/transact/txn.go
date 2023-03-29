package transact

import (
	"fmt"
	"sync"
)

//nolint:gochecknoglobals // singleton pattern
var (
	coordinatorOnce      sync.Once
	coordinatorSingleton Coordinator
	_                    Coordinator = &standardCoordinator{}
	_                    Manager     = &basicTransactionManager{}
)

// The transaction coordinator is singleton
// that orchestrates transaction managers.
type Coordinator interface {
	// Create a new transaction manager.
	NewTxnManager() (Manager, error)
}

type standardCoordinator struct {
}

func (c *standardCoordinator) NewTxnManager() (Manager, error) {
	return nil, fmt.Errorf("not implemented")
}

func GetCoordinatorInstance() (Coordinator, error) {
	var err error
	coordinatorOnce.Do(func() {
		if err != nil {
			return
		}
		coordinatorSingleton = &standardCoordinator{}
	})
	return coordinatorSingleton, err
}

// The transaction manager ensures
// that undo and redo logs are kept
// and that 2PC is performed.
type Manager interface {
	// Begin a new transaction.
	Begin() (Manager, error)
	// Commit the current transaction.
	Commit() error
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
}

type basicTransactionManager struct {
	parent            Manager
	statementSequence []Statement
}

func newBasicTransactionManager(parent Manager) Manager {
	return &basicTransactionManager{
		parent: parent,
	}
}

func NewManager() Manager {
	return newBasicTransactionManager(nil)
}

func (m *basicTransactionManager) Begin() (Manager, error) {
	if m.Depth() >= 1 {
		return nil, fmt.Errorf("cannot begin nested transaction")
	}
	return newBasicTransactionManager(m), nil
}

func (m *basicTransactionManager) Commit() error {
	for _, stmt := range m.statementSequence {
		coDomain := stmt.Execute()
		err := coDomain.GetError()
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *basicTransactionManager) Rollback() error {
	return fmt.Errorf("not implemented")
}

func (m *basicTransactionManager) Enqueue(_ Statement) error {
	return fmt.Errorf("not implemented")
}

func (m *basicTransactionManager) Depth() int {
	if m.parent != nil {
		return m.parent.Depth() + 1
	}
	return 0
}
