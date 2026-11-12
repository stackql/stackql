package typing //nolint:testpackage // tests unexported behaviour in same package for simplicity

import (
	"testing"

	"github.com/lib/pq/oid"
)

type fakeSchema struct {
	typeName string
}

func (fs fakeSchema) GetType() string {
	return fs.typeName
}

func TestGetOidForSchema(t *testing.T) {
	tests := []struct {
		schemaType string
		expected   oid.Oid
	}{
		{"object", oid.T_text},
		{"array", oid.T_text},
		{"boolean", oid.T_text},
		{"bool", oid.T_text},
		{"number", oid.T_numeric},
		{"string", oid.T_text},
		{"", oid.T_text},
	}
	for _, tt := range tests {
		t.Run(tt.schemaType, func(t *testing.T) {
			got := GetOidForSchema(fakeSchema{typeName: tt.schemaType})
			if got != tt.expected {
				t.Errorf("GetOidForSchema(%q) = %d, want %d", tt.schemaType, got, tt.expected)
			}
		})
	}
}
