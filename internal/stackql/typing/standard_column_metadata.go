package typing

import (
	"reflect"

	"github.com/lib/pq/oid"
	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/any-sdk/public/formulation"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/parserutil"
)

var (
	_ ColumnMetadata = &standardColumnMetadata{}
)

type standardColumnMetadata struct {
	coupling ORMCoupling
	column   formulation.ColumnDescriptor
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

func (cd *standardColumnMetadata) getOidForSchema(colSchema anysdk.Schema) oid.Oid {
	return getOidForSchema(colSchema)
}

func GetOidForSchema(colSchema anysdk.Schema) oid.Oid {
	return getOidForSchema(colSchema)
}

func GetOidForParserColType(col sqlparser.ColumnType) oid.Oid {
	return getOidForParserColType(col)
}

func getOidForParserColType(col sqlparser.ColumnType) oid.Oid {
	switch col.Type {
	case "int", "integer", "int2", "int4", "int8", "smallint",
		"bigint", "numeric", "decimal", "real", "float", "float4",
		"float8", "double", "double precision", "serial", "bigserial":
		return oid.T_numeric
	case "bool", "boolean":
		return oid.T_bool
	case "text", "varchar", "char", "character", "character varying", "string":
		return oid.T_text
	case "date", "timestamp", "timestamp with time zone",
		"timestamp without time zone", "time", "time with time zone",
		"time without time zone":
		return oid.T_timestamp
	case "json", "jsonb":
		return oid.T_text
	default:
		return oid.T_text
	}
}

func getOidForSchema(colSchema anysdk.Schema) oid.Oid {
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

func NewColDescriptor(col formulation.ColumnDescriptor, relTypeStr string) ColumnMetadata {
	return &standardColumnMetadata{
		coupling: NewORMCoupling(relTypeStr, reflect.String),
		column:   col,
	}
}
