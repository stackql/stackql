//nolint:dupl // TODO: refactor
package internaldto

import (
	"github.com/stackql/stackql/internal/stackql/typing"
)

var (
	_ RelationDTO = &standardPhysicalTableDTO{}
)

func NewPhysicalTableDTO(viewName, rawViewQuery string) RelationDTO {
	return &standardPhysicalTableDTO{
		viewName:     viewName,
		rawViewQuery: rawViewQuery,
	}
}

type standardPhysicalTableDTO struct {
	rawViewQuery string
	viewName     string
	columns      []typing.RelationalColumn
}

func (v *standardPhysicalTableDTO) GetRawQuery() string {
	return v.rawViewQuery
}

func (v *standardPhysicalTableDTO) GetName() string {
	return v.viewName
}

func (v *standardPhysicalTableDTO) IsMaterialized() bool {
	return false
}

func (v *standardPhysicalTableDTO) IsTable() bool {
	return true
}

func (v *standardPhysicalTableDTO) GetColumns() []typing.RelationalColumn {
	return v.columns
}

func (v *standardPhysicalTableDTO) SetColumns(columns []typing.RelationalColumn) {
	v.columns = columns
}
