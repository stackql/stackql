package garbagecollector

import (
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/gcexec"
	"github.com/stackql/stackql/internal/stackql/internaldto"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
)

var (
	_ GarbageCollector = &standardGarbageCollector{}
)

type GarbageCollector interface {
	Close() error
	Collect() error
	Purge() error
	PurgeCache() error
	PurgeControlTables() error
	PurgeEphemeral() error
	Update(string, internaldto.TxnControlCounters, internaldto.TxnControlCounters) error
}

func NewGarbageCollector(gcExecutor gcexec.GarbageCollectorExecutor, gcCfg dto.GCCfg, sqlEngine sqlengine.SQLEngine) GarbageCollector {
	return newStandardGarbageCollector(gcExecutor, gcCfg, sqlEngine)
}

func newStandardGarbageCollector(gcExecutor gcexec.GarbageCollectorExecutor, policy dto.GCCfg, sqlEngine sqlengine.SQLEngine) GarbageCollector {
	return &standardGarbageCollector{
		gcExecutor: gcExecutor,
		isEager:    policy.IsEager,
		sqlEngine:  sqlEngine,
	}
}

type standardGarbageCollector struct {
	gcExecutor gcexec.GarbageCollectorExecutor
	isEager    bool
	sqlEngine  sqlengine.SQLEngine
}

func (gc *standardGarbageCollector) Update(tableName string, parentTcc, tcc internaldto.TxnControlCounters) error {
	return gc.gcExecutor.Update(tableName, parentTcc, tcc)
}

func (gc *standardGarbageCollector) Close() error {
	if gc.isEager {
		return gc.gcExecutor.Collect()
	}
	return nil
}

func (gc *standardGarbageCollector) Collect() error {
	return gc.gcExecutor.Collect()
}

func (gc *standardGarbageCollector) Purge() error {
	return gc.gcExecutor.Purge()
}

func (gc *standardGarbageCollector) PurgeEphemeral() error {
	return gc.gcExecutor.PurgeEphemeral()
}

func (gc *standardGarbageCollector) PurgeCache() error {
	return gc.gcExecutor.PurgeCache()
}

func (gc *standardGarbageCollector) PurgeControlTables() error {
	return gc.gcExecutor.PurgeControlTables()
}
