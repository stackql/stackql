package txn_context_test

import (
	"testing"

	. "github.com/stackql/stackql/internal/stackql/acid/txn_context"
)

func TestNewTransactionContext(t *testing.T) {
	expectedDepth := 1
	tc := NewTransactionContext(expectedDepth)
	if tc.GetStackDepth() != expectedDepth {
		t.Fatal("test failed, mismatch stack depth for txn coordinator context")
	}
}
