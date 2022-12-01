package internaldto

import (
	"fmt"
	"strings"
)

type HTTPElementType int

const (
	QueryParam HTTPElementType = iota
	PathParam
	Header
	BodyAttribute
	RequestString
	Error
	QueryParamStr    string = "query"
	PathParamStr     string = "path"
	HeaderStr        string = "header"
	BodyAttributeStr string = "body"
	RequestStringStr string = "request"
)

func ExtractHttpElement(s string) (HTTPElementType, error) {
	switch strings.ToLower(s) {
	case QueryParamStr:
		return QueryParam, nil
	case PathParamStr:
		return PathParam, nil
	case HeaderStr:
		return Header, nil
	case BodyAttributeStr:
		return BodyAttribute, nil
	case RequestStringStr:
		return RequestString, nil
	default:
		return Error, fmt.Errorf("cannot accomodate HTTP Element of type: '%s'", s)
	}
}

type HTTPElement struct {
	Type        HTTPElementType
	Name        string
	Transformer func(interface{}) (interface{}, error)
}
