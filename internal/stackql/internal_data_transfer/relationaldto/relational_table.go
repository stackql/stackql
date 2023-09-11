package relationaldto

import (
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/typing"
)

var (
	_ RelationalTable = &standardRelationalTable{}
)

type RelationalTable interface {
	GetAlias() string
	GetBaseName() string
	GetColumns() []typing.RelationalColumn
	GetName() (string, error)
	GetView() (internaldto.RelationDTO, bool)
	PushBackColumn(typing.RelationalColumn)
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

func NewRelationalView(viewDTO internaldto.RelationDTO) RelationalTable {
	return &standardRelationalTable{
		viewDTO: viewDTO,
	}
}
