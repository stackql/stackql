package tableinsertioncontainer

import (
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
)

var (
	_ TableInsertionContainer = &StandardTableInsertionContainer{}
)

type TableInsertionContainer interface {
	GetTableMetadata() *tablemetadata.ExtendedTableMetadata
	IsCountersSet() bool
	SetTableTxnCounters(string, *dto.TxnControlCounters)
	GetTableTxnCounters() (string, *dto.TxnControlCounters)
}

type StandardTableInsertionContainer struct {
	tableName     string
	tm            *tablemetadata.ExtendedTableMetadata
	tcc           *dto.TxnControlCounters
	isCountersSet bool
}

func (ic *StandardTableInsertionContainer) GetTableMetadata() *tablemetadata.ExtendedTableMetadata {
	return ic.tm
}

func (ic *StandardTableInsertionContainer) SetTableTxnCounters(tableName string, tcc *dto.TxnControlCounters) {
	ic.tableName = tableName
	ic.tcc.GenId = tcc.GenId
	ic.tcc.SessionId = tcc.SessionId
	ic.tcc.InsertId = tcc.InsertId
	ic.tcc.TxnId = tcc.TxnId
	ic.tcc.TableName = tableName
	ic.tcc.RequestEncoding = tcc.RequestEncoding
	ic.isCountersSet = true
}

func (ic *StandardTableInsertionContainer) GetTableTxnCounters() (string, *dto.TxnControlCounters) {
	return ic.tableName, ic.tcc
}

func (ic *StandardTableInsertionContainer) IsCountersSet() bool {
	return ic.isCountersSet
}

func NewTableInsertionContainer(tm *tablemetadata.ExtendedTableMetadata) TableInsertionContainer {
	return &StandardTableInsertionContainer{
		tm:  tm,
		tcc: &dto.TxnControlCounters{},
	}
}

func NewTableInsertionContainers(tms []*tablemetadata.ExtendedTableMetadata) []TableInsertionContainer {
	var rv []TableInsertionContainer
	for _, tm := range tms {
		rv = append(rv, NewTableInsertionContainer(tm))
	}
	return rv
}
