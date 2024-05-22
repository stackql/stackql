//nolint:dupl,nolintlint // TODO: refactor
package internaldto

import (
	"github.com/stackql/stackql/internal/stackql/typing"
)

var (
	_ RelationDTO = &standardPhysicalTableDTO{}
)

func NewPhysicalTableDTO(viewName, rawViewQuery, namespace string) RelationDTO {
	return &standardPhysicalTableDTO{
		viewName:     viewName,
		rawViewQuery: rawViewQuery,
		namespace:    namespace,
	}
}

type standardPhysicalTableDTO struct {
	rawViewQuery string
	viewName     string
	namespace    string
	columns      []typing.RelationalColumn
}

func (v *standardPhysicalTableDTO) GetRawQuery() string {
	return v.rawViewQuery
}

func (v *standardPhysicalTableDTO) MatchOnParams(map[string]any) (RelationDTO, bool) {
	return v, true
}

func (v *standardPhysicalTableDTO) Next() (RelationDTO, bool) {
	return nil, false
}

func (v *standardPhysicalTableDTO) WithNext(_ RelationDTO) RelationDTO {
	// v.next = next
	return v
}

func (v *standardPhysicalTableDTO) WithRequiredParams(_ map[string]any) RelationDTO {
	return v
}

func (v *standardPhysicalTableDTO) GetName() string {
	return v.viewName
}

func (v *standardPhysicalTableDTO) GetNamespace() string {
	return v.namespace
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
