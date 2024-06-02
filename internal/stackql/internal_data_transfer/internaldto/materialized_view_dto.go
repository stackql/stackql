//nolint:dupl,nolintlint // TODO: refactor
package internaldto

import (
	"github.com/stackql/stackql/internal/stackql/typing"
)

var (
	_ RelationDTO = &standardMaterializedViewDTO{}
)

func NewMaterializedViewDTO(viewName, rawViewQuery, namespace string) RelationDTO {
	return &standardMaterializedViewDTO{
		viewName:     viewName,
		rawViewQuery: rawViewQuery,
		namespace:    namespace,
	}
}

type standardMaterializedViewDTO struct {
	rawViewQuery string
	viewName     string
	namespace    string
	columns      []typing.RelationalColumn
}

func (v *standardMaterializedViewDTO) GetRawQuery() string {
	return v.rawViewQuery
}

func (v *standardMaterializedViewDTO) Next() (RelationDTO, bool) {
	return nil, false
}

func (v *standardMaterializedViewDTO) WithNext(_ RelationDTO) RelationDTO {
	// v.next = next
	return v
}

func (v *standardMaterializedViewDTO) WithRequiredParams(_ map[string]any) RelationDTO {
	return v
}

func (v *standardMaterializedViewDTO) MatchOnParams(map[string]any) (RelationDTO, bool) {
	return v, true
}

func (v *standardMaterializedViewDTO) GetName() string {
	return v.viewName
}

func (v *standardMaterializedViewDTO) IsMaterialized() bool {
	return true
}

func (v *standardMaterializedViewDTO) GetNamespace() string {
	return v.namespace
}

func (v *standardMaterializedViewDTO) IsTable() bool {
	return false
}

func (v *standardMaterializedViewDTO) GetColumns() []typing.RelationalColumn {
	return v.columns
}

func (v *standardMaterializedViewDTO) SetColumns(columns []typing.RelationalColumn) {
	v.columns = columns
}
