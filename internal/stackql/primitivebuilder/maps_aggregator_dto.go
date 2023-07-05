package primitivebuilder

var (
	_ mapsAggregatorDTO = (*standardMapsAggregatorDTO)(nil)
)

type mapsAggregatorDTO interface {
	getParameterMap() map[int]map[string]interface{}
	// getInputMap() map[int]map[int]interface{}
}

type standardMapsAggregatorDTO struct {
	parameterMap map[int]map[string]interface{}
	inputMap     map[int]map[int]interface{}
}

func (dto *standardMapsAggregatorDTO) getParameterMap() map[int]map[string]interface{} {
	return dto.parameterMap
}

// func (dto *standardMapsAggregatorDTO) getInputMap() map[int]map[int]interface{} {
// 	return dto.inputMap
// }

func newMapsAggregatorDTO(
	parameterMap map[int]map[string]interface{},
	inputMap map[int]map[int]interface{},
) mapsAggregatorDTO {
	return &standardMapsAggregatorDTO{
		parameterMap: parameterMap,
		inputMap:     inputMap,
	}
}
