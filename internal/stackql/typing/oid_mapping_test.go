package typing //nolint:testpackage // tests exported functions in same package for simplicity

import (
	"testing"

	"github.com/lib/pq/oid"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

func TestGetOidForParserColType(t *testing.T) {
	tests := []struct {
		colType  string
		expected oid.Oid
	}{
		{"int", oid.T_numeric},
		{"integer", oid.T_numeric},
		{"int4", oid.T_numeric},
		{"int8", oid.T_numeric},
		{"bigint", oid.T_numeric},
		{"numeric", oid.T_numeric},
		{"decimal", oid.T_numeric},
		{"float", oid.T_numeric},
		{"float8", oid.T_numeric},
		{"double precision", oid.T_numeric},
		{"bool", oid.T_bool},
		{"boolean", oid.T_bool},
		{"text", oid.T_text},
		{"varchar", oid.T_text},
		{"string", oid.T_text},
		{"timestamp", oid.T_timestamp},
		{"json", oid.T_text},
		{"jsonb", oid.T_text},
		{"uuid", oid.T_text},
	}
	for _, tt := range tests {
		t.Run(tt.colType, func(t *testing.T) {
			got := GetOidForParserColType(sqlparser.ColumnType{Type: tt.colType})
			if got != tt.expected {
				t.Errorf("GetOidForParserColType(%q) = %d, want %d", tt.colType, got, tt.expected)
			}
		})
	}
}

func TestGetOidForSchemaNil(t *testing.T) {
	got := GetOidForSchema(nil)
	if got != oid.T_text {
		t.Errorf("GetOidForSchema(nil) = %d, want %d (T_text)", got, oid.T_text)
	}
}
