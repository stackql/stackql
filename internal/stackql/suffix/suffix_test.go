package suffix_test

import (
	"testing"

	"github.com/stackql/stackql/internal/stackql/suffix"
)

// mockAddressable implements formulation.Addressable for testing purposes.
type mockAddressable struct {
	name            string
	typ             string
	conditionResult bool
}

func (m *mockAddressable) ConditionIsValid(lhs string, rhs interface{}) bool {
	return m.conditionResult
}

func (m *mockAddressable) GetName() string {
	return m.name
}

func (m *mockAddressable) GetType() string {
	return m.typ
}

func newMock(name, typ string) *mockAddressable {
	return &mockAddressable{name: name, typ: typ, conditionResult: true}
}

func TestNewParameterSuffixMap(t *testing.T) {
	psm := suffix.NewParameterSuffixMap()
	if psm == nil {
		t.Fatal("NewParameterSuffixMap returned nil")
	}
}

func TestParameterSuffixMapSize(t *testing.T) {
	psm := suffix.NewParameterSuffixMap()
	if got := psm.Size(); got != 0 {
		t.Errorf("expected size 0, got %d", got)
	}
}

func TestParameterSuffixMapPut(t *testing.T) {
	psm := suffix.NewParameterSuffixMap()
	psm.Put("key1", newMock("name1", "type1"))
	if got := psm.Size(); got != 1 {
		t.Errorf("expected size 1, got %d", got)
	}
}

func TestParameterSuffixMapPutMultiple(t *testing.T) {
	psm := suffix.NewParameterSuffixMap()
	psm.Put("key1", newMock("name1", "type1"))
	psm.Put("key2", newMock("name2", "type2"))
	psm.Put("key3", newMock("name3", "type3"))
	if got := psm.Size(); got != 3 {
		t.Errorf("expected size 3, got %d", got)
	}
}

func TestParameterSuffixMapGet(t *testing.T) {
	psm := suffix.NewParameterSuffixMap()
	psm.Put("key1", newMock("name1", "type1"))

	val, ok := psm.Get("key1")
	if !ok {
		t.Error("expected to find key1")
	}
	addr, ok := val.(*mockAddressable)
	if !ok {
		t.Fatalf("expected *mockAddressable, got %T", val)
	}
	if addr.name != "name1" {
		t.Errorf("expected name1, got %s", addr.name)
	}
}

func TestParameterSuffixMapGetNonExistent(t *testing.T) {
	psm := suffix.NewParameterSuffixMap()
	psm.Put("key1", newMock("name1", "type1"))

	_, ok := psm.Get("nonexistent")
	if ok {
		t.Error("expected not to find nonexistent key")
	}
}

func TestParameterSuffixMapGetAll(t *testing.T) {
	psm := suffix.NewParameterSuffixMap()
	psm.Put("key1", newMock("name1", "type1"))
	psm.Put("key2", newMock("name2", "type2"))

	all := psm.GetAll()
	if len(all) != 2 {
		t.Errorf("expected 2 items, got %d", len(all))
	}
	if _, ok := all["key1"]; !ok {
		t.Error("expected key1 in GetAll result")
	}
	if _, ok := all["key2"]; !ok {
		t.Error("expected key2 in GetAll result")
	}
}

func TestParameterSuffixMapGetAllEmpty(t *testing.T) {
	psm := suffix.NewParameterSuffixMap()
	all := psm.GetAll()
	if len(all) != 0 {
		t.Errorf("expected empty map, got %d items", len(all))
	}
}

func TestParameterSuffixMapDelete(t *testing.T) {
	psm := suffix.NewParameterSuffixMap()
	psm.Put("key1", newMock("name1", "type1"))

	if got := psm.Size(); got != 1 {
		t.Fatalf("expected size 1, got %d", got)
	}

	deleted := psm.Delete("key1")
	if !deleted {
		t.Error("expected Delete to return true")
	}

	if got := psm.Size(); got != 0 {
		t.Errorf("expected size 0 after delete, got %d", got)
	}

	_, ok := psm.Get("key1")
	if ok {
		t.Error("expected key1 to be deleted")
	}
}

func TestParameterSuffixMapDeleteNonExistent(t *testing.T) {
	psm := suffix.NewParameterSuffixMap()
	deleted := psm.Delete("nonexistent")
	if deleted {
		t.Error("expected Delete to return false for nonexistent key")
	}
}

func TestParameterSuffixMapDeleteUpdatesGetAll(t *testing.T) {
	psm := suffix.NewParameterSuffixMap()
	psm.Put("key1", newMock("name1", "type1"))
	psm.Put("key2", newMock("name2", "type2"))

	psm.Delete("key1")
	all := psm.GetAll()
	if len(all) != 1 {
		t.Errorf("expected 1 item after delete, got %d", len(all))
	}
	if _, ok := all["key1"]; ok {
		t.Error("expected key1 to be absent after delete")
	}
	if _, ok := all["key2"]; !ok {
		t.Error("expected key2 to remain after delete")
	}
}

func TestParameterSuffixMapPutOverwrite(t *testing.T) {
	psm := suffix.NewParameterSuffixMap()
	psm.Put("key1", newMock("name1", "type1"))
	psm.Put("key1", newMock("name2", "type2"))

	all := psm.GetAll()
	if len(all) != 1 {
		t.Errorf("expected 1 item, got %d", len(all))
	}

	val, _ := psm.Get("key1")
	addr := val.(*mockAddressable)
	if addr.name != "name2" {
		t.Errorf("expected name2, got %s", addr.name)
	}
}

func TestParameterSuffixMapConditionIsValid(t *testing.T) {
	psm := suffix.NewParameterSuffixMap()
	// Test that we can store addressables with different ConditionIsValid results
	trueAddr := &mockAddressable{name: "name1", typ: "type1", conditionResult: true}
	falseAddr := &mockAddressable{name: "name2", typ: "type2", conditionResult: false}
	psm.Put("key1", trueAddr)
	psm.Put("key2", falseAddr)

	v1, _ := psm.Get("key1")
	if !v1.ConditionIsValid("lhs", "rhs") {
		t.Error("expected ConditionIsValid to return true")
	}

	v2, _ := psm.Get("key2")
	if v2.ConditionIsValid("lhs", "rhs") {
		t.Error("expected ConditionIsValid to return false")
	}
}

func TestParameterSuffixMapGetType(t *testing.T) {
	psm := suffix.NewParameterSuffixMap()
	psm.Put("key1", newMock("name1", "typeA"))
	psm.Put("key2", newMock("name2", "typeB"))

	v1, _ := psm.Get("key1")
	if v1.GetType() != "typeA" {
		t.Errorf("expected typeA, got %s", v1.GetType())
	}

	v2, _ := psm.Get("key2")
	if v2.GetType() != "typeB" {
		t.Errorf("expected typeB, got %s", v2.GetType())
	}
}

func TestParameterSuffixMapGetName(t *testing.T) {
	psm := suffix.NewParameterSuffixMap()
	psm.Put("key1", newMock("nameA", "type1"))
	psm.Put("key2", newMock("nameB", "type2"))

	v1, _ := psm.Get("key1")
	if v1.GetName() != "nameA" {
		t.Errorf("expected nameA, got %s", v1.GetName())
	}

	v2, _ := psm.Get("key2")
	if v2.GetName() != "nameB" {
		t.Errorf("expected nameB, got %s", v2.GetName())
	}
}
