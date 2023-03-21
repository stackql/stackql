package internaldto

import (
	"reflect"

	"github.com/lib/pq/oid"
	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql/internal/stackql/parserutil"
)

var (
	_ ColumnMetadata = &standardColumnMetadata{}
)

type standardColumnMetadata struct {
	coupling DRMCoupling
	column   openapistackql.ColumnDescriptor
}

func (cd *standardColumnMetadata) GetColumnOID() oid.Oid {
	return cd.getOidForSchema(cd.column.GetSchema())
}

func (cd *standardColumnMetadata) GetName() string {
	return cd.column.GetName()
}

func (cd *standardColumnMetadata) GetDecorated() string {
	return cd.column.GetDecoratedCol()
}

func (cd *standardColumnMetadata) GetIdentifier() string {
	return cd.column.GetIdentifier()
}

func (cd *standardColumnMetadata) GetType() string {
	if cd.column.GetSchema() != nil {
		return cd.column.GetSchema().GetType()
	}
	return parserutil.ExtractStringRepresentationOfValueColumn(cd.column.GetVal())
}

func (cd *standardColumnMetadata) getOidForSchema(colSchema openapistackql.Schema) oid.Oid {
	return getOidForSchema(colSchema)
}

func GetOidForSchema(colSchema openapistackql.Schema) oid.Oid {
	return getOidForSchema(colSchema)
}

func getOidForSchema(colSchema openapistackql.Schema) oid.Oid {
	if colSchema == nil {
		return oid.T_text
	}
	switch colSchema.GetType() {
	case "object", "array":
		return oid.T_text
	// case "integer":
	// 	return oid.T_numeric
	case "boolean", "bool":
		return oid.T_text
	case "number":
		return oid.T_numeric
	default:
		return oid.T_text
	}
}

func (cd *standardColumnMetadata) GetRelationalType() string {
	return cd.coupling.GetRelationalType()
}

func NewColDescriptor(col openapistackql.ColumnDescriptor, relTypeStr string) ColumnMetadata {
	return &standardColumnMetadata{
		coupling: NewDRMCoupling(relTypeStr, reflect.String),
		column:   col,
	}
}
