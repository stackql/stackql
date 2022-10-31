package txncounter

import (
	"sync"
)

var (
	txnCtrlMutex *sync.Mutex = &sync.Mutex{}
	currentTxnId *int        = new(int)
)

type TxnCounterManager interface {
	GetCurrentGenerationId() (int, error)
	GetCurrentSessionId() (int, error)
	GetNextInsertId() (int, error)
	GetNextTxnId() (int, error)
}

type standardTxnCounterManager struct {
	perTxnMutex     *sync.Mutex
	generationId    int
	sessionId       int
	currentInsertId int
}

func NewTxnCounterManager(generationId, sessionId int) TxnCounterManager {
	return &standardTxnCounterManager{
		generationId: generationId,
		sessionId:    sessionId,
		perTxnMutex:  &sync.Mutex{},
	}
}

func (tc *standardTxnCounterManager) GetCurrentGenerationId() (int, error) {
	return tc.generationId, nil
}

func (tc *standardTxnCounterManager) GetCurrentSessionId() (int, error) {
	return tc.sessionId, nil
}

func (tc *standardTxnCounterManager) GetNextTxnId() (int, error) {
	txnCtrlMutex.Lock()
	defer txnCtrlMutex.Unlock()
	*currentTxnId++
	return *currentTxnId, nil
}

func (tc *standardTxnCounterManager) GetNextInsertId() (int, error) {
	tc.perTxnMutex.Lock()
	defer tc.perTxnMutex.Unlock()
	tc.currentInsertId++
	return tc.currentInsertId, nil
}
