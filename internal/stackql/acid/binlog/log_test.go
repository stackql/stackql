package binlog_test

import (
	"reflect"
	"testing"

	"github.com/stackql/stackql/internal/stackql/acid/binlog"
)

func TestConcatenate(t *testing.T) {
	entry := binlog.NewSimpleLogEntry(
		nil,
		[]string{
			"Undo the delete on my_table",
			"Undo the insert on second_table",
		},
	)
	second := binlog.NewSimpleLogEntry([]byte("insert"), nil)
	third := binlog.NewSimpleLogEntry(
		[]byte("delete"),
		[]string{
			"Undo delete on third_table",
		},
	)

	expectedHumanReadable := append(entry.GetHumanReadable(), third.GetHumanReadable()...)
	entry.Concatenate(second, third)
	raw := entry.GetRaw()
	if !reflect.DeepEqual(raw, []byte("insertdelete")) {
		t.Fatalf("Raw representation of combined log is expected to contain %v. Got %+v", []byte("insertdelete"), raw)
	}

	humanReadable := entry.GetHumanReadable()
	if !reflect.DeepEqual(humanReadable, expectedHumanReadable) {
		t.Fatalf(
			"Human readable representation of combined log is expected to be %v. Got %v",
			expectedHumanReadable,
			humanReadable,
		)
	}
}

func TestClone(t *testing.T) {
	entry := binlog.NewSimpleLogEntry(
		[]byte("insert"),
		[]string{
			"Undo the delete on my_table",
			"Undo the insert on second_table",
		},
	)
	cloned := entry.Clone()
	if &cloned.GetRaw()[0] == &entry.GetRaw()[0] {
		t.Fatal("Expected 'raw' field of the cloned value to be at a different memory location")
	}

	if &cloned.GetHumanReadable()[0] == &entry.GetHumanReadable()[0] {
		t.Fatal("Expected 'humanReadable' field of the cloned value to be at a different memory location")
	}
}

func TestAppend(t *testing.T) {
	entry := binlog.NewSimpleLogEntry(
		[]byte("insert"),
		[]string{
			"Undo the delete on my_table",
			"Undo the insert on second_table",
		},
	)
	entry.AppendRaw([]byte("delete"))
	entry.AppendHumanReadable("Undo delete on third_table")
	expectedRaw := []byte("insertdelete")
	if entry.Size() != len(expectedRaw) {
		t.Fatalf("Expected size of the raw field to be %d, Got %d", len(expectedRaw), entry.Size())
	}

	if !reflect.DeepEqual(entry.GetRaw(), expectedRaw) {
		t.Fatalf("Expected raw field after append operation to be %v, Got %v", expectedRaw, entry.GetRaw())
	}

	expectedHumanReadable := []string{
		"Undo the delete on my_table",
		"Undo the insert on second_table",
		"Undo delete on third_table",
	}
	if !reflect.DeepEqual(entry.GetHumanReadable(), expectedHumanReadable) {
		t.Fatalf(
			"Expected humanReadable field after append operation to be %v, Got %v",
			expectedHumanReadable,
			entry.GetHumanReadable(),
		)
	}
}
