package serde_test

import (
	"testing"

	"github.com/stackql/stackql/pkg/serde"
)

func TestStringArrayMapSerDeSerialize(t *testing.T) {
	s := serde.NewStringArrayMapSerDe()
	result, err := s.Serialize([]string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "a,b,c"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestStringArrayMapSerDeSerializeEmpty(t *testing.T) {
	s := serde.NewStringArrayMapSerDe()
	result, err := s.Serialize([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestStringArrayMapSerDeSerializeSingle(t *testing.T) {
	s := serde.NewStringArrayMapSerDe()
	result, err := s.Serialize([]string{"single"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "single" {
		t.Errorf("expected %q, got %q", "single", result)
	}
}

func TestStringArrayMapSerDeSerializeNil(t *testing.T) {
	s := serde.NewStringArrayMapSerDe()
	result, err := s.Serialize(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestStringArrayMapSerDeSerializeContainsCommas(t *testing.T) {
	s := serde.NewStringArrayMapSerDe()
	result, err := s.Serialize([]string{"a,b", "c", "d,e,f"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Commas inside elements are not escaped, so the result is "a,b,c,d,e,f"
	expected := "a,b,c,d,e,f"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestStringArrayMapSerDeDeserialize(t *testing.T) {
	s := serde.NewStringArrayMapSerDe()
	result, err := s.Deserialize("a,b,c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 elements, got %d", len(result))
	}
	for _, key := range []string{"a", "b", "c"} {
		if _, ok := result[key]; !ok {
			t.Errorf("expected key %q in result", key)
		}
	}
}

func TestStringArrayMapSerDeDeserializeEmpty(t *testing.T) {
	s := serde.NewStringArrayMapSerDe()
	result, err := s.Deserialize("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 elements, got %d", len(result))
	}
}

func TestStringArrayMapSerDeDeserializeSingle(t *testing.T) {
	s := serde.NewStringArrayMapSerDe()
	result, err := s.Deserialize("single")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 element, got %d", len(result))
	}
	if _, ok := result["single"]; !ok {
		t.Error("expected key 'single' in result")
	}
}

func TestStringArrayMapSerDeDeserializeEmptyElements(t *testing.T) {
	s := serde.NewStringArrayMapSerDe()
	result, err := s.Deserialize("a,,b,,")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty elements should be skipped
	if len(result) != 2 {
		t.Errorf("expected 2 elements, got %d", len(result))
	}
	if _, ok := result["a"]; !ok {
		t.Error("expected key 'a' in result")
	}
	if _, ok := result["b"]; !ok {
		t.Error("expected key 'b' in result")
	}
}

func TestStringArrayMapSerDeRoundTrip(t *testing.T) {
	s := serde.NewStringArrayMapSerDe()
	original := []string{"x", "y", "z"}
	serialized, err := s.Serialize(original)
	if err != nil {
		t.Fatalf("serialize error: %v", err)
	}
	deserialized, err := s.Deserialize(serialized)
	if err != nil {
		t.Fatalf("deserialize error: %v", err)
	}
	if len(deserialized) != len(original) {
		t.Errorf("expected %d elements, got %d", len(original), len(deserialized))
	}
	for _, key := range original {
		if _, ok := deserialized[key]; !ok {
			t.Errorf("expected key %q in deserialized map", key)
		}
	}
}

func TestStringArrayMapSerDeValuesAreEmptyStruct(t *testing.T) {
	s := serde.NewStringArrayMapSerDe()
	result, err := s.Deserialize("a,b,c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for k, v := range result {
		if v == nil {
			t.Errorf("expected non-nil value for key %q", k)
		}
	}
}

func TestStringArrayMapSerDeUniqueKeys(t *testing.T) {
	s := serde.NewStringArrayMapSerDe()
	result, err := s.Deserialize("a,b,a,c,a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Map should dedupe
	if len(result) != 3 {
		t.Errorf("expected 3 unique elements, got %d", len(result))
	}
}
