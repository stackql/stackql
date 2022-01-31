package dto

type TxnControlCounters struct {
	GenId, SessionId, TxnId, InsertId, DiscoveryGenerationId int
}

func (tc *TxnControlCounters) SetTxnId(ti int) {
	tc.TxnId = ti
}
