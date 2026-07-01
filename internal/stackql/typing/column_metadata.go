package typing

import (
	"github.com/lib/pq/oid"
)

type ColumnMetadata interface {
	GetColumnOID() oid.Oid
	GetIdentifier() string
	GetName() string
	// GetWireName returns the foreign-API property name the response data is keyed by.
	// It equals GetName unless the column carries a distinct wire name (e.g. a snake_case
	// display alias over a PascalCase wire field), and is used to extract response values.
	GetWireName() string
	GetDecorated() string
	GetRelationalType() string
	GetType() string
}
