package typing //nolint:testpackage // tests unexported behaviour in same package for simplicity

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/lib/pq/oid"
	"github.com/stackql/stackql/internal/stackql/constants"
)

// ---- minimal fake database/sql driver, used solely to obtain *sql.ColumnType values.

const fakeDriverName = "typing_fake_driver"

var registerFakeDriver sync.Once //nolint:gochecknoglobals // test helper

type fakeDriver struct{}

func (fakeDriver) Open(string) (driver.Conn, error) { return fakeConn{}, nil }

type fakeConn struct{}

func (fakeConn) Prepare(query string) (driver.Stmt, error) { return fakeStmt{query: query}, nil }
func (fakeConn) Close() error                              { return nil }
func (fakeConn) Begin() (driver.Tx, error)                 { return nil, fmt.Errorf("not supported") }

type fakeStmt struct{ query string }

func (fakeStmt) Close() error  { return nil }
func (fakeStmt) NumInput() int { return -1 }
func (fakeStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, fmt.Errorf("not supported")
}
func (s fakeStmt) Query([]driver.Value) (driver.Rows, error) {
	return fakeRows{typeNames: strings.Split(s.query, ",")}, nil
}

type fakeRows struct{ typeNames []string }

func (r fakeRows) Columns() []string {
	cols := make([]string, len(r.typeNames))
	for i := range cols {
		cols[i] = fmt.Sprintf("c%d", i)
	}
	return cols
}
func (fakeRows) Close() error              { return nil }
func (fakeRows) Next([]driver.Value) error { return io.EOF }
func (r fakeRows) ColumnTypeDatabaseTypeName(i int) string {
	return r.typeNames[i]
}

// columnTypes returns one *sql.ColumnType per supplied database type name.
func columnTypes(t *testing.T, typeNames ...string) []*sql.ColumnType {
	t.Helper()
	registerFakeDriver.Do(func() { sql.Register(fakeDriverName, fakeDriver{}) })
	db, err := sql.Open(fakeDriverName, "")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	rows, err := db.QueryContext(context.Background(), strings.Join(typeNames, ","))
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	cts, err := rows.ColumnTypes()
	if err != nil {
		t.Fatalf("ColumnTypes: %v", err)
	}
	return cts
}

func mustConfig(t *testing.T, dialect string) Config {
	t.Helper()
	cfg, err := NewTypingConfig(dialect)
	if err != nil {
		t.Fatalf("NewTypingConfig(%q): %v", dialect, err)
	}
	return cfg
}

// ---- tests

func TestNewTypingConfigUnsupportedDialect(t *testing.T) {
	for _, dialect := range []string{"", "snowflake", "nonsense"} {
		cfg, err := NewTypingConfig(dialect)
		if err == nil || cfg != nil {
			t.Errorf("NewTypingConfig(%q) = (%v, %v), want error", dialect, cfg, err)
		}
	}
}

func TestGetRelationalTypeAndGolangKind(t *testing.T) {
	tests := []struct {
		dialect      string
		discoType    string
		expectedType string
		expectedKind reflect.Kind
	}{
		{constants.SQLDialectPostgres, "array", "text", reflect.Slice},
		{constants.SQLDialectPostgres, "boolean", "boolean", reflect.Bool},
		{constants.SQLDialectPostgres, "int", "bigint", reflect.Int64},
		{constants.SQLDialectPostgres, "integer", "bigint", reflect.Int64},
		{constants.SQLDialectPostgres, "object", "text", reflect.Map},
		{constants.SQLDialectPostgres, "string", "text", reflect.String},
		{constants.SQLDialectPostgres, "number", "numeric", reflect.Float64},
		{constants.SQLDialectPostgres, "numeric", "numeric", reflect.Float64},
		{constants.SQLDialectPostgres, "unmapped", "text", reflect.String},
		{constants.SQLDialectSQLite3, "int", "integer", reflect.Int},
		{constants.SQLDialectSQLite3, "integer", "integer", reflect.Int},
		{constants.SQLDialectSQLite3, "boolean", "boolean", reflect.Bool},
		{constants.SQLDialectSQLite3, "number", "text", reflect.String}, // not mapped for sqlite
		{constants.SQLDialectSQLite3, "unmapped", "text", reflect.String},
	}
	for _, tt := range tests {
		t.Run(tt.dialect+"/"+tt.discoType, func(t *testing.T) {
			cfg := mustConfig(t, tt.dialect)
			if got := cfg.GetRelationalType(tt.discoType); got != tt.expectedType {
				t.Errorf("GetRelationalType(%q) = %q, want %q", tt.discoType, got, tt.expectedType)
			}
			if got := cfg.GetGolangKind(tt.discoType); got != tt.expectedKind {
				t.Errorf("GetGolangKind(%q) = %v, want %v", tt.discoType, got, tt.expectedKind)
			}
		})
	}
}

