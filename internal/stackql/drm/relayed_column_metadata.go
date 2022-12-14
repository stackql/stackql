package drm

import (
	"reflect"

	"github.com/lib/pq/oid"
	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql/internal/stackql/internaldto"
	"github.com/stackql/stackql/internal/stackql/parserutil"
)

var (
	_ ColumnMetadata = &relayedColumnMetadata{}
)

type relayedColumnMetadata struct {
	coupling internaldto.DRMCoupling
	column   openapistackql.ColumnDescriptor
}

func (cd *relayedColumnMetadata) GetColumnOID() oid.Oid {
	return oid.T_text
}

func (cd *relayedColumnMetadata) GetName() string {
	return cd.column.Name
}

func (cd *relayedColumnMetadata) GetIdentifier() string {
	return cd.column.GetIdentifier()
}

func (cd *relayedColumnMetadata) GetType() string {
	if cd.column.Schema != nil {
		return cd.column.Schema.Type
	}
	return parserutil.ExtractStringRepresentationOfValueColumn(cd.column.Val)
}

func (cd *relayedColumnMetadata) GetRelationalType() string {
	return cd.coupling.GetRelationalType()
}

func NewRelayedColDescriptor(col openapistackql.ColumnDescriptor, relTypeStr string) ColumnMetadata {
	return &relayedColumnMetadata{
		coupling: internaldto.NewDRMCoupling(relTypeStr, reflect.String),
		column:   col,
	}
}
