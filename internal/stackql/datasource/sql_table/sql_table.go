package sql_table //nolint:revive,stylecheck // decent package name

import (
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/relationaldto"
	"github.com/stackql/stackql/internal/stackql/symtab"
)

type SQLTable interface {
	GetColumns() []relationaldto.RelationalColumn
	GetSymTab() symtab.SymTab
}

type standardSQLTable struct {
	symTab symtab.SymTab
	colz   []relationaldto.RelationalColumn
}

func NewStandardSQLTable(_ []relationaldto.RelationalColumn) (SQLTable, error) {
	rv := &standardSQLTable{
		symTab: symtab.NewHashMapTreeSymTab(),
	}
	return rv, nil
}

func (sqt *standardSQLTable) GetSymTab() symtab.SymTab {
	return sqt.symTab
}

func (sqt *standardSQLTable) GetColumns() []relationaldto.RelationalColumn {
	return sqt.colz
}