func TestGetGolangValue(t *testing.T) {
	tests := []struct {
		dialect   string
		discoType string
		expected  any
	}{
		{constants.SQLDialectPostgres, "string", &sql.NullString{}},
		{constants.SQLDialectPostgres, "object", &sql.NullString{}},
		{constants.SQLDialectPostgres, "array", &sql.NullString{}},
		{constants.SQLDialectPostgres, "boolean", &sql.NullBool{}},
		{constants.SQLDialectPostgres, "int", &sql.NullInt64{}},
		{constants.SQLDialectPostgres, "number", &sql.NullFloat64{}},
		{constants.SQLDialectPostgres, "unmapped", &sql.NullString{}},
		{constants.SQLDialectSQLite3, "int", &sql.NullInt64{}},
		{constants.SQLDialectSQLite3, "boolean", &sql.NullBool{}},
	}
	for _, tt := range tests {
		t.Run(tt.dialect+"/"+tt.discoType, func(t *testing.T) {
			got := mustConfig(t, tt.dialect).GetGolangValue(tt.discoType)
			if reflect.TypeOf(got) != reflect.TypeOf(tt.expected) {
				t.Errorf("GetGolangValue(%q) = %T, want %T", tt.discoType, got, tt.expected)
			}
		})
	}
}

func TestGetGolangValueUnhandledKindFallsBackToString(t *testing.T) {
	tc := &genericTypingConfig{
		typeMappings:          map[string]ORMCoupling{"weird": NewORMCoupling("text", reflect.Complex128)},
		defaultRelationalType: "text",
		defaultGolangKind:     reflect.String,
	}
	if got := tc.GetGolangValue("weird"); reflect.TypeOf(got) != reflect.TypeOf(&sql.NullString{}) {
		t.Errorf("GetGolangValue = %T, want *sql.NullString", got)
	}
	// reflect.Array is handled explicitly
	tc.typeMappings["arr"] = NewORMCoupling("text", reflect.Array)
	if got := tc.GetGolangValue("arr"); reflect.TypeOf(got) != reflect.TypeOf(&sql.NullString{}) {
		t.Errorf("GetGolangValue = %T, want *sql.NullString", got)
	}
}

func TestGetDefaultOID(t *testing.T) {
	if got := mustConfig(t, constants.SQLDialectPostgres).GetDefaultOID(); got != oid.T_text {
		t.Errorf("GetDefaultOID() = %d, want %d", got, oid.T_text)
	}
}

