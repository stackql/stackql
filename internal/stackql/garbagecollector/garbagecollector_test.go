package garbagecollector_test

import (
	"testing"

	"github.com/stackql/stackql/internal/stackql/garbagecollector"
)

func TestLegacyCollectionDoesNotTouchActiveQueryTables(t *testing.T) {
	// A nil backend deliberately fails if compatibility collection reaches SQL.
	gc := garbagecollector.NewGarbageCollector(nil)
	if err := gc.Collect(); err != nil {
		t.Fatal(err)
	}
	if err := gc.PurgeEphemeral(); err != nil {
		t.Fatal(err)
	}
	if err := gc.PurgeControlTables(); err != nil {
		t.Fatal(err)
	}
}
