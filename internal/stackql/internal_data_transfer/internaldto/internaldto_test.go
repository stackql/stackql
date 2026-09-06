package internaldto_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
)

func TestNewBasicPrimitiveContext(t *testing.T) {
	var w, ew bytes.Buffer
	authFn := func(string) (*dto.AuthCtx, error) { return nil, nil }

	ctx := internaldto.NewBasicPrimitiveContext(authFn, &w, &ew)
	if ctx == nil {
		t.Fatal("NewBasicPrimitiveContext returned nil")
	}
}

func TestBasicPrimitiveContext_GetWriter(t *testing.T) {
	var buf bytes.Buffer
	authFn := func(string) (*dto.AuthCtx, error) { return nil, nil }

	ctx := internaldto.NewBasicPrimitiveContext(authFn, &buf, io.Discard)
	if got := ctx.GetWriter(); got != &buf {
		t.Error("GetWriter() did not return the expected writer")
	}
}

func TestBasicPrimitiveContext_GetErrWriter(t *testing.T) {
	var buf bytes.Buffer
	authFn := func(string) (*dto.AuthCtx, error) { return nil, nil }

	ctx := internaldto.NewBasicPrimitiveContext(authFn, io.Discard, &buf)
	if got := ctx.GetErrWriter(); got != &buf {
		t.Error("GetErrWriter() did not return the expected writer")
	}
}

func TestBasicPrimitiveContext_GetWriter_NilWriter(t *testing.T) {
	authFn := func(string) (*dto.AuthCtx, error) { return nil, nil }

	ctx := internaldto.NewBasicPrimitiveContext(authFn, nil, nil)
	if got := ctx.GetWriter(); got != nil {
		t.Errorf("GetWriter() = %v, want nil", got)
	}
	if got := ctx.GetErrWriter(); got != nil {
		t.Errorf("GetErrWriter() = %v, want nil", got)
	}
}

func TestBasicPrimitiveContext_GetAuthContext_Success(t *testing.T) {
	expected := &dto.AuthCtx{}
	authFn := func(prov string) (*dto.AuthCtx, error) {
		if prov != "test-provider" {
			t.Errorf("GetAuthContext() called with %q, want 'test-provider'", prov)
		}
		return expected, nil
	}

	ctx := internaldto.NewBasicPrimitiveContext(authFn, io.Discard, io.Discard)
	got, err := ctx.GetAuthContext("test-provider")
	if err != nil {
		t.Fatalf("GetAuthContext() error = %v", err)
	}
	if got != expected {
		t.Errorf("GetAuthContext() = %v, want %v", got, expected)
	}
}

func TestBasicPrimitiveContext_GetAuthContext_Error(t *testing.T) {
	wantErr := errors.New("auth failed")
	authFn := func(string) (*dto.AuthCtx, error) {
		return nil, wantErr
	}

	ctx := internaldto.NewBasicPrimitiveContext(authFn, io.Discard, io.Discard)
	_, err := ctx.GetAuthContext("provider")
	if !errors.Is(err, wantErr) {
		t.Errorf("GetAuthContext() error = %v, want %v", err, wantErr)
	}
}

func TestBasicPrimitiveContext_GetAuthContext_DifferentProviders(t *testing.T) {
	calls := make([]string, 0)
	authFn := func(prov string) (*dto.AuthCtx, error) {
		calls = append(calls, prov)
		return nil, nil
	}

	ctx := internaldto.NewBasicPrimitiveContext(authFn, io.Discard, io.Discard)
	ctx.GetAuthContext("provider-a")
	ctx.GetAuthContext("provider-b")
	ctx.GetAuthContext("provider-c")

	if len(calls) != 3 {
		t.Errorf("GetAuthContext() called %d times, want 3", len(calls))
	}
	if calls[0] != "provider-a" || calls[1] != "provider-b" || calls[2] != "provider-c" {
		t.Errorf("providers = %v, want [provider-a, provider-b, provider-c]", calls)
	}
}
