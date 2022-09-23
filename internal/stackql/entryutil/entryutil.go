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
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"

	"github.com/stackql/stackql/pkg/preprocessor"
	"github.com/stackql/stackql/pkg/txncounter"

	lrucache "vitess.io/vitess/go/cache"
)

func BuildInputBundle(runtimeCtx dto.RuntimeCtx) (bundle.Bundle, error) {
	se, err := buildSQLEngine(runtimeCtx)
	if err != nil {
		return nil, err
	}
	namespaces, err := initNamespaces(runtimeCtx.NamespaceCfgRaw, se)
	if err != nil {
		return nil, err
	}
	gcCfg, err := dto.GetGCCfg("")
	if err != nil {
		return nil, err
	}
	gcExec, err := buildGCExec(se, namespaces, runtimeCtx)
	if err != nil {
		return nil, err
	}
	gc := buildGC(gcExec, gcCfg)
	return bundle.NewBundle(gc, namespaces, se), nil
}

func initNamespaces(namespaceCfgRaw string, sqlEngine sqlengine.SQLEngine) (tablenamespace.TableNamespaceCollection, error) {
	cfgs, err := dto.GetNamespaceCfg(namespaceCfgRaw)
	if err != nil {
		return nil, err
	}
	return tablenamespace.NewStandardTableNamespaceCollection(cfgs, sqlEngine)
}

func buildSQLEngine(runtimeCtx dto.RuntimeCtx) (sqlengine.SQLEngine, error) {
	sqlCfg := sqlengine.NewSQLEngineConfig(runtimeCtx)
	return sqlengine.NewSQLEngine(sqlCfg)
}

func buildGCExec(sqlEngine sqlengine.SQLEngine, namespaces tablenamespace.TableNamespaceCollection, runtimeCtx dto.RuntimeCtx) (gcexec.GarbageCollectorExecutor, error) {
	return gcexec.GetGarbageCollectorExecutorInstance(sqlEngine, namespaces, "sqlite")
}

func buildGC(gcExec gcexec.GarbageCollectorExecutor, gcCfg dto.GCCfg) garbagecollector.GarbageCollector {
	return garbagecollector.NewGarbageCollector(gcExec, gcCfg)
}

func GetTxnCounterManager(handlerCtx handler.HandlerContext) (*txncounter.TxnCounterManager, error) {
	genId, err := handlerCtx.SQLEngine.GetCurrentGenerationId()
	if err != nil {
		genId, err = handlerCtx.SQLEngine.GetNextGenerationId()
		if err != nil {
			return nil, err
		}
	}
	sessionId, err := handlerCtx.SQLEngine.GetNextSessionId(genId)
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
