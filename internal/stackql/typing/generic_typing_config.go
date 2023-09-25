package typing

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/lib/pq/oid"
	"github.com/stackql/psql-wire/pkg/sqldata"
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

func (tc *genericTypingConfig) GetOidForSQLType(colType *sql.ColumnType) oid.Oid {
	return getOidForSQLType(colType)
}

func (tc *genericTypingConfig) getDefaultGolangKind() reflect.Kind {
	return tc.defaultGolangKind
}

func (tc *genericTypingConfig) GetPlaceholderColumn(
	table sqldata.ISQLTable, colName string, colOID oid.Oid) sqldata.ISQLColumn {
	return tc.getPlaceholderColumn(table, colName, colOID)
}

func (tc *genericTypingConfig) getPlaceholderColumn(
	table sqldata.ISQLTable, colName string, colOID oid.Oid) sqldata.ISQLColumn {
	return sqldata.NewSQLColumn(
		table,
		colName,
		0,
		uint32(colOID),
		1024, //nolint:gomnd // TODO: refactor
		0,
		"TextFormat",
	)
}

func (tc *genericTypingConfig) GetPlaceholderColumnForNativeResult(
	table sqldata.ISQLTable,
	colName string, colSchema *sql.ColumnType) sqldata.ISQLColumn {
	return tc.getPlaceholderColumnForNativeResult(
		table,
		colName,
		colSchema,
	)
}

func (tc *genericTypingConfig) GetDefaultOID() oid.Oid {
	return oid.T_text
}

func (tc *genericTypingConfig) getPlaceholderColumnForNativeResult(
	table sqldata.ISQLTable,
	colName string, colSchema *sql.ColumnType) sqldata.ISQLColumn {
	return sqldata.NewSQLColumn(
		table,
		colName,
		0,
		uint32(tc.GetOidForSQLType(colSchema)),
		1024, //nolint:gomnd // TODO: refactor
		0,
		"TextFormat",
	)
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

//nolint:goconst // defer cleanup
func getOidForSQLDatabaseTypeName(typeName string) oid.Oid {
	typeNameLowered := strings.ToLower(typeName)
	switch strings.ToLower(typeNameLowered) {
	case "object", "array":
		return oid.T_text
	case "boolean", "bool":
		return oid.T_bool
	case "number", "int", "bigint", "smallint", "tinyint":
		return oid.T_numeric
	default:
		return oid.T_text
	}
}

func getOidForSQLType(colType *sql.ColumnType) oid.Oid {
	if colType == nil {
		return oid.T_text
	}
	return getOidForSQLDatabaseTypeName(colType.DatabaseTypeName())
}

func (tc *genericTypingConfig) ExtractFromGolangValue(val interface{}) interface{} {
	return tc.extractFromGolangValue(val)
}

func (tc *genericTypingConfig) extractFromGolangValue(val interface{}) interface{} {
	if val == nil {
		return nil
	}
	var retVal interface{}
	//nolint:gocritic // TODO: fix
	switch v := val.(type) {
	case *sql.NullString:
		retVal, _ = (*v).Value()
	case *sql.NullBool:
		retVal, _ = (*v).Value()
	case *sql.NullInt64:
		retVal, _ = (*v).Value()
	case *sql.NullFloat64:
		retVal, _ = (*v).Value()
	}
	return retVal
}

func (tc *genericTypingConfig) GetScannableObjectForNativeResult(colSchema *sql.ColumnType) any {
	return tc.getScannableObjectForNativeResult(colSchema)
}

func (tc *genericTypingConfig) getScannableObjectForNativeResult(colSchema *sql.ColumnType) any {
	switch strings.ToLower(colSchema.DatabaseTypeName()) {
	case "int", "int32", "smallint", "tinyint":
		return new(sql.NullInt32)
	case "uint", "uint32":
		return new(sql.NullInt64)
	case "int64", "bigint":
		return new(sql.NullInt64)
	//nolint:goconst // let it ride
	case "numeric", "decimal", "float", "float32", "float64":
		return new(sql.NullFloat64)
	case "bool":
		return new(sql.NullBool)
	default:
		return new(sql.NullString)
	}
}
