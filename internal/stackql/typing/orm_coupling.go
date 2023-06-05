package typing

import (
	"reflect"
)

var (
	_ ORMCoupling = &standardORMCoupling{}
)

type ORMCoupling interface {
	GetRelationalType() string
	GetGolangKind() reflect.Kind
}

type standardORMCoupling struct {
	relationalType string
	golangKind     reflect.Kind
}

func (dc *standardORMCoupling) GetRelationalType() string {
	return dc.relationalType
}

func (dc *standardORMCoupling) GetGolangKind() reflect.Kind {
	return dc.golangKind
}

func NewORMCoupling(relationalType string, golangKind reflect.Kind) ORMCoupling {
	return &standardORMCoupling{
		relationalType: relationalType,
		golangKind:     golangKind,
	}
}
