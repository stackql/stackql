package typing

import (
	"reflect"
)

type Config interface {
	GetGolangKind(discoType string) reflect.Kind
	GetGolangValue(discoType string) interface{}
	GetRelationalType(discoType string) string
}

func NewTypingConfig(sqlDialect string) (Config, error) {
	return newTypingConfig(sqlDialect)
}