func TestExtractFromGolangValue(t *testing.T) {
	cfg := mustConfig(t, constants.SQLDialectPostgres)
	tests := []struct {
		name     string
		input    any
		expected any
	}{
		{"nil", nil, nil},
		{"valid string", &sql.NullString{String: "x", Valid: true}, "x"},
		{"null string", &sql.NullString{}, nil},
		{"valid bool", &sql.NullBool{Bool: true, Valid: true}, true},
		{"null bool", &sql.NullBool{}, nil},
		{"valid int64", &sql.NullInt64{Int64: 7, Valid: true}, int64(7)},
		{"null int64", &sql.NullInt64{}, nil},
		{"valid float64", &sql.NullFloat64{Float64: 1.5, Valid: true}, 1.5},
		{"null float64", &sql.NullFloat64{}, nil},
		{"unsupported type", 42, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cfg.ExtractFromGolangValue(tt.input); got != tt.expected {
				t.Errorf("ExtractFromGolangValue(%v) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGetOidForSQLDatabaseTypeName(t *testing.T) {
	tests := []struct {
		typeName string
		expected oid.Oid
	}{
		{"object", oid.T_text},
		{"ARRAY", oid.T_text},
		{"boolean", oid.T_bool},
		{"BOOL", oid.T_bool},
		{"number", oid.T_numeric},
		{"int", oid.T_numeric},
		{"BIGINT", oid.T_numeric},
		{"smallint", oid.T_numeric},
		{"tinyint", oid.T_numeric},
		{"varchar", oid.T_text},
		{"", oid.T_text},
	}
	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			if got := getOidForSQLDatabaseTypeName(tt.typeName); got != tt.expected {
				t.Errorf("getOidForSQLDatabaseTypeName(%q) = %d, want %d", tt.typeName, got, tt.expected)
			}
		})
	}
}

func TestGetOidForSQLType(t *testing.T) {
	cfg := mustConfig(t, constants.SQLDialectSQLite3)
	if got := cfg.GetOidForSQLType(nil); got != oid.T_text {
		t.Errorf("GetOidForSQLType(nil) = %d, want %d", got, oid.T_text)
	}
	names := []string{"BOOL", "BIGINT", "TEXT"}
	expected := []oid.Oid{oid.T_bool, oid.T_numeric, oid.T_text}
	for i, ct := range columnTypes(t, names...) {
		if got := cfg.GetOidForSQLType(ct); got != expected[i] {
			t.Errorf("GetOidForSQLType(%s) = %d, want %d", names[i], got, expected[i])
		}
	}
}

func TestGetScannableObjectForNativeResult(t *testing.T) {
	cfg := mustConfig(t, constants.SQLDialectSQLite3)
	tests := []struct {
		typeName string
		expected any
	}{
		{"int", new(sql.NullInt32)},
		{"INT32", new(sql.NullInt32)},
		{"smallint", new(sql.NullInt32)},
		{"tinyint", new(sql.NullInt32)},
		{"uint", new(sql.NullInt64)},
		{"uint32", new(sql.NullInt64)},
		{"int64", new(sql.NullInt64)},
		{"BIGINT", new(sql.NullInt64)},
		{"numeric", new(sql.NullFloat64)},
		{"decimal", new(sql.NullFloat64)},
		{"float", new(sql.NullFloat64)},
		{"float32", new(sql.NullFloat64)},
		{"float64", new(sql.NullFloat64)},
		{"bool", new(sql.NullBool)},
		{"text", new(sql.NullString)},
		{"", new(sql.NullString)},
	}
	names := make([]string, len(tests))
	for i, tt := range tests {
		names[i] = tt.typeName
	}
	cts := columnTypes(t, names...)
	for i, tt := range tests {
		got := cfg.GetScannableObjectForNativeResult(cts[i])
		if reflect.TypeOf(got) != reflect.TypeOf(tt.expected) {
			t.Errorf("GetScannableObjectForNativeResult(%q) = %T, want %T", tt.typeName, got, tt.expected)
		}
	}
}

func TestGetPlaceholderColumn(t *testing.T) {
	cfg := mustConfig(t, constants.SQLDialectPostgres)
	col := cfg.GetPlaceholderColumn(nil, "colA", oid.T_numeric)
	assertPlaceholder(t, col, "colA", uint32(oid.T_numeric))
}

func TestGetPlaceholderColumnForNativeResult(t *testing.T) {
	cfg := mustConfig(t, constants.SQLDialectSQLite3)
	cts := columnTypes(t, "BOOL", "BIGINT")
	assertPlaceholder(t, cfg.GetPlaceholderColumnForNativeResult(nil, "b", cts[0]), "b", uint32(oid.T_bool))
	assertPlaceholder(t, cfg.GetPlaceholderColumnForNativeResult(nil, "n", cts[1]), "n", uint32(oid.T_numeric))
	assertPlaceholder(t, cfg.GetPlaceholderColumnForNativeResult(nil, "x", nil), "x", uint32(oid.T_text))
}

func assertPlaceholder(t *testing.T, col interface {
	GetName() string
	GetObjectID() uint32
	GetWidth() int16
	GetFormat() string
}, name string, objectID uint32,
) {
	t.Helper()
	if col.GetName() != name {
		t.Errorf("name = %q, want %q", col.GetName(), name)
	}
	if col.GetObjectID() != objectID {
		t.Errorf("object ID = %d, want %d", col.GetObjectID(), objectID)
	}
	if col.GetWidth() != 1024 {
		t.Errorf("width = %d, want 1024", col.GetWidth())
	}
	if col.GetFormat() != "TextFormat" {
		t.Errorf("format = %q, want TextFormat", col.GetFormat())
	}
}
