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

var (
	_ HTTPElement = &standardHTTPElement{}
)

type HTTPElement interface {
	GetName() string
	GetType() HTTPElementType
	SetTransformer(transformer func(interface{}) (interface{}, error))
	Transformer(interface{}) (interface{}, error)
	IsTransformerPresent() bool
}

type standardHTTPElement struct {
	httpElemType HTTPElementType
	name         string
	transformer  func(interface{}) (interface{}, error)
}

func (he *standardHTTPElement) GetName() string {
	return he.name
}

func (he *standardHTTPElement) GetType() HTTPElementType {
	return he.httpElemType
}

func (he *standardHTTPElement) IsTransformerPresent() bool {
	return he.transformer != nil
}

func (he *standardHTTPElement) Transformer(input interface{}) (interface{}, error) {
	if he.transformer == nil {
		return nil, fmt.Errorf("nil transformer disallowed")
	}
	return he.transformer(input)
}

func (he *standardHTTPElement) SetTransformer(transformer func(interface{}) (interface{}, error)) {
	he.transformer = transformer
}

func NewHTTPElement(httpElemType HTTPElementType, name string) HTTPElement {
	return &standardHTTPElement{
		httpElemType: httpElemType,
		name:         name,
	}
}
