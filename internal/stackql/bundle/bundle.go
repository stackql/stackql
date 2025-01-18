package bundle

import (
	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/acid/txn_context"
	"github.com/stackql/stackql/internal/stackql/datasource/sql_datasource"
	"github.com/stackql/stackql/internal/stackql/dbmsinternal"
	"github.com/stackql/stackql/internal/stackql/garbagecollector"
	"github.com/stackql/stackql/internal/stackql/kstore"
	"github.com/stackql/stackql/internal/stackql/sql_system"
	"github.com/stackql/stackql/internal/stackql/sqlcontrol"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
	"github.com/stackql/stackql/internal/stackql/typing"
	"github.com/stackql/stackql/pkg/txncounter"
)

type Bundle interface {
	GetAuthContexts() map[string]*dto.AuthCtx
	GetControlAttributes() sqlcontrol.ControlAttributes
	GetGC() garbagecollector.GarbageCollector
	GetNamespaceCollection() tablenamespace.Collection
	GetDBMSInternalRouter() dbmsinternal.Router
	GetSQLDataSources() map[string]sql_datasource.SQLDataSource
	GetSQLSystem() sql_system.SQLSystem
	GetSQLEngine() sqlengine.SQLEngine
	GetTxnCounterManager() txncounter.Manager
	GetTxnStore() kstore.KStore
	GetTxnCoordinatorContext() txn_context.ITransactionCoordinatorContext
	GetTypingConfig() typing.Config
	GetSessionContext() dto.SessionContext
}

func NewBundle(
	garbageCollector garbagecollector.GarbageCollector,
	namespaces tablenamespace.Collection,
	sqlEngine sqlengine.SQLEngine,
	sqlSystem sql_system.SQLSystem,
	pgInternalRouter dbmsinternal.Router,
	controlAttributes sqlcontrol.ControlAttributes,
	txnStore kstore.KStore,
	txnCtrMgr txncounter.Manager,
	authContexts map[string]*dto.AuthCtx,
	sqlDataSources map[string]sql_datasource.SQLDataSource,
	txnCoordintatorContext txn_context.ITransactionCoordinatorContext,
	typCfg typing.Config,
	sessionCtx dto.SessionContext,
) Bundle {
	return &simpleBundle{
		garbageCollector:       garbageCollector,
		namespaces:             namespaces,
		sqlEngine:              sqlEngine,
		sqlSystem:              sqlSystem,
		controlAttributes:      controlAttributes,
		txnStore:               txnStore,
		txnCtrMgr:              txnCtrMgr,
		formatter:              sqlSystem.GetASTFormatter(),
		pgInternalRouter:       pgInternalRouter,
		authContexts:           authContexts,
		sqlDataSources:         sqlDataSources,
		txnCoordintatorContext: txnCoordintatorContext,
		typCfg:                 typCfg,
		sessionCtx:             sessionCtx,
	}
}

type simpleBundle struct {
	controlAttributes      sqlcontrol.ControlAttributes
	garbageCollector       garbagecollector.GarbageCollector
	namespaces             tablenamespace.Collection
	sqlEngine              sqlengine.SQLEngine
	sqlSystem              sql_system.SQLSystem
	txnStore               kstore.KStore
	txnCtrMgr              txncounter.Manager
	typCfg                 typing.Config
	formatter              sqlparser.NodeFormatter
	pgInternalRouter       dbmsinternal.Router
	sqlDataSources         map[string]sql_datasource.SQLDataSource
	authContexts           map[string]*dto.AuthCtx
	txnCoordintatorContext txn_context.ITransactionCoordinatorContext
	sessionCtx             dto.SessionContext
}

func (sb *simpleBundle) GetSessionContext() dto.SessionContext {
	return sb.sessionCtx
}

func (sb *simpleBundle) GetTxnCoordinatorContext() txn_context.ITransactionCoordinatorContext {
	return sb.txnCoordintatorContext
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

func (sb *simpleBundle) GetDBMSInternalRouter() dbmsinternal.Router {
	return sb.pgInternalRouter
}

func (sb *simpleBundle) GetASTFormatter() sqlparser.NodeFormatter {
	return sb.formatter
}

func (sb *simpleBundle) GetTxnStore() kstore.KStore {
	return sb.txnStore
}

func (sb *simpleBundle) GetTxnCounterManager() txncounter.Manager {
	return sb.txnCtrMgr
}

func (sb *simpleBundle) GetTypingConfig() typing.Config {
	return sb.typCfg
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

func (sb *simpleBundle) GetNamespaceCollection() tablenamespace.Collection {
	return sb.namespaces
}
