package typing //nolint:testpackage // tests unexported behaviour in same package for simplicity

import (
	"testing"

	"github.com/lib/pq/oid"
)

func TestRelayedColumnMetadataAccessors(t *testing.T) {
	col := NewRelationalColumn("name", "varchar").WithDecorated("dec")
	md := NewRelayedColDescriptor(col, "text")

	if got := md.GetName(); got != "name" {
		t.Errorf("GetName() = %q", got)
	}
	if got := md.GetWireName(); got != "name" {
		t.Errorf("GetWireName() = %q", got)
	}
	if got := md.GetDecorated(); got != "dec" {
		t.Errorf("GetDecorated() = %q", got)
	}
	if got := md.GetType(); got != "varchar" {
		t.Errorf("GetType() = %q", got)
	}
	if got := md.GetRelationalType(); got != "text" {
		t.Errorf("GetRelationalType() = %q", got)
	}
}

func TestRelayedColumnMetadataIdentifier(t *testing.T) {
	plain := NewRelayedColDescriptor(NewRelationalColumn("name", "text"), "text")
	if got := plain.GetIdentifier(); got != "name" {
		t.Errorf("GetIdentifier() = %q, want %q", got, "name")
	}
	aliased := NewRelayedColDescriptor(NewRelationalColumn("name", "text").WithAlias("al"), "text")
	if got := aliased.GetIdentifier(); got != "al" {
		t.Errorf("GetIdentifier() = %q, want %q", got, "al")
	}
	// wire name ignores alias
	if got := aliased.GetWireName(); got != "name" {
		t.Errorf("GetWireName() = %q, want %q", got, "name")
	}
}

func TestRelayedColumnMetadataOIDFromRelationalType(t *testing.T) {
	tests := []struct {
		relType  string
		expected oid.Oid
	}{
		{"object", oid.T_text},
		{"array", oid.T_text},
		{"text", oid.T_text},
		{"boolean", oid.T_text},
		{"bool", oid.T_text},
		{"number", oid.T_numeric},
		{"decimal", oid.T_numeric},
		{"numeric", oid.T_numeric},
		{"real", oid.T_numeric},
		{"NUMERIC", oid.T_numeric},
		{"something-else", oid.T_text},
		{"", oid.T_text},
	}
	for _, tt := range tests {
		t.Run(tt.relType, func(t *testing.T) {
			md := NewRelayedColDescriptor(NewRelationalColumn("c", "t"), tt.relType)
			if got := md.GetColumnOID(); got != tt.expected {
				t.Errorf("GetColumnOID() with relational type %q = %d, want %d", tt.relType, got, tt.expected)
			}
		})
	}
}

func TestRelayedColumnMetadataExplicitOIDWins(t *testing.T) {
	col := NewRelationalColumn("c", "t").WithOID(oid.T_bool)
	md := NewRelayedColDescriptor(col, "numeric")
	if got := md.GetColumnOID(); got != oid.T_bool {
		t.Errorf("GetColumnOID() = %d, want %d", got, oid.T_bool)
	}
}
