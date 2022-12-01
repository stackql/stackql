package dto

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
	genId, sessionId, txnId, insertId int
	tableName                         string
	requestEncoding                   []string
}

func NewTxnControlCounters(txnCtrMgr txncounter.TxnCounterManager) (TxnControlCounters, error) {
	if txnCtrMgr == nil {
		return &standardTxnControlCounters{}, nil
	}
	genId, err := txnCtrMgr.GetCurrentGenerationId()
	if err != nil {
		return nil, err
	}
	ssnId, err := txnCtrMgr.GetCurrentSessionId()
	if err != nil {
		return nil, err
	}
	txnId, err := txnCtrMgr.GetNextTxnId()
	if err != nil {
		return nil, err
	}
	insertId, err := txnCtrMgr.GetNextInsertId()
	if err != nil {
		return nil, err
	}
	return &standardTxnControlCounters{
		genId:     genId,
		sessionId: ssnId,
		txnId:     txnId,
		insertId:  insertId,
	}, nil
}

func NewTxnControlCountersFromVals(genId, ssnId, txnId, insertId int) TxnControlCounters {
	return &standardTxnControlCounters{
		genId:     genId,
		sessionId: ssnId,
		txnId:     txnId,
		insertId:  insertId,
	}
}

func (tc *standardTxnControlCounters) SetTxnID(tID int) {
	tc.txnId = tID
}

func (tc *standardTxnControlCounters) SetTableName(tn string) {
	tc.tableName = tn
}

func (tc *standardTxnControlCounters) GetGenID() int {
	return tc.genId
}

func (tc *standardTxnControlCounters) GetSessionID() int {
	return tc.sessionId
}

func (tc *standardTxnControlCounters) GetTxnID() int {
	return tc.txnId
}

func (tc *standardTxnControlCounters) GetInsertID() int {
	return tc.insertId
}

func (tc *standardTxnControlCounters) Copy(input TxnControlCounters) TxnControlCounters {
	tc.genId = input.GetGenID()
	tc.insertId = input.GetInsertID()
	tc.sessionId = input.GetSessionID()
	tc.txnId = input.GetTxnID()
	return tc
}

func (tc *standardTxnControlCounters) Clone() TxnControlCounters {
	return &standardTxnControlCounters{
		genId:           tc.genId,
		sessionId:       tc.sessionId,
		txnId:           tc.txnId,
		insertId:        tc.insertId,
		requestEncoding: tc.requestEncoding,
	}
}

func (tc *standardTxnControlCounters) CloneAndIncrementInsertID() TxnControlCounters {
	return &standardTxnControlCounters{
		genId:           tc.genId,
		sessionId:       tc.sessionId,
		txnId:           tc.txnId,
		insertId:        tc.insertId + 1,
		requestEncoding: tc.requestEncoding,
	}
}

func (tc *standardTxnControlCounters) SetTxnId(ti int) {
	tc.txnId = ti
}
