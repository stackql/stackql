package entryutil

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/stackql/stackql/internal/stackql/bundle"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/garbagecollector"
	"github.com/stackql/stackql/internal/stackql/gcexec"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/iqlerror"
	"github.com/stackql/stackql/internal/stackql/kstore"
	"github.com/stackql/stackql/internal/stackql/sqlcontrol"
	"github.com/stackql/stackql/internal/stackql/sqldialect"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"

	"github.com/stackql/stackql/pkg/preprocessor"
	"github.com/stackql/stackql/pkg/txncounter"

	lrucache "vitess.io/vitess/go/cache"
)

func BuildInputBundle(runtimeCtx dto.RuntimeCtx) (bundle.Bundle, error) {
	controlAttributes := sqlcontrol.GetControlAttributes("standard")
	sqlCfg, err := dto.GetSQLBackendCfg(runtimeCtx.SQLBackendCfgRaw)
	if err != nil {
		return nil, err
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
	dialect, err := sqldialect.NewSQLDialect(se, namespaces.GetAnalyticsCacheTableNamespaceConfigurator().GetLikeString(), controlAttributes, sqlCfg.SQLDialect)
	if err != nil {
		return nil, err
	}
	namespaces, err = namespaces.WithSQLDialect(dialect)
	if err != nil {
		return nil, err
	}
	txnStore, err := kstore.GetKStore(txnStoreCfg)
	if err != nil {
		return nil, err
	}
	gcExec, err := buildGCExec(se, namespaces, dialect, txnStore)
	if err != nil {
		return nil, err
	}
	gc := buildGC(gcExec, gcCfg, se)
	txnCtrMgr, err := getTxnCounterManager(se)
	if err != nil {
		return nil, err
	}
	return bundle.NewBundle(gc, namespaces, se, dialect, controlAttributes, txnStore, txnCtrMgr), nil
}

func initNamespaces(namespaceCfgRaw string, sqlEngine sqlengine.SQLEngine) (tablenamespace.TableNamespaceCollection, error) {
	cfgs, err := dto.GetNamespaceCfg(namespaceCfgRaw)
	if err != nil {
		return nil, err
	}
	return tablenamespace.NewStandardTableNamespaceCollection(cfgs, sqlEngine)
}

func buildSQLEngine(sqlCfg dto.SQLBackendCfg, controlAttributes sqlcontrol.ControlAttributes) (sqlengine.SQLEngine, error) {
	return sqlengine.NewSQLEngine(sqlCfg, controlAttributes)
}

func buildGCExec(sqlEngine sqlengine.SQLEngine, namespaces tablenamespace.TableNamespaceCollection, dialect sqldialect.SQLDialect, txnStore kstore.KStore) (gcexec.GarbageCollectorExecutor, error) {
	return gcexec.GetGarbageCollectorExecutorInstance(sqlEngine, namespaces, dialect, txnStore)
}

func buildGC(gcExec gcexec.GarbageCollectorExecutor, gcCfg dto.GCCfg, sqlEngine sqlengine.SQLEngine) garbagecollector.GarbageCollector {
	return garbagecollector.NewGarbageCollector(gcExec, gcCfg, sqlEngine)
}

func getTxnCounterManager(sqlEngine sqlengine.SQLEngine) (txncounter.TxnCounterManager, error) {
	genId, err := sqlEngine.GetCurrentGenerationId()
	if err != nil {
		genId, err = sqlEngine.GetNextGenerationId()
		if err != nil {
			return nil, err
		}
	}
	sessionId, err := sqlEngine.GetNextSessionId(genId)
	if err != nil {
		return nil, err
	}
	return txncounter.NewTxnCounterManager(genId, sessionId), nil
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
		prepRd, err = pp.Prepare(rdr, runtimeCtx.InfilePath)
		if err != nil {
			return nil, err
		}
	} else {
		externalTmplRdr, err = os.Open(runtimeCtx.TemplateCtxFilePath)
		if err != nil {
			return nil, err
		}
		prepRd = rdr
		err = pp.PrepareExternal(strings.Trim(strings.ToLower(filepath.Ext(runtimeCtx.TemplateCtxFilePath)), "."), externalTmplRdr, runtimeCtx.TemplateCtxFilePath)
	}
	if err != nil {
		return nil, err
	}
	ppRd, err := pp.Render(prepRd)
	if err != nil {
		return nil, err
	}
	var bb []byte
	bb, err = ioutil.ReadAll(ppRd)
	return bb, err
}

func BuildHandlerContext(runtimeCtx dto.RuntimeCtx, rdr io.Reader, lruCache *lrucache.LRUCache, inputBundle bundle.Bundle) (handler.HandlerContext, error) {
	bb, err := assemblePreprocessor(runtimeCtx, rdr)
	iqlerror.PrintErrorAndExitOneIfError(err)
	return handler.GetHandlerCtx(strings.TrimSpace(string(bb)), runtimeCtx, lruCache, inputBundle)
}

func BuildHandlerContextNoPreProcess(runtimeCtx dto.RuntimeCtx, lruCache *lrucache.LRUCache, inputBundle bundle.Bundle) (handler.HandlerContext, error) {
	return handler.GetHandlerCtx("", runtimeCtx, lruCache, inputBundle)
}
