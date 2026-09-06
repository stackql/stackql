package symtab_test

import (
	"testing"

	"github.com/stackql/stackql/internal/stackql/symtab"
)

func TestNewSymTabEntry(t *testing.T) {
	entry := symtab.NewSymTabEntry("test_type", "test_data", "test_in")
	if entry.Type != "test_type" {
		t.Errorf("expected Type 'test_type', got %q", entry.Type)
	}
	if entry.Data != "test_data" {
		t.Errorf("expected Data 'test_data', got %v", entry.Data)
	}
	if entry.In != "test_in" {
		t.Errorf("expected In 'test_in', got %q", entry.In)
	}
}

func TestNewHashMapTreeSymTab(t *testing.T) {
	st := symtab.NewHashMapTreeSymTab()
	if st == nil {
		t.Fatal("NewHashMapTreeSymTab returned nil")
	}
}

func TestSymTabSetSymbol(t *testing.T) {
	st := symtab.NewHashMapTreeSymTab()
	err := st.SetSymbol("key1", symtab.NewSymTabEntry("type1", "data1", "in1"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSymTabSetSymbolDuplicate(t *testing.T) {
	st := symtab.NewHashMapTreeSymTab()
	_ = st.SetSymbol("key1", symtab.NewSymTabEntry("type1", "data1", "in1"))
	err := st.SetSymbol("key1", symtab.NewSymTabEntry("type2", "data2", "in2"))
	if err == nil {
		t.Error("expected error when setting duplicate key")
	}
}

func TestSymTabGetSymbol(t *testing.T) {
	st := symtab.NewHashMapTreeSymTab()
	_ = st.SetSymbol("key1", symtab.NewSymTabEntry("type1", "data1", "in1"))

	entry, err := st.GetSymbol("key1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if entry.Type != "type1" {
		t.Errorf("expected Type 'type1', got %q", entry.Type)
	}
	if entry.Data != "data1" {
		t.Errorf("expected Data 'data1', got %v", entry.Data)
	}
	if entry.In != "in1" {
		t.Errorf("expected In 'in1', got %q", entry.In)
	}
}

func TestSymTabGetSymbolNonExistent(t *testing.T) {
	st := symtab.NewHashMapTreeSymTab()
	_, err := st.GetSymbol("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent key")
	}
}

func TestSymTabGetSymbolFromLeaf(t *testing.T) {
	root := symtab.NewHashMapTreeSymTab()
	leaf, err := root.NewLeaf(0)
	if err != nil {
		t.Fatalf("unexpected error creating leaf: %v", err)
	}
	_ = leaf.SetSymbol("leaf_key", symtab.NewSymTabEntry("leaf_type", "leaf_data", "leaf_in"))

	entry, err := root.GetSymbol("leaf_key")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if entry.Type != "leaf_type" {
		t.Errorf("expected Type 'leaf_type', got %q", entry.Type)
	}
}

func TestSymTabNewLeaf(t *testing.T) {
	st := symtab.NewHashMapTreeSymTab()
	leaf, err := st.NewLeaf(0)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if leaf == nil {
		t.Fatal("NewLeaf returned nil")
	}
}

func TestSymTabNewLeafDuplicate(t *testing.T) {
	st := symtab.NewHashMapTreeSymTab()
	_, err := st.NewLeaf(0)
	if err != nil {
		t.Errorf("unexpected error on first leaf: %v", err)
	}
	_, err = st.NewLeaf(0)
	if err == nil {
		t.Error("expected error when creating duplicate leaf")
	}
}

func TestSymTabNewLeafMultiple(t *testing.T) {
	st := symtab.NewHashMapTreeSymTab()
	_, err := st.NewLeaf(0)
	if err != nil {
		t.Errorf("unexpected error on first leaf: %v", err)
	}
	_, err = st.NewLeaf(1)
	if err != nil {
		t.Errorf("unexpected error on second leaf: %v", err)
	}
}

func TestSymTabMerge(t *testing.T) {
	root := symtab.NewHashMapTreeSymTab()
	other := symtab.NewHashMapTreeSymTab()

	_ = other.SetSymbol("key1", symtab.NewSymTabEntry("type1", "data1", "in1"))

	err := root.Merge(other, "")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	entry, err := root.GetSymbol("key1")
	if err != nil {
		t.Errorf("expected to find merged key: %v", err)
	}
	if entry.Type != "type1" {
		t.Errorf("expected Type 'type1', got %q", entry.Type)
	}
}

func TestSymTabMergeWithPrefix(t *testing.T) {
	root := symtab.NewHashMapTreeSymTab()
	other := symtab.NewHashMapTreeSymTab()

	_ = other.SetSymbol("key1", symtab.NewSymTabEntry("type1", "data1", "in1"))

	err := root.Merge(other, "myprefix")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	entry, err := root.GetSymbol("myprefix.key1")
	if err != nil {
		t.Errorf("expected to find merged key with prefix: %v", err)
	}
	if entry.Type != "type1" {
		t.Errorf("expected Type 'type1', got %q", entry.Type)
	}
}

func TestSymTabMergeConflict(t *testing.T) {
	root := symtab.NewHashMapTreeSymTab()
	_ = root.SetSymbol("key1", symtab.NewSymTabEntry("type1", "data1", "in1"))

	other := symtab.NewHashMapTreeSymTab()
	_ = other.SetSymbol("key1", symtab.NewSymTabEntry("type2", "data2", "in2"))

	err := root.Merge(other, "")
	if err == nil {
		t.Error("expected error when merging duplicate keys")
	}
}

func TestSymTabMergeWithPrefixAndNonStringKey(t *testing.T) {
	root := symtab.NewHashMapTreeSymTab()
	other := symtab.NewHashMapTreeSymTab()

	_ = other.SetSymbol("key1", symtab.NewSymTabEntry("type1", "data1", "in1"))
	_ = other.SetSymbol(42, symtab.NewSymTabEntry("int_type", "int_data", "int_in"))

	err := root.Merge(other, "prefix")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// String key should get prefixed
	entry, err := root.GetSymbol("prefix.key1")
	if err != nil {
		t.Errorf("expected to find prefixed key: %v", err)
	}
	if entry.Type != "type1" {
		t.Errorf("expected Type 'type1', got %q", entry.Type)
	}

	// Non-string key should remain as-is
	_, err = root.GetSymbol(42)
	if err != nil {
		t.Errorf("expected to find non-string key: %v", err)
	}
}

func TestSymTabMergeLeaves(t *testing.T) {
	root := symtab.NewHashMapTreeSymTab()
	other := symtab.NewHashMapTreeSymTab()
	_, _ = other.NewLeaf(0)

	err := root.Merge(other, "")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	leaf, err := root.NewLeaf(0)
	if err == nil {
		t.Error("expected error since leaf 0 was merged from other")
	}
	_ = leaf // unused
}

func TestSymTabMergeEmptyOther(t *testing.T) {
	root := symtab.NewHashMapTreeSymTab()
	other := symtab.NewHashMapTreeSymTab()

	err := root.Merge(other, "")
	if err != nil {
		t.Errorf("unexpected error merging empty: %v", err)
	}
}

func TestSymTabMultipleSetGet(t *testing.T) {
	st := symtab.NewHashMapTreeSymTab()
	_ = st.SetSymbol("key1", symtab.NewSymTabEntry("type1", "data1", "in1"))
	_ = st.SetSymbol("key2", symtab.NewSymTabEntry("type2", "data2", "in2"))
	_ = st.SetSymbol("key3", symtab.NewSymTabEntry("type3", "data3", "in3"))

	for i, key := range []string{"key1", "key2", "key3"} {
		entry, err := st.GetSymbol(key)
		if err != nil {
			t.Errorf("unexpected error for key %s: %v", key, err)
		}
		expectedType := []string{"type1", "type2", "type3"}[i]
		if entry.Type != expectedType {
			t.Errorf("expected Type %q, got %q", expectedType, entry.Type)
		}
	}
}

func TestSymTabGetWithDifferentTypesOfKeys(t *testing.T) {
	st := symtab.NewHashMapTreeSymTab()
	_ = st.SetSymbol(42, symtab.NewSymTabEntry("int_type", "int_data", "int_in"))
	_ = st.SetSymbol("string_key", symtab.NewSymTabEntry("string_type", "string_data", "string_in"))

	intEntry, err := st.GetSymbol(42)
	if err != nil {
		t.Errorf("unexpected error for int key: %v", err)
	}
	if intEntry.Type != "int_type" {
		t.Errorf("expected Type 'int_type', got %q", intEntry.Type)
	}

	strEntry, err := st.GetSymbol("string_key")
	if err != nil {
		t.Errorf("unexpected error for string key: %v", err)
	}
	if strEntry.Type != "string_type" {
		t.Errorf("expected Type 'string_type', got %q", strEntry.Type)
	}
}

func TestSymTabMergeMultipleKeys(t *testing.T) {
	root := symtab.NewHashMapTreeSymTab()
	other := symtab.NewHashMapTreeSymTab()
	_ = other.SetSymbol("key1", symtab.NewSymTabEntry("type1", "data1", "in1"))
	_ = other.SetSymbol("key2", symtab.NewSymTabEntry("type2", "data2", "in2"))
	_ = other.SetSymbol("key3", symtab.NewSymTabEntry("type3", "data3", "in3"))

	err := root.Merge(other, "")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	for _, key := range []string{"key1", "key2", "key3"} {
		_, err := root.GetSymbol(key)
		if err != nil {
			t.Errorf("expected to find merged key %s: %v", key, err)
		}
	}
}
