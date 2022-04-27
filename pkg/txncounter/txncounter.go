package txncounter

import (
	"sync"
)

var (
	genCtrlMutex        *sync.Mutex = &sync.Mutex{}
	txnCtrlMutex        *sync.Mutex = &sync.Mutex{}
	currentTxnId        *int        = new(int)
	currentGenerationId *int        = new(int)
)

func GetNextGenerationId() int {
	txnCtrlMutex.Lock()
	defer txnCtrlMutex.Unlock()
	*currentGenerationId++
	return *currentGenerationId
}

type TxnCounterManager struct {
	perTxnMutex     *sync.Mutex
	generationId    int
	sessionId       int
	currentInsertId int
}

func NewTxnCounterManager(generationId, sessionId int) *TxnCounterManager {
	return &TxnCounterManager{
		generationId: generationId,
		sessionId:    sessionId,
		perTxnMutex:  &sync.Mutex{},
	}
}

func (tc *TxnCounterManager) GetCurrentGenerationId() int {
	return tc.generationId
}

func (tc *TxnCounterManager) GetCurrentSessionId() int {
	return tc.sessionId
}

func (tc *TxnCounterManager) GetNextTxnId() int {
	txnCtrlMutex.Lock()
	defer txnCtrlMutex.Unlock()
	*currentTxnId++
	return *currentTxnId
}

func (tc *TxnCounterManager) GetNextInsertId() int {
	tc.perTxnMutex.Lock()
	defer tc.perTxnMutex.Unlock()
	tc.currentInsertId++
	return tc.currentInsertId
}
