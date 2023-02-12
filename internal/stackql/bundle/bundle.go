package bundle

import (
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/datasource/sql_datasource"
	"github.com/stackql/stackql/internal/stackql/dbmsinternal"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/garbagecollector"
	"github.com/stackql/stackql/internal/stackql/kstore"
	"github.com/stackql/stackql/internal/stackql/sql_system"
	"github.com/stackql/stackql/internal/stackql/sqlcontrol"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
	"github.com/stackql/stackql/pkg/txncounter"
)

type Bundle interface {
	GetAuthContexts() map[string]*dto.AuthCtx
	GetControlAttributes() sqlcontrol.ControlAttributes
	GetGC() garbagecollector.GarbageCollector
	GetNamespaceCollection() tablenamespace.TableNamespaceCollection
	GetDBMSInternalRouter() dbmsinternal.DBMSInternalRouter
	GetSQLDataSources() map[string]sql_datasource.SQLDataSource
	GetSQLSystem() sql_system.SQLSystem
	GetSQLEngine() sqlengine.SQLEngine
	GetTxnCounterManager() txncounter.TxnCounterManager
	GetTxnStore() kstore.KStore
}

func NewBundle(
	garbageCollector garbagecollector.GarbageCollector,
	namespaces tablenamespace.TableNamespaceCollection,
	sqlEngine sqlengine.SQLEngine,
	sqlSystem sql_system.SQLSystem,
	pgInternalRouter dbmsinternal.DBMSInternalRouter,
	controlAttributes sqlcontrol.ControlAttributes,
	txnStore kstore.KStore,
	txnCtrMgr txncounter.TxnCounterManager,
	authContexts map[string]*dto.AuthCtx,
	sqlDataSources map[string]sql_datasource.SQLDataSource,
) Bundle {
	return &simpleBundle{
		garbageCollector:  garbageCollector,
		namespaces:        namespaces,
		sqlEngine:         sqlEngine,
		sqlSystem:         sqlSystem,
		controlAttributes: controlAttributes,
		txnStore:          txnStore,
		txnCtrMgr:         txnCtrMgr,
		formatter:         sqlSystem.GetASTFormatter(),
		pgInternalRouter:  pgInternalRouter,
		authContexts:      authContexts,
		sqlDataSources:    sqlDataSources,
	}
}

type simpleBundle struct {
	controlAttributes sqlcontrol.ControlAttributes
	garbageCollector  garbagecollector.GarbageCollector
	namespaces        tablenamespace.TableNamespaceCollection
	sqlEngine         sqlengine.SQLEngine
	sqlSystem         sql_system.SQLSystem
	txnStore          kstore.KStore
	txnCtrMgr         txncounter.TxnCounterManager
	formatter         sqlparser.NodeFormatter
	pgInternalRouter  dbmsinternal.DBMSInternalRouter
	sqlDataSources    map[string]sql_datasource.SQLDataSource
	authContexts      map[string]*dto.AuthCtx
}

func (sb *simpleBundle) GetSQLDataSources() map[string]sql_datasource.SQLDataSource {
	return sb.sqlDataSources
}

func (sb *simpleBundle) GetAuthContexts() map[string]*dto.AuthCtx {
	return sb.authContexts
}

func (sb *simpleBundle) GetControlAttributes() sqlcontrol.ControlAttributes {
	return sb.controlAttributes
}

func (sb *simpleBundle) GetDBMSInternalRouter() dbmsinternal.DBMSInternalRouter {
	return sb.pgInternalRouter
}

func (sb *simpleBundle) GetASTFormatter() sqlparser.NodeFormatter {
	return sb.formatter
}

func (sb *simpleBundle) GetTxnStore() kstore.KStore {
	return sb.txnStore
}

func (sb *simpleBundle) GetTxnCounterManager() txncounter.TxnCounterManager {
	return sb.txnCtrMgr
}

func (sb *simpleBundle) GetGC() garbagecollector.GarbageCollector {
	return sb.garbageCollector
}

func (sb *simpleBundle) GetSQLEngine() sqlengine.SQLEngine {
	return sb.sqlEngine
}

func (sb *simpleBundle) GetSQLSystem() sql_system.SQLSystem {
	return sb.sqlSystem
}

func (sb *simpleBundle) GetNamespaceCollection() tablenamespace.TableNamespaceCollection {
	return sb.namespaces
}
