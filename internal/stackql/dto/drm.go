package dto

import (
	"reflect"
)

type DRMCoupling struct {
	RelationalType string
	GolangKind     reflect.Kind
}
