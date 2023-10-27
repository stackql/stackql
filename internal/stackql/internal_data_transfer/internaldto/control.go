package internaldto

import (
	"sync"

	"github.com/stackql/stackql/pkg/txncounter"
)

var (
	_ TxnControlCounters = &standardTxnControlCounters{}
)

type TxnControlCounters interface {
	GetGenID() int
	GetInsertID() int
	GetSessionID() int
	GetTxnID() int
	SetTableName(string)
	SetTxnID(int)
	Clone() TxnControlCounters
	Copy(TxnControlCounters)
	CloneAndIncrementInsertID() TxnControlCounters
}

type standardTxnControlCounters struct {
	perTxnMutex                       *sync.Mutex
	genID, sessionID, txnID, insertID int
	maxInsertID                       *int
	tableName                         string
	requestEncoding                   []string
}

func NewTxnControlCounters(txnCtrMgr txncounter.Manager) (TxnControlCounters, error) {
	if txnCtrMgr == nil {
		return &standardTxnControlCounters{
			perTxnMutex: &sync.Mutex{},
		}, nil
	}
	genID, err := txnCtrMgr.GetCurrentGenerationID()
	if err != nil {
		return nil, err
	}
	ssnID, err := txnCtrMgr.GetCurrentSessionID()
	if err != nil {
		return nil, err
	}
	txnID, err := txnCtrMgr.GetNextTxnID()
	if err != nil {
		return nil, err
	}
	insertID, err := txnCtrMgr.GetNextInsertID()
	if err != nil {
		return nil, err
	}
	return &standardTxnControlCounters{
		perTxnMutex: &sync.Mutex{},
		genID:       genID,
		sessionID:   ssnID,
		txnID:       txnID,
		insertID:    insertID,
		maxInsertID: &insertID,
	}, nil
}

func NewTxnControlCountersFromVals(genID, ssnID, txnID, insertID int) TxnControlCounters {
	return &standardTxnControlCounters{
		perTxnMutex: &sync.Mutex{},
		genID:       genID,
		sessionID:   ssnID,
		txnID:       txnID,
		insertID:    insertID,
		maxInsertID: &insertID,
	}
}

func (tc *standardTxnControlCounters) SetTableName(tn string) {
	tc.tableName = tn
}

func (tc *standardTxnControlCounters) GetGenID() int {
	return tc.genID
}

func (tc *standardTxnControlCounters) GetSessionID() int {
	return tc.sessionID
}

func (tc *standardTxnControlCounters) GetTxnID() int {
	return tc.txnID
}

func (tc *standardTxnControlCounters) GetInsertID() int {
	tc.perTxnMutex.Lock()
	defer tc.perTxnMutex.Unlock()
	return tc.insertID
}

func (tc *standardTxnControlCounters) Copy(input TxnControlCounters) {
	tc.genID = input.GetGenID()
	tc.insertID = input.GetInsertID()
	tc.sessionID = input.GetSessionID()
	tc.txnID = input.GetTxnID()
}

func (tc *standardTxnControlCounters) Clone() TxnControlCounters {
	tc.perTxnMutex.Lock()
	defer tc.perTxnMutex.Unlock()
	return &standardTxnControlCounters{
		perTxnMutex:     tc.perTxnMutex,
		genID:           tc.genID,
		sessionID:       tc.sessionID,
		txnID:           tc.txnID,
		insertID:        tc.insertID,
		maxInsertID:     tc.maxInsertID,
		requestEncoding: tc.requestEncoding,
	}
}

func (tc *standardTxnControlCounters) CloneAndIncrementInsertID() TxnControlCounters {
	tc.perTxnMutex.Lock()
	defer tc.perTxnMutex.Unlock()
	nextInsertID := *tc.maxInsertID + 1
	*tc.maxInsertID = nextInsertID
	return &standardTxnControlCounters{
		perTxnMutex:     tc.perTxnMutex,
		genID:           tc.genID,
		sessionID:       tc.sessionID,
		txnID:           tc.txnID,
		insertID:        nextInsertID,
		maxInsertID:     tc.maxInsertID,
		requestEncoding: tc.requestEncoding,
	}
}

func (tc *standardTxnControlCounters) SetTxnID(ti int) {
	tc.txnID = ti
}
