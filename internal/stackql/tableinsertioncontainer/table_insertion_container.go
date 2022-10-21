package tableinsertioncontainer

import (
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
)

var (
	_ TableInsertionContainer = &StandardTableInsertionContainer{}
)

type TableInsertionContainer interface {
	GetTableMetadata() *tablemetadata.ExtendedTableMetadata
	IsCountersSet() bool
	SetTableTxnCounters(string, *dto.TxnControlCounters) error
	GetTableTxnCounters() (string, *dto.TxnControlCounters)
}

type StandardTableInsertionContainer struct {
	tableName     string
	tm            *tablemetadata.ExtendedTableMetadata
	tcc           *dto.TxnControlCounters
	sqlEngine     sqlengine.SQLEngine
	isCountersSet bool
}

func (ic *StandardTableInsertionContainer) GetTableMetadata() *tablemetadata.ExtendedTableMetadata {
	return ic.tm
}

func (ic *StandardTableInsertionContainer) SetTableTxnCounters(tableName string, tcc *dto.TxnControlCounters) error {
	ic.tableName = tableName
	ic.tcc.GenId = tcc.GenId
	ic.tcc.SessionId = tcc.SessionId
	ic.tcc.InsertId = tcc.InsertId
	ic.tcc.TxnId = tcc.TxnId
	ic.tcc.TableName = tableName
	ic.tcc.RequestEncoding = tcc.RequestEncoding
	ic.isCountersSet = true
	return nil
}

func (ic *StandardTableInsertionContainer) GetTableTxnCounters() (string, *dto.TxnControlCounters) {
	return ic.tableName, ic.tcc
}

func (ic *StandardTableInsertionContainer) IsCountersSet() bool {
	return ic.isCountersSet
}

func NewTableInsertionContainer(tm *tablemetadata.ExtendedTableMetadata, sqlEngine sqlengine.SQLEngine) TableInsertionContainer {
	return &StandardTableInsertionContainer{
		tm:        tm,
		tcc:       &dto.TxnControlCounters{},
		sqlEngine: sqlEngine,
	}
}
