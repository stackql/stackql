package util

import (
	"github.com/stackql/go-openapistackql/openapistackql"
)

type Column interface {
	GetName() string
	GetSchema() *openapistackql.Schema
}

type simpleColumn struct {
	name   string
	schema *openapistackql.Schema
}

func newSimpleColumn(name string, schema *openapistackql.Schema) Column {
	return &simpleColumn{
		name:   name,
		schema: schema,
	}
}

func (sc simpleColumn) GetName() string {
	return sc.name
}

func (sc simpleColumn) GetSchema() *openapistackql.Schema {
	return sc.schema
}
