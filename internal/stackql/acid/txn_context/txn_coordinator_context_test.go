package txn_context_test

import (
	"testing"

	. "github.com/stackql/stackql/internal/stackql/acid/txn_context"
)

func TestNewTransactionCoordinatorContext(t *testing.T) {
	expectedMaxDepth := 1
	tcc := NewTransactionCoordinatorContext(expectedMaxDepth)
	if tcc.GetMaxStackDepth() != expectedMaxDepth {
		t.Fatal("test failed, mismatch max stack depth for txn context")
	}
}
