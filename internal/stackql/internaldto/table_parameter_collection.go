package internaldto

import "github.com/stackql/stackql/internal/stackql/suffix"

var (
	_ TableParameterCollection = &standardTableParameterCollection{}
)

type TableParameterCollection interface {
	GetOptionalParams() suffix.ParameterSuffixMap
	GetRemainingRequiredParams() suffix.ParameterSuffixMap
	GetRequiredParams() suffix.ParameterSuffixMap
}

func NewTableParameterCollection(requiredParams, optionalParams, remainingRequiredParameters suffix.ParameterSuffixMap) TableParameterCollection {
	return &standardTableParameterCollection{
		requiredParams:              requiredParams,
		optionalParams:              optionalParams,
		remainingRequiredParameters: remainingRequiredParameters,
	}
}

type standardTableParameterCollection struct {
	requiredParams              suffix.ParameterSuffixMap
	optionalParams              suffix.ParameterSuffixMap
	remainingRequiredParameters suffix.ParameterSuffixMap
}

func (pc *standardTableParameterCollection) GetRequiredParams() suffix.ParameterSuffixMap {
	return pc.requiredParams
}

func (pc *standardTableParameterCollection) GetRemainingRequiredParams() suffix.ParameterSuffixMap {
	return pc.remainingRequiredParameters
}

func (pc *standardTableParameterCollection) GetOptionalParams() suffix.ParameterSuffixMap {
	return pc.optionalParams
}
