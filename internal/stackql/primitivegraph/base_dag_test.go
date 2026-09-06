package primitivegraph_test

import (
	"sync"
	"testing"

	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
)

// Executors append txn control counters to the graph from inside the
// errgroup, so concurrent UNION arms race on the slice unless it is guarded.
// Run with -race to catch a regression; the length check catches lost writes.
func TestAddTxnControlCountersConcurrent(t *testing.T) {
	const writers = 64
	const perWriter = 100
	holder := primitivegraph.NewPrimitiveGraphHolder(-1)
	var wg sync.WaitGroup
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < perWriter; i++ {
				holder.AddTxnControlCounters(internaldto.NewTxnControlCountersFromVals(id, id, i, i))
			}
		}(w)
	}
	wg.Wait()
	if got := len(holder.GetTxnControlCounterSlice()); got != writers*perWriter {
		t.Fatalf("expected %d txn control counters, got %d", writers*perWriter, got)
	}
}
