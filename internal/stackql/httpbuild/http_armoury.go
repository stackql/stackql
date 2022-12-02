package httpbuild

import (
	"github.com/stackql/go-openapistackql/openapistackql"
)

var (
	_ HTTPArmoury = &standardHTTPArmoury{}
)

type HTTPArmoury interface {
	AddRequestParams(HTTPArmouryParameters)
	GetRequestParams() []HTTPArmouryParameters
	GetRequestSchema() *openapistackql.Schema
	GetResponseSchema() *openapistackql.Schema
	SetRequestParams([]HTTPArmouryParameters)
	SetRequestSchema(*openapistackql.Schema)
	SetResponseSchema(*openapistackql.Schema)
}

type standardHTTPArmoury struct {
	RequestParams  []HTTPArmouryParameters
	RequestSchema  *openapistackql.Schema
	ResponseSchema *openapistackql.Schema
}

func (ih *standardHTTPArmoury) GetRequestParams() []HTTPArmouryParameters {
	return ih.RequestParams
}

func (ih *standardHTTPArmoury) SetRequestParams(ps []HTTPArmouryParameters) {
	ih.RequestParams = ps
}

func (ih *standardHTTPArmoury) AddRequestParams(p HTTPArmouryParameters) {
	ih.RequestParams = append(ih.RequestParams, p)
}

func (ih *standardHTTPArmoury) SetRequestSchema(s *openapistackql.Schema) {
	ih.RequestSchema = s
}

func (ih *standardHTTPArmoury) SetResponseSchema(s *openapistackql.Schema) {
	ih.ResponseSchema = s
}

func (ih *standardHTTPArmoury) GetRequestSchema() *openapistackql.Schema {
	return ih.RequestSchema
}

func (ih *standardHTTPArmoury) GetResponseSchema() *openapistackql.Schema {
	return ih.ResponseSchema
}

func NewHTTPArmoury() HTTPArmoury {
	return &standardHTTPArmoury{}
}
