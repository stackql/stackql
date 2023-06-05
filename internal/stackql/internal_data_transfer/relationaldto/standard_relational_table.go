package relationaldto

import (
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/typing"
)

type standardRelationalTable struct {
	alias       string
	name        string
	baseName    string
	discoveryID int
	hIDs        internaldto.HeirarchyIdentifiers
	viewDTO     internaldto.ViewDTO
	columns     []typing.RelationalColumn
}

func (rt *standardRelationalTable) WithView(viewDTO internaldto.ViewDTO) RelationalTable {
	rt.viewDTO = viewDTO
	return rt
}

func (rt *standardRelationalTable) GetName() (string, error) {
	return rt.name, nil
}

func (rt *standardRelationalTable) GetBaseName() string {
	return rt.baseName
}

func (rt *standardRelationalTable) IsView() bool {
	return false
}

func (rt *standardRelationalTable) GetAlias() string {
	return rt.alias
}

func (rt *standardRelationalTable) WithAlias(alias string) RelationalTable {
	rt.alias = alias
	return rt
}

func (rt *standardRelationalTable) GetView() (internaldto.ViewDTO, bool) {
	return rt.viewDTO, rt.viewDTO != nil
}

func (rt *standardRelationalTable) GetColumns() []typing.RelationalColumn {
	return rt.columns
}

func (rt *standardRelationalTable) PushBackColumn(col typing.RelationalColumn) {
	rt.columns = append(rt.columns, col)
}
