package dto

import (
	"github.com/stackql/stackql/internal/pkg/txncounter"
)

type TxnControlCounters struct {
	GenId, SessionId, TxnId, InsertId, DiscoveryGenerationId int
}

func NewTxnControlCounters(txnCtrMgr *txncounter.TxnCounterManager, discoveryGenerationID int) *TxnControlCounters {
	return &TxnControlCounters{
		GenId:                 txnCtrMgr.GetCurrentGenerationId(),
		SessionId:             txnCtrMgr.GetCurrentSessionId(),
		TxnId:                 txnCtrMgr.GetNextTxnId(),
		InsertId:              txnCtrMgr.GetNextInsertId(),
		DiscoveryGenerationId: discoveryGenerationID,
	}
}

func (tc *TxnControlCounters) CloneWithDiscoGenID(discoveryGenerationID int) *TxnControlCounters {
	return &TxnControlCounters{
		GenId:                 tc.GenId,
		SessionId:             tc.SessionId,
		TxnId:                 tc.TxnId,
		InsertId:              tc.InsertId,
		DiscoveryGenerationId: discoveryGenerationID,
	}
}

func (tc *TxnControlCounters) CloneAndIncrementInsertID() *TxnControlCounters {
	return &TxnControlCounters{
		GenId:                 tc.GenId,
		SessionId:             tc.SessionId,
		TxnId:                 tc.TxnId,
		InsertId:              tc.InsertId + 1,
		DiscoveryGenerationId: tc.DiscoveryGenerationId,
	}
}

func (tc *TxnControlCounters) SetTxnId(ti int) {
	tc.TxnId = ti
}
