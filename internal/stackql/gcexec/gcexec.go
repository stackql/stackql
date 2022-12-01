package gcexec

import (
	"sync"

	"github.com/stackql/stackql/internal/stackql/internaldto"
	"github.com/stackql/stackql/internal/stackql/kstore"
	"github.com/stackql/stackql/internal/stackql/sqldialect"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
)

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
func GetGarbageCollectorExecutorInstance(sqlEngine sqlengine.SQLEngine, ns tablenamespace.TableNamespaceCollection, dialect sqldialect.SQLDialect, txnStore kstore.KStore) (GarbageCollectorExecutor, error) {
	var err error
	once.Do(func() {
		if err != nil {
			return
		}
		garbageCollectorExecutor, err = newBasicGarbageCollectorExecutor(dialect, ns, txnStore)
	})
	return garbageCollectorExecutor, err
}

func newBasicGarbageCollectorExecutor(dialect sqldialect.SQLDialect, ns tablenamespace.TableNamespaceCollection, txnStore kstore.KStore) (GarbageCollectorExecutor, error) {
	return &basicGarbageCollectorExecutor{
		gcMutex:    &sync.Mutex{},
		ns:         ns,
		sqlDialect: dialect,
		txnStore:   txnStore,
	}, nil
}

// Algorithm summary:
//   - `Collect()` will reclaim resources from all txns **not** < supplied min ID.
type basicGarbageCollectorExecutor struct {
	gcMutex    *sync.Mutex
	ns         tablenamespace.TableNamespaceCollection
	sqlDialect sqldialect.SQLDialect
	txnStore   kstore.KStore
}

func (rc *basicGarbageCollectorExecutor) Update(tableName string, parentTcc, tcc internaldto.TxnControlCounters) error {
	rc.gcMutex.Lock()
	defer rc.gcMutex.Unlock()
	if rc.ns.GetAnalyticsCacheTableNamespaceConfigurator().IsAllowed(tableName) {
		templatedName, err := rc.ns.GetAnalyticsCacheTableNamespaceConfigurator().RenderTemplate(tableName)
		if err != nil {
			return err
		}
		err = rc.sqlDialect.GCAdd(templatedName, parentTcc, tcc)
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
	minId, minValid := rc.txnStore.Min()
	if !minValid {
		return rc.sqlDialect.GCCollectAll()
	}
	return rc.sqlDialect.GCCollectObsoleted(minId)
}

// Algorithm, **must be done during pause**:
//   - Obtain **minimum** active transaction.
//   - Retrieve GC queries from control table.
//   - Execute GC queries in a txn.
func (rc *basicGarbageCollectorExecutor) Purge() error {
	rc.gcMutex.Lock()
	defer rc.gcMutex.Unlock()
	return rc.sqlDialect.PurgeAll()
}

func (rc *basicGarbageCollectorExecutor) PurgeCache() error {
	rc.gcMutex.Lock()
	defer rc.gcMutex.Unlock()
	return rc.sqlDialect.GCPurgeCache()
}

func (rc *basicGarbageCollectorExecutor) PurgeControlTables() error {
	rc.gcMutex.Lock()
	defer rc.gcMutex.Unlock()
	return rc.sqlDialect.GCControlTablesPurge()
}

func (rc *basicGarbageCollectorExecutor) PurgeEphemeral() error {
	rc.gcMutex.Lock()
	defer rc.gcMutex.Unlock()
	return rc.sqlDialect.GCPurgeEphemeral()
}
