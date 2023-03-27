package gcexec

import (
	"sync"

	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/kstore"
	"github.com/stackql/stackql/internal/stackql/sql_system"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
)

//nolint:gochecknoglobals // supporting singleton
var (
	once                     sync.Once
	garbageCollectorExecutor GarbageCollectorExecutor
)

type BrutalGarbageCollectorExecutor interface {
	Purge() error
	PurgeCache() error
	PurgeControlTables() error
	PurgeEphemeral() error
}

type AbstractFlatGarbageCollectorExecutor interface {
	Update(string, internaldto.TxnControlCounters, internaldto.TxnControlCounters) error
	Collect() error
}

type GarbageCollectorExecutor interface {
	BrutalGarbageCollectorExecutor
	AbstractFlatGarbageCollectorExecutor
}

// Idiomatic golang singleton
// Credit to http://marcio.io/2015/07/singleton-pattern-in-go/
func GetGarbageCollectorExecutorInstance(
	sqlEngine sqlengine.SQLEngine, //nolint:revive // future proofing
	ns tablenamespace.Collection,
	system sql_system.SQLSystem, txnStore kstore.KStore,
) (GarbageCollectorExecutor, error) {
	var err error
	once.Do(func() {
		if err != nil {
			return
		}
		garbageCollectorExecutor, err = newBasicGarbageCollectorExecutor(system, ns, txnStore)
	})
	return garbageCollectorExecutor, err
}

func newBasicGarbageCollectorExecutor(
	system sql_system.SQLSystem,
	ns tablenamespace.Collection,
	txnStore kstore.KStore,
) (GarbageCollectorExecutor, error) { //nolint:unparam // future proofing
	return &basicGarbageCollectorExecutor{
		gcMutex:   &sync.Mutex{},
		ns:        ns,
		sqlSystem: system,
		txnStore:  txnStore,
	}, nil
}

// Algorithm summary:
//   - `Collect()` will reclaim resources from all txns **not** < supplied min ID.
type basicGarbageCollectorExecutor struct {
	gcMutex   *sync.Mutex
	ns        tablenamespace.Collection
	sqlSystem sql_system.SQLSystem
	txnStore  kstore.KStore
}

func (rc *basicGarbageCollectorExecutor) Update(tableName string, parentTcc, tcc internaldto.TxnControlCounters) error {
	rc.gcMutex.Lock()
	defer rc.gcMutex.Unlock()
	if rc.ns.GetAnalyticsCacheTableNamespaceConfigurator().IsAllowed(tableName) {
		templatedName, err := rc.ns.GetAnalyticsCacheTableNamespaceConfigurator().RenderTemplate(tableName)
		if err != nil {
			return err
		}
		err = rc.sqlSystem.GCAdd(templatedName, parentTcc, tcc)
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

// Algorithm, **must be done during pause**:
//   - Obtain **minimum** active transaction.
//   - Retrieve GC queries from control table.
//   - Execute GC queries in a txn.
func (rc *basicGarbageCollectorExecutor) Collect() error {
	rc.gcMutex.Lock()
	defer rc.gcMutex.Unlock()
	minID, minValid := rc.txnStore.Min()
	if !minValid {
		return rc.sqlSystem.GCCollectAll()
	}
	return rc.sqlSystem.GCCollectObsoleted(minID)
}

// Algorithm, **must be done during pause**:
//   - Obtain **minimum** active transaction.
//   - Retrieve GC queries from control table.
//   - Execute GC queries in a txn.
func (rc *basicGarbageCollectorExecutor) Purge() error {
	rc.gcMutex.Lock()
	defer rc.gcMutex.Unlock()
	return rc.sqlSystem.PurgeAll()
}

func (rc *basicGarbageCollectorExecutor) PurgeCache() error {
	rc.gcMutex.Lock()
	defer rc.gcMutex.Unlock()
	return rc.sqlSystem.GCPurgeCache()
}

func (rc *basicGarbageCollectorExecutor) PurgeControlTables() error {
	rc.gcMutex.Lock()
	defer rc.gcMutex.Unlock()
	return rc.sqlSystem.GCControlTablesPurge()
}

func (rc *basicGarbageCollectorExecutor) PurgeEphemeral() error {
	rc.gcMutex.Lock()
	defer rc.gcMutex.Unlock()
	return rc.sqlSystem.GCPurgeEphemeral()
}
