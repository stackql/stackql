package sqltable

import (
	"github.com/stackql/stackql/internal/stackql/symtab"
	"github.com/stackql/stackql/internal/stackql/typing"
)

type SQLTable interface {
	GetColumns() []typing.RelationalColumn
	GetSymTab() symtab.SymTab
}

type StandardSQLTable struct {
	symTab  symtab.SymTab
	columns []typing.RelationalColumn
}

func NewStandardSQLTable(relationalColumns []typing.RelationalColumn) (SQLTable, error) {
	copiedSlice := make([]typing.RelationalColumn, len(relationalColumns))
	copy(copiedSlice, relationalColumns)
	rv := &StandardSQLTable{
		symTab:  symtab.NewHashMapTreeSymTab(),
		columns: copiedSlice,
	}
	return rv, nil
}

func (sqt *StandardSQLTable) GetSymTab() symtab.SymTab {
	return sqt.symTab
}

func (sqt *StandardSQLTable) GetColumns() []typing.RelationalColumn {
	return sqt.columns
}
