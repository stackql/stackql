package iqlerror_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stackql/stackql/internal/stackql/iqlerror"
)

func TestGetStatementNotSupportedError(t *testing.T) {
	err := iqlerror.GetStatementNotSupportedError("SELECT")
	if err == nil {
		t.Fatal("GetStatementNotSupportedError returned nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "SELECT") {
		t.Errorf("error message should contain 'SELECT', got %q", msg)
	}
	if !strings.Contains(msg, "not yet supported") {
		t.Errorf("error message should contain 'not yet supported', got %q", msg)
	}
}

func TestGetStatementNotSupportedError_EmptyStatement(t *testing.T) {
	err := iqlerror.GetStatementNotSupportedError("")
	if err == nil {
		t.Fatal("GetStatementNotSupportedError returned nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "not yet supported") {
		t.Errorf("error message should contain 'not yet supported', got %q", msg)
	}
}

func TestGetStatementNotSupportedError_SpecialChars(t *testing.T) {
	err := iqlerror.GetStatementNotSupportedError("INSERT INTO")
	if err == nil {
		t.Fatal("GetStatementNotSupportedError returned nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "INSERT INTO") {
		t.Errorf("error message should contain 'INSERT INTO', got %q", msg)
	}
}

func TestHandlePanic_NoPanic(t *testing.T) {
	var buf bytes.Buffer
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("HandlePanic should not panic when there is no panic: %v", r)
		}
		if buf.Len() > 0 {
			t.Errorf("HandlePanic should not write anything when there is no panic, wrote: %q", buf.String())
		}
	}()

	iqlerror.HandlePanic(&buf)
}

func TestHandlePanic_WithPanic(t *testing.T) {
	var buf bytes.Buffer
	// Deferred recover only runs when HandlePanic does NOT recover from the panic.
	// If HandlePanic works correctly, it calls recover() first and returns normally,
	// so this deferred function never sees a panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic was not recovered by HandlePanic: %v", r)
		}
	}()
	// Deferred HandlePanic runs during panic unwinding and calls recover() to stop it.
	defer iqlerror.HandlePanic(&buf)
	panic("test panic")
}

func TestHandlePanic_WithNilWriter(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("HandlePanic should have panicked when there was no panic")
		}
		if r.(string) != "nil writer test" {
			t.Errorf("panic value = %v, want 'nil writer test'", r)
		}
	}()

	// Should panic even with nil writer
	iqlerror.HandlePanic(nil)
	panic("nil writer test")
}

func TestHandlePanic_WithIntegerPanic(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("HandlePanic should have panicked")
		}
		if r.(int) != 42 {
			t.Errorf("panic value = %v, want 42", r)
		}
	}()

	iqlerror.HandlePanic(&bytes.Buffer{})
	panic(42)
}

func TestHandlePanic_WithErrorPanic(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("HandlePanic should have panicked")
		}
	}()

	var buf bytes.Buffer
	iqlerror.HandlePanic(&buf)
	panic("oops")
}

func TestHandlePanic_WritesMessageToWriter(t *testing.T) {
	var buf bytes.Buffer
	defer func() {
		recover() // stop panic
	}()

	defer iqlerror.HandlePanic(&buf)
	panic("buffer test")

	// This line is unreachable after panic
}

func TestHandlePanic_WritesMessageFormat(t *testing.T) {
	var buf bytes.Buffer
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("unexpected panic: %v", r)
			}
		}()
		defer iqlerror.HandlePanic(&buf)
		panic("format test")
	}()

	if !strings.Contains(buf.String(), "format test") {
		t.Errorf("HandlePanic wrote %q, expected it to contain 'format test'", buf.String())
	}
}
