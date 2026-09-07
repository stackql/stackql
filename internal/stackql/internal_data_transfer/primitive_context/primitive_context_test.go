package primitive_context_test

import (
	"testing"

	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/primitive_context"
)

func TestNewPrimitiveContext(t *testing.T) {
	pc := primitive_context.NewPrimitiveContext()
	if pc == nil {
		t.Fatal("NewPrimitiveContext returned nil")
	}
}

func TestPrimitiveContextDefaultIsReadOnly(t *testing.T) {
	pc := primitive_context.NewPrimitiveContext()
	if pc.IsReadOnly() {
		t.Error("expected IsReadOnly to be false by default")
	}
}

func TestPrimitiveContextSetIsReadOnly(t *testing.T) {
	pc := primitive_context.NewPrimitiveContext()

	pc.SetIsReadOnly(true)
	if !pc.IsReadOnly() {
		t.Error("expected IsReadOnly to be true after SetIsReadOnly(true)")
	}

	pc.SetIsReadOnly(false)
	if pc.IsReadOnly() {
		t.Error("expected IsReadOnly to be false after SetIsReadOnly(false)")
	}
}

func TestPrimitiveContextSetIsReadOnlyFalseByDefault(t *testing.T) {
	pc := primitive_context.NewPrimitiveContext()
	pc.SetIsReadOnly(false)
	if pc.IsReadOnly() {
		t.Error("expected IsReadOnly to remain false")
	}
}

func TestPrimitiveContextSetIsReadOnlyTrue(t *testing.T) {
	pc := primitive_context.NewPrimitiveContext()
	pc.SetIsReadOnly(true)
	if !pc.IsReadOnly() {
		t.Error("expected IsReadOnly to be true")
	}
}

func TestPrimitiveContextMultipleSetIsReadOnly(t *testing.T) {
	pc := primitive_context.NewPrimitiveContext()

	// Toggle multiple times
	pc.SetIsReadOnly(true)
	if !pc.IsReadOnly() {
		t.Error("expected IsReadOnly to be true")
	}

	pc.SetIsReadOnly(false)
	if pc.IsReadOnly() {
		t.Error("expected IsReadOnly to be false")
	}

	pc.SetIsReadOnly(true)
	if !pc.IsReadOnly() {
		t.Error("expected IsReadOnly to be true")
	}

	pc.SetIsReadOnly(false)
	if pc.IsReadOnly() {
		t.Error("expected IsReadOnly to be false")
	}
}

func TestPrimitiveContextMultipleInstances(t *testing.T) {
	pc1 := primitive_context.NewPrimitiveContext()
	pc2 := primitive_context.NewPrimitiveContext()

	// pc1 is read-only
	pc1.SetIsReadOnly(true)

	// pc2 should not be affected
	if pc2.IsReadOnly() {
		t.Error("pc2 should not be read-only when pc1 is read-only")
	}

	// pc2 is read-only
	pc2.SetIsReadOnly(true)

	// pc1 should not be affected
	if !pc1.IsReadOnly() {
		t.Error("pc1 should remain read-only")
	}
}
