package internaldto

import (
	"reflect"
)

var (
	_ DRMCoupling = &standardDRMCoupling{}
)

type DRMCoupling interface {
	GetRelationalType() string
	GetGolangKind() reflect.Kind
}

type standardDRMCoupling struct {
	relationalType string
	golangKind     reflect.Kind
}

func (dc *standardDRMCoupling) GetRelationalType() string {
	return dc.relationalType
}

func (dc *standardDRMCoupling) GetGolangKind() reflect.Kind {
	return dc.golangKind
}

func NewDRMCoupling(relationalType string, golangKind reflect.Kind) DRMCoupling {
	return &standardDRMCoupling{
		relationalType: relationalType,
		golangKind:     golangKind,
	}
}
