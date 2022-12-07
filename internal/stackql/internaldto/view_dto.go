package internaldto

var (
	_ ViewDTO = &standardViewDTO{}
)

func NewViewDTO(rawViewQuery string) ViewDTO {
	return &standardViewDTO{
		rawViewQuery: rawViewQuery,
	}
}

type ViewDTO interface {
	GetRawQuery() string
}

type standardViewDTO struct {
	rawViewQuery string
}

func (v *standardViewDTO) GetRawQuery() string {
	return v.rawViewQuery
}
