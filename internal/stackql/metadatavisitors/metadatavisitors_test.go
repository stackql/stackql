package metadatavisitors_test

import (
	"testing"

	"github.com/stackql/stackql/internal/stackql/metadatavisitors"
)

func TestNewTemplatedProduct(t *testing.T) {
	tp := metadatavisitors.NewTemplatedProduct("body-content", "placeholder-content")
	if tp == nil {
		t.Fatal("NewTemplatedProduct returned nil")
	}
}

func TestTemplatedProduct_GetBody(t *testing.T) {
	tp := metadatavisitors.NewTemplatedProduct("the-body", "the-placeholder")
	if got := tp.GetBody(); got != "the-body" {
		t.Errorf("GetBody() = %q, want 'the-body'", got)
	}
}

func TestTemplatedProduct_GetPlaceholder(t *testing.T) {
	tp := metadatavisitors.NewTemplatedProduct("the-body", "the-placeholder")
	if got := tp.GetPlaceholder(); got != "the-placeholder" {
		t.Errorf("GetPlaceholder() = %q, want 'the-placeholder'", got)
	}
}

func TestTemplatedProduct_EmptyBody(t *testing.T) {
	tp := metadatavisitors.NewTemplatedProduct("", "placeholder")
	if got := tp.GetBody(); got != "" {
		t.Errorf("GetBody() = %q, want ''", got)
	}
}

func TestTemplatedProduct_EmptyPlaceholder(t *testing.T) {
	tp := metadatavisitors.NewTemplatedProduct("body", "")
	if got := tp.GetPlaceholder(); got != "" {
		t.Errorf("GetPlaceholder() = %q, want ''", got)
	}
}

func TestTemplatedProduct_BothEmpty(t *testing.T) {
	tp := metadatavisitors.NewTemplatedProduct("", "")
	if got := tp.GetBody(); got != "" {
		t.Errorf("GetBody() = %q, want ''", got)
	}
	if got := tp.GetPlaceholder(); got != "" {
		t.Errorf("GetPlaceholder() = %q, want ''", got)
	}
}

func TestTemplatedProduct_IndependentInstances(t *testing.T) {
	tp1 := metadatavisitors.NewTemplatedProduct("body1", "ph1")
	tp2 := metadatavisitors.NewTemplatedProduct("body2", "ph2")
	if tp1.GetBody() == tp2.GetBody() {
		t.Error("two instances should have independent body values")
	}
	if tp1.GetPlaceholder() == tp2.GetPlaceholder() {
		t.Error("two instances should have independent placeholder values")
	}
}
