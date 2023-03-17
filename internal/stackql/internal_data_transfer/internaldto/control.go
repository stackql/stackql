package internaldto

import (
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
	Copy(TxnControlCounters) TxnControlCounters
	CloneAndIncrementInsertID() TxnControlCounters
}

type standardTxnControlCounters struct {
	genID, sessionID, txnID, insertID int
	tableName                         string
	requestEncoding                   []string
}

func NewTxnControlCounters(txnCtrMgr txncounter.Manager) (TxnControlCounters, error) {
	if txnCtrMgr == nil {
		return &standardTxnControlCounters{}, nil
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
		genID:     genID,
		sessionID: ssnID,
		txnID:     txnID,
		insertID:  insertID,
	}, nil
}

func NewTxnControlCountersFromVals(genID, ssnID, txnID, insertID int) TxnControlCounters {
	return &standardTxnControlCounters{
		genID:     genID,
		sessionID: ssnID,
		txnID:     txnID,
		insertID:  insertID,
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
	return tc.insertID
}

func (tc *standardTxnControlCounters) Copy(input TxnControlCounters) TxnControlCounters {
	tc.genID = input.GetGenID()
	tc.insertID = input.GetInsertID()
	tc.sessionID = input.GetSessionID()
	tc.txnID = input.GetTxnID()
	return tc
}

func (tc *standardTxnControlCounters) Clone() TxnControlCounters {
	return &standardTxnControlCounters{
		genID:           tc.genID,
		sessionID:       tc.sessionID,
		txnID:           tc.txnID,
		insertID:        tc.insertID,
		requestEncoding: tc.requestEncoding,
	}
}

func (tc *standardTxnControlCounters) CloneAndIncrementInsertID() TxnControlCounters {
	return &standardTxnControlCounters{
		genID:           tc.genID,
		sessionID:       tc.sessionID,
		txnID:           tc.txnID,
		insertID:        tc.insertID + 1,
		requestEncoding: tc.requestEncoding,
	}
}

func (tc *standardTxnControlCounters) SetTxnID(ti int) {
	tc.txnID = ti
}
