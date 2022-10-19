package garbagecollector

import (
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/gcexec"
	"github.com/stackql/stackql/internal/stackql/tableinsertioncontainer"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
)

type GarbageCollector interface {
	AddInsertContainer(tm *tablemetadata.ExtendedTableMetadata) tableinsertioncontainer.TableInsertionContainer
	Close() error
	Collect() error
}

func NewGarbageCollector(gcExecutor gcexec.GarbageCollectorExecutor, gcCfg dto.GCCfg) GarbageCollector {
	return newStandardGarbageCollector(gcExecutor, gcCfg)
}

func newStandardGarbageCollector(gcExecutor gcexec.GarbageCollectorExecutor, policy dto.GCCfg) GarbageCollector {
	return &standardGarbageCollector{
		gcExecutor: gcExecutor,
		isEager:    policy.IsEager,
	}
}

type standardGarbageCollector struct {
	gcExecutor       gcexec.GarbageCollectorExecutor
	insertContainers []tableinsertioncontainer.TableInsertionContainer
	isEager          bool
}

func (gc *standardGarbageCollector) Close() error {
	for _, ic := range gc.insertContainers {
		gc.gcExecutor.Condemn(ic.GetTableTxnCounters())
	}
	if gc.isEager {
		return gc.gcExecutor.Collect()
	}
	return nil
}

func (gc *standardGarbageCollector) Collect() error {
	return gc.gcExecutor.Collect()
}

func (gc *standardGarbageCollector) AddInsertContainer(tm *tablemetadata.ExtendedTableMetadata) tableinsertioncontainer.TableInsertionContainer {
	rv := tableinsertioncontainer.NewTableInsertionContainer(tm)
	gc.insertContainers = append(gc.insertContainers, rv)
	return rv
}

func (gc *standardGarbageCollector) GetGarbageCollectorExecutor() gcexec.GarbageCollectorExecutor {
	return gc.gcExecutor
}

func (gc *standardGarbageCollector) GetInsertContainers() []tableinsertioncontainer.TableInsertionContainer {
	return gc.insertContainers
}
