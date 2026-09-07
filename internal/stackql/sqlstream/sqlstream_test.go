package sqlstream_test

import (
	"io"
	"testing"

	"github.com/stackql/stackql/internal/stackql/sqlstream"
)

func TestNewStaticMapStream(t *testing.T) {
	payload := []map[string]interface{}{
		{"id": 1, "name": "test"},
	}
	stream := sqlstream.NewStaticMapStream(payload)
	if stream == nil {
		t.Fatal("NewStaticMapStream returned nil")
	}
}

func TestStaticMapStream_Write(t *testing.T) {
	payload := []map[string]interface{}{}
	stream := sqlstream.NewStaticMapStream(payload)
	err := stream.Write(nil)
	if err != nil {
		t.Errorf("Write(nil) error = %v, want nil", err)
	}
}

func TestStaticMapStream_Write_WithData(t *testing.T) {
	payload := []map[string]interface{}{}
	stream := sqlstream.NewStaticMapStream(payload)
	err := stream.Write([]map[string]interface{}{
		{"key": "value"},
	})
	if err != nil {
		t.Errorf("Write() error = %v, want nil", err)
	}
}

func TestStaticMapStream_Read_ReturnsPayload(t *testing.T) {
	payload := []map[string]interface{}{
		{"id": 1, "name": "alice"},
		{"id": 2, "name": "bob"},
	}
	stream := sqlstream.NewStaticMapStream(payload)
	got, err := stream.Read()
	if err != io.EOF {
		t.Errorf("Read() error = %v, want io.EOF", err)
	}
	if len(got) != 2 {
		t.Errorf("Read() returned %d rows, want 2", len(got))
	}
	if got[0]["id"] != 1 {
		t.Errorf("Read()[0][id] = %v, want 1", got[0]["id"])
	}
	if got[1]["name"] != "bob" {
		t.Errorf("Read()[1][name] = %v, want 'bob'", got[1]["name"])
	}
}

func TestStaticMapStream_Read_EmptyPayload(t *testing.T) {
	payload := []map[string]interface{}{}
	stream := sqlstream.NewStaticMapStream(payload)
	got, err := stream.Read()
	if err != io.EOF {
		t.Errorf("Read() error = %v, want io.EOF", err)
	}
	if len(got) != 0 {
		t.Errorf("Read() returned %d rows, want 0", len(got))
	}
}

func TestStaticMapStream_Read_NilPayload(t *testing.T) {
	stream := sqlstream.NewStaticMapStream(nil)
	got, err := stream.Read()
	if err != io.EOF {
		t.Errorf("Read() error = %v, want io.EOF", err)
	}
	if got != nil {
		t.Errorf("Read() returned %v, want nil for nil payload", got)
	}
}

func TestStaticMapStream_Read_MultipleRows(t *testing.T) {
	payload := []map[string]interface{}{
		{"a": 1, "b": 2, "c": 3},
		{"a": 4, "b": 5, "c": 6},
		{"a": 7, "b": 8, "c": 9},
	}
	stream := sqlstream.NewStaticMapStream(payload)
	got, err := stream.Read()
	if err != io.EOF {
		t.Errorf("Read() error = %v, want io.EOF", err)
	}
	if len(got) != 3 {
		t.Errorf("Read() returned %d rows, want 3", len(got))
	}
}

func TestStaticMapStream_Read_NestedMaps(t *testing.T) {
	payload := []map[string]interface{}{
		{
			"user": map[string]interface{}{
				"name": "test",
				"age":  30,
			},
			"active": true,
		},
	}
	stream := sqlstream.NewStaticMapStream(payload)
	got, err := stream.Read()
	if err != io.EOF {
		t.Errorf("Read() error = %v, want io.EOF", err)
	}
	nested, ok := got[0]["user"].(map[string]interface{})
	if !ok {
		t.Fatalf("Read()[0][user] is not a map, got %T", got[0]["user"])
	}
	if nested["name"] != "test" {
		t.Errorf("Read()[0][user][name] = %v, want 'test'", nested["name"])
	}
}
