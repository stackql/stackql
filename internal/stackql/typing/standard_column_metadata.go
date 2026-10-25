package typing

import (
	"github.com/lib/pq/oid"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

type schemaType interface {
	GetType() string
}

func GetOidForSchema(colSchema schemaType) oid.Oid {
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

func getOidForSchema(colSchema schemaType) oid.Oid {
	if colSchema == nil {
		return oid.T_text
	}
	switch colSchema.GetType() {
	case "object", "array":
		return oid.T_text
	case "boolean", "bool":
		return oid.T_text
	case "number":
		return oid.T_numeric
	default:
		return oid.T_text
	}
}
