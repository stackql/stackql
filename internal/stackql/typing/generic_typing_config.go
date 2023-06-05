package typing

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/stackql/stackql/internal/stackql/constants"
)

var (
	_ Config = &genericTypingConfig{}
)

func getPostgresTypeMappings() map[string]ORMCoupling {
	return map[string]ORMCoupling{
		"array":   NewORMCoupling("text", reflect.Slice),
		"boolean": NewORMCoupling("boolean", reflect.Bool),
		"int":     NewORMCoupling("bigint", reflect.Int64),
		"integer": NewORMCoupling("bigint", reflect.Int64),
		"object":  NewORMCoupling("text", reflect.Map),
		"string":  NewORMCoupling("text", reflect.String),
		"number":  NewORMCoupling("numeric", reflect.Float64),
		"numeric": NewORMCoupling("numeric", reflect.Float64),
	}
}

func getSQLiteTypeMappings() map[string]ORMCoupling {
	return map[string]ORMCoupling{
		"array":   NewORMCoupling("text", reflect.Slice),
		"boolean": NewORMCoupling("boolean", reflect.Bool),
		"int":     NewORMCoupling("integer", reflect.Int),
		"integer": NewORMCoupling("integer", reflect.Int),
		"object":  NewORMCoupling("text", reflect.Map),
		"string":  NewORMCoupling("text", reflect.String),
	}
}

func getTypeMappings(sqlDialect string) (map[string]ORMCoupling, error) {
	switch sqlDialect {
	case constants.SQLDialectPostgres:
		return getPostgresTypeMappings(), nil
	case constants.SQLDialectSQLite3:
		return getSQLiteTypeMappings(), nil
	default:
		return nil, fmt.Errorf("cannot support type mappings for sqlDialect = '%s'", sqlDialect)
	}
}

//nolint:goconst,unparam // let it ride
func getDefaultRelationalType(sqlDialect string) string {
	switch sqlDialect {
	case constants.SQLDialectPostgres:
		return "text"
	case constants.SQLDialectSQLite3:
		return "text"
	default:
		return "text"
	}
}

//nolint:unparam // let it ride
func getDefaultGolangKind(sqlDialect string) reflect.Kind {
	switch sqlDialect {
	case constants.SQLDialectPostgres:
		return reflect.String
	case constants.SQLDialectSQLite3:
		return reflect.String
	default:
		return reflect.String
	}
}

type genericTypingConfig struct {
	typeMappings          map[string]ORMCoupling
	defaultRelationalType string
	defaultGolangKind     reflect.Kind
}

func (tc *genericTypingConfig) GetRelationalType(discoType string) string {
	rv, ok := tc.typeMappings[discoType]
	if ok {
		return rv.GetRelationalType()
	}
	return tc.defaultRelationalType
}

func (tc *genericTypingConfig) getDefaultGolangValue() interface{} {
	return &sql.NullString{}
}

func (tc *genericTypingConfig) GetGolangValue(discoType string) interface{} {
	rv, ok := tc.typeMappings[discoType]
	if !ok {
		return tc.getDefaultGolangValue()
	}
	//nolint:exhaustive //TODO: address this
	switch rv.GetGolangKind() {
	case reflect.String:
		return &sql.NullString{}
	case reflect.Array:
		return &sql.NullString{}
	case reflect.Bool:
		return &sql.NullBool{}
	case reflect.Map:
		return &sql.NullString{}
	case reflect.Int, reflect.Int64:
		return &sql.NullInt64{}
	case reflect.Float64:
		return &sql.NullFloat64{}
	}
	return tc.getDefaultGolangValue()
}

func (tc *genericTypingConfig) GetGolangKind(discoType string) reflect.Kind {
	rv, ok := tc.typeMappings[discoType]
	if !ok {
		return tc.getDefaultGolangKind()
	}
	return rv.GetGolangKind()
}

func (tc *genericTypingConfig) getDefaultGolangKind() reflect.Kind {
	return tc.defaultGolangKind
}

func newTypingConfig(sqlDialect string) (Config, error) {
	typeMappings, err := getTypeMappings(sqlDialect)
	if err != nil {
		return nil, err
	}
	defaultRelationalType := getDefaultRelationalType(sqlDialect)
	defaultGolangKind := getDefaultGolangKind(sqlDialect)
	return &genericTypingConfig{
		typeMappings:          typeMappings,
		defaultRelationalType: defaultRelationalType,
		defaultGolangKind:     defaultGolangKind,
	}, nil
}
