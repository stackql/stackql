package handler //nolint:testpackage // Verify session-owned storage without external persistence.

import (
	"context"
	"errors"
	"testing"

	"github.com/stackql-labs/omnisdk/public/secrets"
)

func TestSessionSecretsAreIsolatedAndDetached(t *testing.T) {
	ctx := context.Background()
	first, second := newSessionSecretStore(), newSessionSecretStore()
	input := []byte("first")
	if err := first.Set(ctx, "account", input); err != nil {
		t.Fatal(err)
	}
	input[0] = 'x'
	got, err := first.Get(ctx, "account")
	if err != nil || string(got) != "first" {
		t.Fatalf("stored value changed: %v", err)
	}
	got[0] = 'x'
	again, err := first.Get(ctx, "account")
	if err != nil || string(again) != "first" {
		t.Fatalf("returned value aliases storage: %v", err)
	}
	if _, err = second.Get(ctx, "account"); !errors.Is(err, secrets.ErrNotFound) {
		t.Fatal("secret leaked across stores")
	}
	if err = first.Delete(ctx, "account"); err != nil {
		t.Fatal(err)
	}
	if _, err = first.Get(ctx, "account"); !errors.Is(err, secrets.ErrNotFound) {
		t.Fatal("deleted secret remains")
	}
}

func TestSessionSecretsHonorCancellation(t *testing.T) {
	store := newSessionSecretStore()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := store.Set(ctx, "account", []byte("value")); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, "account"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, "account"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestClientContextsDoNotInheritSecretStores(t *testing.T) {
	parent := sessionTestHandler()
	if err := parent.SetConfigAtPath("secrets.parent", "private", ""); err != nil {
		t.Fatal(err)
	}
	first := parent.NewClientContext().(*standardHandlerContext)
	second := parent.NewClientContext().(*standardHandlerContext)
	if _, err := first.providerSessionState().store.Get(context.Background(), "parent"); !errors.Is(err, secrets.ErrNotFound) {
		t.Fatal("client inherited another session's secret")
	}
	if err := first.SetConfigAtPath("secrets.private", "first", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := second.providerSessionState().store.Get(context.Background(), "private"); !errors.Is(err, secrets.ErrNotFound) {
		t.Fatal("secret leaked to second client")
	}
	if err := first.SetConfigAtPath("secret_store", nil, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := first.providerSessionState().store.Get(context.Background(), "private"); !errors.Is(err, secrets.ErrNotFound) {
		t.Fatal("reset secret store retained values")
	}
}
