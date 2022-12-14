package relationaldto

import (
	"github.com/stackql/stackql/internal/stackql/internaldto"
)

var (
	_ RelationalTable = &standardRelationalTable{}
)

type RelationalTable interface {
	GetAlias() string
	GetBaseName() string
	GetColumns() []RelationalColumn
	GetName() (string, error)
	GetView() (internaldto.ViewDTO, bool)
	PushBackColumn(RelationalColumn)
	WithAlias(alias string) RelationalTable
}

func NewRelationalTable(hIDs internaldto.HeirarchyIdentifiers, discoveryID int, name, baseName string) RelationalTable {
	return &standardRelationalTable{
		hIDs:        hIDs,
		name:        name,
		baseName:    baseName,
		discoveryID: discoveryID,
	}
}

func NewRelationalView(viewDTO internaldto.ViewDTO) RelationalTable {
	return &standardRelationalTable{
		viewDTO: viewDTO,
	}
}
