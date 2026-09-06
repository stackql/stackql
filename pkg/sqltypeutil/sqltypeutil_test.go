package sqltypeutil_test

import (
	"strings"
	"testing"

	"github.com/stackql/stackql/pkg/sqltypeutil"
)

func TestInterfaceToSQLType_String(t *testing.T) {
	val, err := sqltypeutil.InterfaceToSQLType("hello")
	if err != nil {
		t.Fatalf("InterfaceToSQLType(string) error = %v", err)
	}
	if val.ToString() != "hello" {
		t.Errorf("InterfaceToSQLType(string).ToString() = %q, want 'hello'", val.ToString())
	}
}

func TestInterfaceToSQLType_EmptyString(t *testing.T) {
	val, err := sqltypeutil.InterfaceToSQLType("")
	if err != nil {
		t.Fatalf("InterfaceToSQLType(empty string) error = %v", err)
	}
	if val.ToString() != "" {
		t.Errorf("InterfaceToSQLType(empty string).ToString() = %q, want ''", val.ToString())
	}
}

func TestInterfaceToSQLType_BoolTrue(t *testing.T) {
	val, err := sqltypeutil.InterfaceToSQLType(true)
	if err != nil {
		t.Fatalf("InterfaceToSQLType(true) error = %v", err)
	}
	if val.ToString() != "1" {
		t.Errorf("InterfaceToSQLType(true).ToString() = %q, want '1'", val.ToString())
	}
}

func TestInterfaceToSQLType_BoolFalse(t *testing.T) {
	val, err := sqltypeutil.InterfaceToSQLType(false)
	if err != nil {
		t.Fatalf("InterfaceToSQLType(false) error = %v", err)
	}
	if val.ToString() != "0" {
		t.Errorf("InterfaceToSQLType(false).ToString() = %q, want '0'", val.ToString())
	}
}

func TestInterfaceToSQLType_Int64(t *testing.T) {
	val, err := sqltypeutil.InterfaceToSQLType(int64(12345678901234))
	if err != nil {
		t.Fatalf("InterfaceToSQLType(int64) error = %v", err)
	}
	if val.ToString() != "12345678901234" {
		t.Errorf("InterfaceToSQLType(int64).ToString() = %q, want '12345678901234'", val.ToString())
	}
}

func TestInterfaceToSQLType_Float64(t *testing.T) {
	val, err := sqltypeutil.InterfaceToSQLType(float64(3.14))
	if err != nil {
		t.Fatalf("InterfaceToSQLType(float64) error = %v", err)
	}
	if val.ToString() != "3.14" {
		t.Errorf("InterfaceToSQLType(float64).ToString() = %q, want '3.14'", val.ToString())
	}
}

func TestInterfaceToSQLType_Nil(t *testing.T) {
	val, err := sqltypeutil.InterfaceToSQLType(nil)
	if err != nil {
		t.Fatalf("InterfaceToSQLType(nil) error = %v", err)
	}
	if !val.IsNull() {
		t.Errorf("InterfaceToSQLType(nil).IsNull() = false, want true")
	}
}

func TestInterfaceToSQLType_Bytes(t *testing.T) {
	val, err := sqltypeutil.InterfaceToSQLType([]byte("binary data"))
	if err != nil {
		t.Fatalf("InterfaceToSQLType([]byte) error = %v", err)
	}
	if val.ToString() != "binary data" {
		t.Errorf("InterfaceToSQLType([]byte).ToString() = %q, want 'binary data'", val.ToString())
	}
}

func TestInterfaceToSQLType_SpecialChars(t *testing.T) {
	val, err := sqltypeutil.InterfaceToSQLType("hello\nworld\ttab")
	if err != nil {
		t.Fatalf("InterfaceToSQLType(special chars) error = %v", err)
	}
	if val.ToString() != "hello\nworld\ttab" {
		t.Errorf("InterfaceToSQLType(special chars).ToString() = %q, want 'hello\\nworld\\ttab'", val.ToString())
	}
}

func TestInterfaceToSQLType_Unicode(t *testing.T) {
	val, err := sqltypeutil.InterfaceToSQLType("こんにちは世界")
	if err != nil {
		t.Fatalf("InterfaceToSQLType(unicode) error = %v", err)
	}
	if val.ToString() != "こんにちは世界" {
		t.Errorf("InterfaceToSQLType(unicode).ToString() = %q, want 'こんにちは世界'", val.ToString())
	}
}

func TestInterfaceToSQLType_UnsupportedType(t *testing.T) {
	// plain int is not supported, falls through to InterfaceToValue which returns error
	_, err := sqltypeutil.InterfaceToSQLType(int(42))
	if err == nil {
		t.Error("InterfaceToSQLType(int) expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected type int") {
		t.Errorf("unexpected error message: %v", err)
	}
}
