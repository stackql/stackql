package internaldto

var (
	_ ViewDTO = &standardViewDTO{}
)

func NewViewDTO(viewName, rawViewQuery string) ViewDTO {
	return &standardViewDTO{
		viewName:     viewName,
		rawViewQuery: rawViewQuery,
	}
}

type ViewDTO interface {
	GetRawQuery() string
	GetName() string
}

type standardViewDTO struct {
	rawViewQuery string
	viewName     string
}

func (v *standardViewDTO) GetRawQuery() string {
	return v.rawViewQuery
}

func (v *standardViewDTO) GetName() string {
	return v.viewName
}
