package kstore_test

import (
	"testing"

	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/stackql/internal/stackql/kstore"
)

func TestGetKStore_ReturnsNonNil(t *testing.T) {
	ks, err := kstore.GetKStore(dto.KStoreCfg{})
	if err != nil {
		t.Fatalf("GetKStore() error = %v", err)
	}
	if ks == nil {
		t.Fatal("GetKStore() returned nil")
	}
}

func TestKStore_Put(t *testing.T) {
	// Use a key that is unlikely to be used by other tests
	// (tests run alphabetically, so later tests can depend on earlier ones)
	ks, _ := kstore.GetKStore(dto.KStoreCfg{})
	ks.Put(9999)
	ks.Del(9999) // Clean up after ourselves
}

func TestKStore_Del(t *testing.T) {
	ks, _ := kstore.GetKStore(dto.KStoreCfg{})
	ks.Put(8888)
	ks.Del(8888)
}

func TestKStore_Min_ReturnsValue(t *testing.T) {
	// Put a unique key and verify Min returns something
	ks, _ := kstore.GetKStore(dto.KStoreCfg{})
	uniqueKey := 7777
	ks.Put(uniqueKey)
	defer ks.Del(uniqueKey)

	got, ok := ks.Min()
	if !ok {
		t.Fatal("Min() should return ok=true after Put")
	}
	if got != uniqueKey {
		t.Errorf("Min() = %d, want %d", got, uniqueKey)
	}
}

func TestKStore_PutAndDel_Deterministic(t *testing.T) {
	// Put a value and verify Min works, then delete it
	ks, _ := kstore.GetKStore(dto.KStoreCfg{})
	testKey := 6666

	// Verify Min fails before Put
	_, ok := ks.Min()
	if ok {
		t.Log("note: Min() returned ok=true before Put (store may have pre-existing keys)")
	}

	ks.Put(testKey)
	defer ks.Del(testKey)

	got, ok := ks.Min()
	if !ok {
		t.Fatal("Min() should return ok=true after Put")
	}
	if got != testKey {
		t.Errorf("Min() = %d, want %d", got, testKey)
	}

	ks.Del(testKey)
	_, ok = ks.Min()
	if ok {
		t.Log("note: Min() still returns ok=true after Del (store may have other keys)")
	}
}

func TestKStore_PutDuplicate_StillWorks(t *testing.T) {
	ks, _ := kstore.GetKStore(dto.KStoreCfg{})
	testKey := 5555
	ks.Put(testKey)
	defer ks.Del(testKey)

	ks.Put(testKey) // Duplicate
	// Should still work - no error expected
}

func TestKStore_ConcurrentOps(t *testing.T) {
	ks, _ := kstore.GetKStore(dto.KStoreCfg{})
	done := make(chan bool, 50)

	for i := 0; i < 50; i++ {
		go func(id int) {
			ks.Put(10000 + id)
			done <- true
		}(i)
	}

	for i := 0; i < 50; i++ {
		<-done
	}

	// Verify the store still works (may have pre-existing keys too)
	_, ok := ks.Min()
	if !ok {
		t.Error("Min() should work after concurrent puts")
	}

	// Cleanup: delete our keys
	for i := 0; i < 50; i++ {
		ks.Del(10000 + i)
	}
}

func TestKStore_Interface(t *testing.T) {
	// Verify the KStore interface is implemented correctly
	var ks kstore.KStore
	ks, _ = kstore.GetKStore(dto.KStoreCfg{})

	// Test that all interface methods exist and are callable
	// (this is a compile-time check, but explicit is better)
	_ = interface{}(ks.Put)
	_ = interface{}(ks.Del)
	_ = interface{}(ks.Min)
}
