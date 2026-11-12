package typing //nolint:testpackage // tests unexported behaviour in same package for simplicity

import (
	"reflect"
	"testing"
)

func TestNewORMCoupling(t *testing.T) {
	tests := []struct {
		name           string
		relationalType string
		golangKind     reflect.Kind
	}{
		{
			name:           "text string",
			relationalType: "text",
			golangKind:     reflect.String,
		},
		{
			name:           "bigint int64",
			relationalType: "bigint",
			golangKind:     reflect.Int64,
		},
		{
			name:           "empty values",
			relationalType: "",
			golangKind:     reflect.Invalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewORMCoupling(tt.relationalType, tt.golangKind)

			if got := c.GetRelationalType(); got != tt.relationalType {
				t.Errorf("GetRelationalType() = %q, want %q", got, tt.relationalType)
			}

			if got := c.GetGolangKind(); got != tt.golangKind {
				t.Errorf("GetGolangKind() = %v, want %v", got, tt.golangKind)
			}
		})
	}
}
