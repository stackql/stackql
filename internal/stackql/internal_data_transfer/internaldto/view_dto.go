package internaldto

import "github.com/stackql/stackql/internal/stackql/typing"

var (
	_ RelationDTO = &standardViewDTO{}
)

func NewViewDTO(viewName, rawViewQuery string) RelationDTO {
	return &standardViewDTO{
		viewName:     viewName,
		rawViewQuery: rawViewQuery,
	}
}

type RelationDTO interface {
	GetRawQuery() string
	GetName() string
	IsMaterialized() bool
	IsTable() bool
	GetNamespace() string
	GetColumns() []typing.RelationalColumn
	SetColumns(columns []typing.RelationalColumn)
	MatchOnParams(map[string]any) (RelationDTO, bool)
	WithRequiredParams(map[string]any) RelationDTO
	Next() (RelationDTO, bool)
	WithNext(RelationDTO) RelationDTO
}

type standardViewDTO struct {
	rawViewQuery   string
	viewName       string
	columns        []typing.RelationalColumn
	requiredParams map[string]any
	next           RelationDTO
}

func (v *standardViewDTO) GetRawQuery() string {
	return v.rawViewQuery
}

func (v *standardViewDTO) Next() (RelationDTO, bool) {
	return v.next, v.next != nil
}

func (v *standardViewDTO) WithNext(next RelationDTO) RelationDTO {
	v.next = next
	return v.next
}

func (v *standardViewDTO) WithRequiredParams(req map[string]any) RelationDTO {
	v.requiredParams = req
	return v
}

func (v *standardViewDTO) MatchOnParams(params map[string]any) (RelationDTO, bool) {
	if len(params) == 0 {
		if len(v.requiredParams) == 0 {
			return v, true
		}
		return nil, false
	}
	for k := range v.requiredParams {
		_, ok := params[k]
		if !ok {
			return nil, false
		}
	}
	return v, true
}

func (v *standardViewDTO) GetName() string {
	return v.viewName
}

func (v *standardViewDTO) GetNamespace() string {
	return ""
}

func (v *standardViewDTO) IsMaterialized() bool {
	return false
}

func (v *standardViewDTO) IsTable() bool {
	return false
}

func (v *standardViewDTO) GetColumns() []typing.RelationalColumn {
	return v.columns
}

func (v *standardViewDTO) SetColumns(columns []typing.RelationalColumn) {
	v.columns = columns
}
