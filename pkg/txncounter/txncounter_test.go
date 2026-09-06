package txncounter_test

import (
	"testing"

	"github.com/stackql/stackql/pkg/txncounter"
)

func TestNewTxnCounterManager(t *testing.T) {
	mgr := txncounter.NewTxnCounterManager(42, 7)
	if mgr == nil {
		t.Fatal("NewTxnCounterManager returned nil")
	}
}

func TestTxnCounterManager_GetCurrentGenerationID(t *testing.T) {
	mgr := txncounter.NewTxnCounterManager(42, 7)
	got, err := mgr.GetCurrentGenerationID()
	if err != nil {
		t.Fatalf("GetCurrentGenerationID() error = %v", err)
	}
	if got != 42 {
		t.Errorf("GetCurrentGenerationID() = %d, want 42", got)
	}
}

func TestTxnCounterManager_GetCurrentSessionID(t *testing.T) {
	mgr := txncounter.NewTxnCounterManager(42, 7)
	got, err := mgr.GetCurrentSessionID()
	if err != nil {
		t.Fatalf("GetCurrentSessionID() error = %v", err)
	}
	if got != 7 {
		t.Errorf("GetCurrentSessionID() = %d, want 7", got)
	}
}

func TestTxnCounterManager_GetNextInsertID_StartsAtOne(t *testing.T) {
	mgr := txncounter.NewTxnCounterManager(0, 0)
	id1, err := mgr.GetNextInsertID()
	if err != nil {
		t.Fatalf("GetNextInsertID() error = %v", err)
	}
	if id1 != 1 {
		t.Errorf("first GetNextInsertID() = %d, want 1", id1)
	}
}

func TestTxnCounterManager_GetNextInsertID_Increments(t *testing.T) {
	mgr := txncounter.NewTxnCounterManager(0, 0)
	id1, _ := mgr.GetNextInsertID()
	id2, _ := mgr.GetNextInsertID()
	id3, _ := mgr.GetNextInsertID()
	if id1 == id2 || id2 == id3 || id1 == id3 {
		t.Errorf("insert IDs should be distinct: %d, %d, %d", id1, id2, id3)
	}
	if id2 != id1+1 {
		t.Errorf("id2 = %d, want %d", id2, id1+1)
	}
	if id3 != id2+1 {
		t.Errorf("id3 = %d, want %d", id3, id2+1)
	}
}

func TestTxnCounterManager_InsertIDPerInstance(t *testing.T) {
	// Each manager instance has its own insert ID sequence
	mgr1 := txncounter.NewTxnCounterManager(0, 0)
	mgr2 := txncounter.NewTxnCounterManager(0, 0)

	id1, _ := mgr1.GetNextInsertID()
	id2, _ := mgr2.GetNextInsertID()

	// Both should be 1 (each manager starts fresh)
	if id1 != 1 || id2 != 1 {
		t.Errorf("expected both managers to start at 1, got %d and %d", id1, id2)
	}
}

func TestTxnCounterManager_GetNextTxnID_StartsAtOne(t *testing.T) {
	mgr := txncounter.NewTxnCounterManager(0, 0)
	id, err := mgr.GetNextTxnID()
	if err != nil {
		t.Fatalf("GetNextTxnID() error = %v", err)
	}
	if id != 1 {
		t.Errorf("first GetNextTxnID() = %d, want 1", id)
	}
}

func TestTxnCounterManager_GetNextTxnID_Increments(t *testing.T) {
	mgr := txncounter.NewTxnCounterManager(0, 0)
	id1, _ := mgr.GetNextTxnID()
	id2, _ := mgr.GetNextTxnID()
	id3, _ := mgr.GetNextTxnID()
	if id2 != id1+1 {
		t.Errorf("id2 = %d, want %d", id2, id1+1)
	}
	if id3 != id2+1 {
		t.Errorf("id3 = %d, want %d", id3, id2+1)
	}
}

func TestTxnCounterManager_GetNextTxnID_Concurrent(t *testing.T) {
	mgr := txncounter.NewTxnCounterManager(0, 0)
	results := make(chan int, 100)

	// Launch 100 concurrent calls
	for i := 0; i < 100; i++ {
		go func() {
			id, _ := mgr.GetNextTxnID()
			results <- id
		}()
	}

	// Collect results
	var ids []int
	for i := 0; i < 100; i++ {
		ids = append(ids, <-results)
	}

	// All IDs should be unique and form a continuous sequence starting at some base
	seen := make(map[int]bool)
	for _, id := range ids {
		if seen[id] {
			t.Errorf("duplicate txn ID: %d", id)
		}
		seen[id] = true
	}
}

func TestTxnCounterManager_GetNextInsertID_Concurrent(t *testing.T) {
	mgr := txncounter.NewTxnCounterManager(0, 0)
	results := make(chan int, 100)

	for i := 0; i < 100; i++ {
		go func() {
			id, _ := mgr.GetNextInsertID()
			results <- id
		}()
	}

	var ids []int
	for i := 0; i < 100; i++ {
		ids = append(ids, <-results)
	}

	seen := make(map[int]bool)
	for _, id := range ids {
		if seen[id] {
			t.Errorf("duplicate insert ID: %d", id)
		}
		seen[id] = true
	}
}
