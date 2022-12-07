package relationaldto

import "github.com/stackql/stackql/internal/stackql/internaldto"

type standardRelationalTable struct {
	alias       string
	name        string
	baseName    string
	discoveryID int
	hIDs        internaldto.HeirarchyIdentifiers
	viewDTO     internaldto.ViewDTO
	columns     []RelationalColumn
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

func (rt *standardRelationalTable) GetColumns() []RelationalColumn {
	return rt.columns
}

func (rt *standardRelationalTable) PushBackColumn(col RelationalColumn) {
	rt.columns = append(rt.columns, col)
}
