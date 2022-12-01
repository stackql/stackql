package relationaldto

import (
	"github.com/stackql/stackql/internal/stackql/dto"
)

type standardRelationalView struct {
	alias       string
	name        string
	baseName    string
	discoveryID int
	hIDs        dto.HeirarchyIdentifiers
	columns     []RelationalColumn
}

func (rt *standardRelationalView) GetName() (string, error) {
	return rt.name, nil
}

func (rt *standardRelationalView) IsView() bool {
	return true
}

func (rt *standardRelationalView) GetBaseName() string {
	return rt.baseName
}

func (rt *standardRelationalView) GetAlias() string {
	return rt.alias
}

func (rt *standardRelationalView) WithAlias(alias string) RelationalTable {
	rt.alias = alias
	return rt
}

func (rt *standardRelationalView) GetColumns() []RelationalColumn {
	return rt.columns
}

func (rt *standardRelationalView) PushBackColumn(col RelationalColumn) {
	rt.columns = append(rt.columns, col)
}
