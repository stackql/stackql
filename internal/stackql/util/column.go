package util

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stackql/go-openapistackql/openapistackql"
)

type Column interface {
	GetName() string
	GetSchema() openapistackql.Schema
	GetWidth() int
}

type simpleColumn struct {
	name   string
	schema openapistackql.Schema
}

func newSimpleColumn(name string, schema openapistackql.Schema) Column {
	return &simpleColumn{
		name:   name,
		schema: schema,
	}
}

func newSimpleStringColumn(name string, m openapistackql.OperationStore) Column {
	sc := openapi3.NewSchema()
	sc.Type = "string"
	return newSimpleColumn(name, openapistackql.NewSchema(
		sc,
		m.GetService(),
		"",
		"",
	),
	)
}

func (sc simpleColumn) GetName() string {
	return sc.name
}

func (sc simpleColumn) GetWidth() int {
	return -1
}

func (sc simpleColumn) GetSchema() openapistackql.Schema {
	return sc.schema
}
