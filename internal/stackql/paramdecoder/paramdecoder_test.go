package paramdecoder_test

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/lib/pq/oid"
	"github.com/stackql/stackql/internal/stackql/paramdecoder"
)

func TestDecodeTextParams(t *testing.T) {
	d := paramdecoder.NewDecoder()
	results, err := d.DecodeParams(
		[]uint32{uint32(oid.T_text), uint32(oid.T_text)},
		[]int16{0}, // all text
		[][]byte{[]byte("hello"), []byte("world")},
	)
	if err != nil {
		t.Fatal(err)
	}
	if results[0] != "hello" || results[1] != "world" {
		t.Errorf("got %v", results)
	}
}

func TestDecodeNullParam(t *testing.T) {
	d := paramdecoder.NewDecoder()
	results, err := d.DecodeParams(
		[]uint32{uint32(oid.T_text)},
		nil,
		[][]byte{nil},
	)
	if err != nil {
		t.Fatal(err)
	}
	if results[0] != "NULL" {
		t.Errorf("got %q, want NULL", results[0])
	}
}

func TestDecodeBinaryInt4(t *testing.T) {
	d := paramdecoder.NewDecoder()
	val := make([]byte, 4)
	binary.BigEndian.PutUint32(val, uint32(42))
	results, err := d.DecodeParams(
		[]uint32{uint32(oid.T_int4)},
		[]int16{1}, // binary
		[][]byte{val},
	)
	if err != nil {
		t.Fatal(err)
	}
	if results[0] != "42" {
		t.Errorf("got %q, want 42", results[0])
	}
}

func TestDecodeBinaryInt8(t *testing.T) {
	d := paramdecoder.NewDecoder()
	val := make([]byte, 8)
	binary.BigEndian.PutUint64(val, uint64(9999999999))
	results, err := d.DecodeParams(
		[]uint32{uint32(oid.T_int8)},
		[]int16{1},
		[][]byte{val},
	)
	if err != nil {
		t.Fatal(err)
	}
	if results[0] != "9999999999" {
		t.Errorf("got %q, want 9999999999", results[0])
	}
}

func TestDecodeBinaryFloat8(t *testing.T) {
	d := paramdecoder.NewDecoder()
	val := make([]byte, 8)
	binary.BigEndian.PutUint64(val, math.Float64bits(3.14))
	results, err := d.DecodeParams(
		[]uint32{uint32(oid.T_float8)},
		[]int16{1},
		[][]byte{val},
	)
	if err != nil {
		t.Fatal(err)
	}
	if results[0] != "3.14" {
		t.Errorf("got %q, want 3.14", results[0])
	}
}

func TestDecodeBinaryBool(t *testing.T) {
	d := paramdecoder.NewDecoder()
	results, err := d.DecodeParams(
		[]uint32{uint32(oid.T_bool), uint32(oid.T_bool)},
		[]int16{1},
		[][]byte{{1}, {0}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if results[0] != "true" || results[1] != "false" {
		t.Errorf("got %v", results)
	}
}

func TestDecodeMixedFormats(t *testing.T) {
	d := paramdecoder.NewDecoder()
	int4Val := make([]byte, 4)
	binary.BigEndian.PutUint32(int4Val, uint32(100))
	results, err := d.DecodeParams(
		[]uint32{uint32(oid.T_text), uint32(oid.T_int4)},
		[]int16{0, 1}, // first text, second binary
		[][]byte{[]byte("hello"), int4Val},
	)
	if err != nil {
		t.Fatal(err)
	}
	if results[0] != "hello" || results[1] != "100" {
		t.Errorf("got %v", results)
	}
}

func TestDecodeUnknownOIDBinaryFallsBackToText(t *testing.T) {
	d := paramdecoder.NewDecoder()
	results, err := d.DecodeParams(
		[]uint32{99999},
		[]int16{1}, // binary
		[][]byte{[]byte("raw-bytes")},
	)
	if err != nil {
		t.Fatal(err)
	}
	if results[0] != "raw-bytes" {
		t.Errorf("got %q, want raw-bytes", results[0])
	}
}
