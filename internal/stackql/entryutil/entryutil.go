package entryutil

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/stackql/any-sdk/pkg/db/sqlcontrol"
	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/any-sdk/public/sqlengine"
	"github.com/stackql/stackql/internal/stackql/acid/txn_context"
	"github.com/stackql/stackql/internal/stackql/bundle"
	"github.com/stackql/stackql/internal/stackql/datasource/sql_datasource"
	"github.com/stackql/stackql/internal/stackql/dbmsinternal"
	"github.com/stackql/stackql/internal/stackql/garbagecollector"
	"github.com/stackql/stackql/internal/stackql/gcexec"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/iqlerror"
	"github.com/stackql/stackql/internal/stackql/kstore"
	"github.com/stackql/stackql/internal/stackql/sql_system"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
	"github.com/stackql/stackql/internal/stackql/typing"
	"gopkg.in/yaml.v2"

	"github.com/stackql/stackql/pkg/preprocessor"
	"github.com/stackql/stackql/pkg/txncounter"

	lrucache "github.com/stackql/stackql-parser/go/cache"
)

//nolint:funlen // let us not worry about tidyness in this boilerplate
func BuildInputBundle(runtimeCtx dto.RuntimeCtx) (bundle.Bundle, error) {
	controlAttributes := sqlcontrol.GetControlAttributes("standard")
	sqlCfg, err := dto.GetSQLBackendCfg(runtimeCtx.SQLBackendCfgRaw)
	if err != nil {
		return nil, err
	}
	typCfg, err := typing.NewTypingConfig(sqlCfg.GetSQLDialect())
	if err != nil {
		return nil, err
	}
	ac := make(map[string]*dto.AuthCtx)
	err = yaml.Unmarshal([]byte(runtimeCtx.AuthRaw), ac)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling auth: %w", err)
	}
	se, err := buildSQLEngine(sqlCfg, controlAttributes)
	if err != nil {
		return nil, err
	}
	namespaces, err := initNamespaces(runtimeCtx.NamespaceCfgRaw, se)
	if err != nil {
		return nil, err
	}
	gcCfg, err := dto.GetGCCfg(runtimeCtx.GCCfgRaw)
	if err != nil {
		return nil, err
	}
	txnStoreCfg, err := dto.GetKStoreCfg(runtimeCtx.StoreTxnCfgRaw)
	if err != nil {
		return nil, err
	}
	system, err := sql_system.NewSQLSystem(
		se,
		namespaces.GetAnalyticsCacheTableNamespaceConfigurator().GetLikeString(),
		controlAttributes,
		sqlCfg,
		ac, typCfg, runtimeCtx.ExportAlias)
	if err != nil {
		return nil, err
	}
	pgInternalCfg, err := dto.GetDBMSInternalCfg(runtimeCtx.DBInternalCfgRaw)
	if err != nil {
		return nil, err
	}
	pgInternal, err := dbmsinternal.GetDBMSInternalRouter(pgInternalCfg, system)
	if err != nil {
		return nil, err
	}
	namespaces, err = namespaces.WithSQLSystem(system)
	if err != nil {
		return nil, err
	}
	txnStore, err := kstore.GetKStore(txnStoreCfg)
	if err != nil {
		return nil, err
	}
	gcExec, err := buildGCExec(se, namespaces, system, txnStore)
	if err != nil {
		return nil, err
	}
	gc := buildGC(gcExec, gcCfg, se)
	txnCtrMgr, err := getTxnCounterManager(se)
	if err != nil {
		return nil, err
	}
	sqlDataSources, err := initSQLDataSources(ac)
	if err != nil {
		return nil, fmt.Errorf("error initializing SQL data sources: %w", err)
	}
	txnCoordinatorCfg, err := dto.GetTxnCoordinatorCfgCfg(runtimeCtx.ACIDCfgRaw)
	if err != nil {
		return nil, fmt.Errorf("error initializing Transaction Coordinator config: %w", err)
	}
	sessionConfig, sessionConfigErr := dto.NewSessionContext(runtimeCtx.SessionCtxRaw)
	if sessionConfigErr != nil {
		return nil, fmt.Errorf("error initializing session config: %w", sessionConfigErr)
	}
	txnCoordinatorCtx := txn_context.NewTransactionCoordinatorContext(txnCoordinatorCfg.GetMaxTxnDepth())
	return bundle.NewBundle(
		gc,
		namespaces,
		se,
		system,
		pgInternal,
		controlAttributes,
		txnStore,
		txnCtrMgr,
		ac,
		sqlDataSources,
		txnCoordinatorCtx,
		typCfg,
		sessionConfig,
	), nil
}

