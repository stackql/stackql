package primitivebuilder //nolint:testpackage // exercise the wire adapter

import "testing"

func TestDependencySQLRowNulls(t *testing.T) {
	row := newDependencySQLRow([]interface{}{nil, []byte(""), []byte("name")})
	raw := row.GetRowDataNaive()
	if raw[0] != nil {
		t.Fatal("CLI NULL representation changed")
	}
	wire := row.GetRowDataForPgWire()
	null, ok := wire[0].([]byte)
	if !ok || null != nil {
		t.Fatalf("expected a typed nil for the wire text encoder, got %#v", wire[0])
	}
	if wire[1] != "" || wire[2] != "name" {
		t.Fatalf("non-NULL values changed: %v", wire)
	}
}
