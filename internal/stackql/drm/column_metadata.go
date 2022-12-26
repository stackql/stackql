package drm

import (
	"github.com/lib/pq/oid"
)

type ColumnMetadata interface {
	GetColumnOID() oid.Oid
	GetIdentifier() string
	GetName() string
	GetDecorated() string
	GetRelationalType() string
	GetType() string
}
