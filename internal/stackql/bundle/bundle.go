package bundle

import (
	"github.com/stackql/stackql/internal/stackql/garbagecollector"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
)

type Bundle interface {
	GetGC() garbagecollector.GarbageCollector
	GetNamespaceCollection() tablenamespace.TableNamespaceCollection
	GetSQLEngine() sqlengine.SQLEngine
}

func NewBundle(
	garbageCollector garbagecollector.GarbageCollector,
	namespaces tablenamespace.TableNamespaceCollection,
	sqlEngine sqlengine.SQLEngine,
) Bundle {
	return &simpleBundle{
		garbageCollector: garbageCollector,
		namespaces:       namespaces,
		sqlEngine:        sqlEngine,
	}
}

type simpleBundle struct {
	garbageCollector garbagecollector.GarbageCollector
	namespaces       tablenamespace.TableNamespaceCollection
	sqlEngine        sqlengine.SQLEngine
}

func (sb *simpleBundle) GetGC() garbagecollector.GarbageCollector {
	return sb.garbageCollector
}

func (sb *simpleBundle) GetSQLEngine() sqlengine.SQLEngine {
	return sb.sqlEngine
}

func (sb *simpleBundle) GetNamespaceCollection() tablenamespace.TableNamespaceCollection {
	return sb.namespaces
}
