package typing

import (
	"reflect"
	"strings"

	"github.com/lib/pq/oid"
)

var (
	_ ColumnMetadata = &relayedColumnMetadata{}
)

type relayedColumnMetadata struct {
	coupling ORMCoupling
	column   RelationalColumn
}

func (cd *relayedColumnMetadata) GetColumnOID() oid.Oid {
	if oID, ok := cd.column.GetOID(); ok {
		return oID
	}
	return cd.getOidForRelationalType(cd.coupling.GetRelationalType())
}

func (cd *relayedColumnMetadata) GetName() string {
	return cd.column.GetName()
}

func (cd *relayedColumnMetadata) GetDecorated() string {
	return cd.column.GetDecorated()
}

func (cd *relayedColumnMetadata) GetIdentifier() string {
	alias := cd.column.GetAlias()
	if alias != "" {
		return alias
	}
	return cd.column.GetName()
}

func (cd *relayedColumnMetadata) GetType() string {
	return cd.column.GetType()
}

func (cd *relayedColumnMetadata) GetRelationalType() string {
	return cd.coupling.GetRelationalType()
}

func (cd *relayedColumnMetadata) getOidForRelationalType(relType string) oid.Oid {
	relType = strings.ToLower(relType)
	switch relType {
	case "object", "array", "text":
		return oid.T_text
	// case "integer":
	// 	return oid.T_numeric
	case "boolean", "bool":
		return oid.T_text
	case "number", "decimal", "numeric", "real":
		return oid.T_numeric
	default:
		return oid.T_text
	}
}

func NewRelayedColDescriptor(col RelationalColumn, relTypeStr string) ColumnMetadata {
	return &relayedColumnMetadata{
		coupling: NewORMCoupling(relTypeStr, reflect.String),
		column:   col,
	}
}
