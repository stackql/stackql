package txncounter

import (
	"sync"
)

//nolint:revive,gochecknoglobals // Explicit type declaration removes any ambiguity
var (
	//TODO: Remove global variables
	txnCtrlMutex *sync.Mutex = &sync.Mutex{}
	currentTxnID *int        = new(int)
)

type Manager interface {
	GetCurrentGenerationID() (int, error)
	GetCurrentSessionID() (int, error)
	GetNextInsertID() (int, error)
	GetNextTxnID() (int, error)
}

type standardTxnCounterManager struct {
	perTxnMutex     *sync.Mutex
	generationID    int
	sessionID       int
	currentInsertID int
}

func NewTxnCounterManager(generationID, sessionID int) Manager {
	return &standardTxnCounterManager{
		generationID: generationID,
		sessionID:    sessionID,
		perTxnMutex:  &sync.Mutex{},
	}
}

func (tc *standardTxnCounterManager) GetCurrentGenerationID() (int, error) {
	return tc.generationID, nil
}

func (tc *standardTxnCounterManager) GetCurrentSessionID() (int, error) {
	return tc.sessionID, nil
}

func (tc *standardTxnCounterManager) GetNextTxnID() (int, error) {
	txnCtrlMutex.Lock()
	defer txnCtrlMutex.Unlock()
	*currentTxnID++
	return *currentTxnID, nil
}

func (tc *standardTxnCounterManager) GetNextInsertID() (int, error) {
	tc.perTxnMutex.Lock()
	defer tc.perTxnMutex.Unlock()
	tc.currentInsertID++
	return tc.currentInsertID, nil
}
