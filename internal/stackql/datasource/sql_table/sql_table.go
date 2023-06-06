package sql_table //nolint:revive,stylecheck // decent package name

import (
	"github.com/stackql/stackql/internal/stackql/symtab"
	"github.com/stackql/stackql/internal/stackql/typing"
)

type SQLTable interface {
	GetColumns() []typing.RelationalColumn
	GetSymTab() symtab.SymTab
}

type standardSQLTable struct {
	symTab symtab.SymTab
	colz   []typing.RelationalColumn
}

func NewStandardSQLTable(_ []typing.RelationalColumn) (SQLTable, error) {
	rv := &standardSQLTable{
		symTab: symtab.NewHashMapTreeSymTab(),
	}
	return rv, nil
}

func (sqt *standardSQLTable) GetSymTab() symtab.SymTab {
	return sqt.symTab
}

func (sqt *standardSQLTable) GetColumns() []typing.RelationalColumn {
	return sqt.colz
}
