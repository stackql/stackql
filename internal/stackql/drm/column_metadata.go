package drm

import (
	"reflect"

	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/parserutil"
)

var (
	_ ColumnMetadata = &standardColumnMetadata{}
)

type ColumnMetadata interface {
	GetColumn() openapistackql.ColumnDescriptor
	GetIdentifier() string
	GetName() string
	GetRelationalType() string
	GetType() string
}

type standardColumnMetadata struct {
	coupling dto.DRMCoupling
	column   openapistackql.ColumnDescriptor
}

func (cd *standardColumnMetadata) GetColumn() openapistackql.ColumnDescriptor {
	return cd.column
}

func (cd *standardColumnMetadata) GetName() string {
	return cd.column.Name
}

func (cd *standardColumnMetadata) GetIdentifier() string {
	return cd.column.GetIdentifier()
}

func (cd *standardColumnMetadata) GetType() string {
	if cd.column.Schema != nil {
		return cd.column.Schema.Type
	}
	return parserutil.ExtractStringRepresentationOfValueColumn(cd.column.Val)
}

func (cd *standardColumnMetadata) GetRelationalType() string {
	return cd.coupling.RelationalType
}

func NewColDescriptor(col openapistackql.ColumnDescriptor, relTypeStr string) ColumnMetadata {
	return &standardColumnMetadata{
		coupling: dto.DRMCoupling{RelationalType: relTypeStr, GolangKind: reflect.String},
		column:   col,
	}
}