func initSQLDataSources(authContextMap map[string]*dto.AuthCtx) (map[string]sql_datasource.SQLDataSource, error) {
	rv := make(map[string]sql_datasource.SQLDataSource)
	for k, ac := range authContextMap {
		_, isSQLCfg := ac.GetSQLCfg()
		if isSQLCfg {
			sqlDataSource, err := sql_datasource.NewDataSource(ac, sql_datasource.NewGenericSQLDataSource)
			if err != nil {
				return nil, err
			}
			rv[k] = sqlDataSource
		}
	}
	return rv, nil
}

func initNamespaces(
	namespaceCfgRaw string,
	sqlEngine sqlengine.SQLEngine,
) (tablenamespace.Collection, error) {
	cfgs, err := dto.GetNamespaceCfg(namespaceCfgRaw)
	if err != nil {
		return nil, err
	}
	return tablenamespace.NewStandardTableNamespaceCollection(cfgs, sqlEngine)
}

func buildSQLEngine(
	sqlCfg dto.SQLBackendCfg,
	controlAttributes sqlcontrol.ControlAttributes,
) (sqlengine.SQLEngine, error) {
	return sqlengine.NewSQLEngine(sqlCfg, controlAttributes)
}

func buildGCExec(
	sqlEngine sqlengine.SQLEngine,
	namespaces tablenamespace.Collection,
	system sql_system.SQLSystem,
	txnStore kstore.KStore,
) (gcexec.GarbageCollectorExecutor, error) {
	return gcexec.GetGarbageCollectorExecutorInstance(sqlEngine, namespaces, system, txnStore)
}

func buildGC(
	gcExec gcexec.GarbageCollectorExecutor,
	gcCfg dto.GCCfg,
	sqlEngine sqlengine.SQLEngine,
) garbagecollector.GarbageCollector {
	return garbagecollector.NewGarbageCollector(gcExec, gcCfg, sqlEngine)
}

func getTxnCounterManager(sqlEngine sqlengine.SQLEngine) (txncounter.Manager, error) {
	genID, err := sqlEngine.GetCurrentGenerationID()
	if err != nil {
		genID, err = sqlEngine.GetNextGenerationID()
		if err != nil {
			return nil, err
		}
	}
	sessionID, err := sqlEngine.GetNextSessionID(genID)
	if err != nil {
		return nil, err
	}
	return txncounter.NewTxnCounterManager(genID, sessionID), nil
}

func PreprocessInline(runtimeCtx dto.RuntimeCtx, s string) (string, error) {
	rdr := strings.NewReader(s)
	bt, err := assemblePreprocessor(runtimeCtx, rdr)
	if err != nil || bt == nil {
		return s, err
	}
	return string(bt), nil
}

func assemblePreprocessor(runtimeCtx dto.RuntimeCtx, rdr io.Reader) ([]byte, error) {
	var err error
	var prepRd, externalTmplRdr io.Reader
	pp := preprocessor.NewPreprocessor(preprocessor.TripleLessThanToken, preprocessor.TripleGreaterThanToken)
	if pp == nil {
		return nil, fmt.Errorf("preprocessor error")
	}
	if runtimeCtx.TemplateCtxFilePath == "" {
		prepRd, err = pp.Prepare(rdr, runtimeCtx.InfilePath, runtimeCtx.VarList)
		if err != nil {
			return nil, err
		}
	} else {
		externalTmplRdr, err = os.Open(runtimeCtx.TemplateCtxFilePath)
		if err != nil {
			return nil, err
		}
		prepRd = rdr
		err = pp.PrepareExternal(
			strings.Trim(
				strings.ToLower(filepath.Ext(runtimeCtx.TemplateCtxFilePath)), "."),
			externalTmplRdr,
			runtimeCtx.TemplateCtxFilePath,
			runtimeCtx.VarList,
		)
	}
	if err != nil {
		return nil, err
	}
	ppRd, err := pp.Render(prepRd)
	if err != nil {
		return nil, err
	}
	var bb []byte
	bb, err = io.ReadAll(ppRd)
	return bb, err
}

func BuildHandlerContext(
	runtimeCtx dto.RuntimeCtx,
	rdr io.Reader,
	lruCache *lrucache.LRUCache,
	inputBundle bundle.Bundle,
	isPreprocess bool,
) (handler.HandlerContext, error) {
	if !isPreprocess {
		return handler.NewHandlerCtx(
			"", runtimeCtx, lruCache,
			inputBundle, "v0.1.1")
	}
	bb, err := assemblePreprocessor(runtimeCtx, rdr)
	iqlerror.PrintErrorAndExitOneIfError(err)
	return handler.NewHandlerCtx(
		strings.TrimSpace(string(bb)), runtimeCtx, lruCache,
		inputBundle, "v0.1.1")
}
