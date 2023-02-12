package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/go-openapistackql/pkg/nomenclature"
	"github.com/stackql/stackql/internal/stackql/bundle"
	"github.com/stackql/stackql/internal/stackql/datasource/sql_datasource"
	"github.com/stackql/stackql/internal/stackql/dbmsinternal"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/garbagecollector"
	"github.com/stackql/stackql/internal/stackql/kstore"
	"github.com/stackql/stackql/internal/stackql/netutils"
	"github.com/stackql/stackql/internal/stackql/provider"
	"github.com/stackql/stackql/internal/stackql/sql_system"
	"github.com/stackql/stackql/internal/stackql/sqlcontrol"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
	"github.com/stackql/stackql/pkg/txncounter"

	lrucache "github.com/stackql/stackql-parser/go/cache"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"gopkg.in/yaml.v2"
)

var (
	_ HandlerContext = &standardHandlerContext{}
)

type HandlerContext interface {
	Clone() HandlerContext
	//
	GetASTFormatter() sqlparser.NodeFormatter
	GetAuthContext(providerName string) (*dto.AuthCtx, error)
	GetDBMSInternalRouter() dbmsinternal.DBMSInternalRouter
	GetProvider(providerName string) (provider.IProvider, error)
	GetSupportedProviders(extended bool) (map[string]map[string]interface{}, error)
	LogHTTPResponseMap(target interface{})
	//
	GetRawQuery() string
	GetQuery() string
	GetRuntimeContext() dto.RuntimeCtx
	GetProviders() map[string]provider.IProvider
	GetControlAttributes() sqlcontrol.ControlAttributes
	GetCurrentProvider() string
	GetAuthContexts() map[string]*dto.AuthCtx
	GetRegistry() openapistackql.RegistryAPI
	GetErrorPresentation() string
	GetOutfile() io.Writer
	GetOutErrFile() io.Writer
	GetLRUCache() *lrucache.LRUCache
	GetSQLDataSource(name string) (sql_datasource.SQLDataSource, bool)
	GetSQLEngine() sqlengine.SQLEngine
	GetSQLSystem() sql_system.SQLSystem
	GetGarbageCollector() garbagecollector.GarbageCollector
	GetDrmConfig() drm.DRMConfig
	GetTxnCounterMgr() txncounter.TxnCounterManager
	GetTxnStore() kstore.KStore
	GetNamespaceCollection() tablenamespace.TableNamespaceCollection
	GetFormatter() sqlparser.NodeFormatter
	GetPGInternalRouter() dbmsinternal.DBMSInternalRouter
	//
	SetCurrentProvider(string)
	SetOutfile(io.Writer)
	SetOutErrFile(io.Writer)
	SetQuery(string)
	SetRawQuery(string)
}

type standardHandlerContext struct {
	rawQuery            string
	query               string
	runtimeContext      dto.RuntimeCtx
	providers           map[string]provider.IProvider
	controlAttributes   sqlcontrol.ControlAttributes
	currentProvider     string
	authContexts        map[string]*dto.AuthCtx
	sqlDataSources      map[string]sql_datasource.SQLDataSource
	registry            openapistackql.RegistryAPI
	errorPresentation   string
	outfile             io.Writer
	outErrFile          io.Writer
	lRUCache            *lrucache.LRUCache
	sqlEngine           sqlengine.SQLEngine
	sqlSystem           sql_system.SQLSystem
	garbageCollector    garbagecollector.GarbageCollector
	drmConfig           drm.DRMConfig
	txnCounterMgr       txncounter.TxnCounterManager
	txnStore            kstore.KStore
	namespaceCollection tablenamespace.TableNamespaceCollection
	formatter           sqlparser.NodeFormatter
	pgInternalRouter    dbmsinternal.DBMSInternalRouter
}

func (hc *standardHandlerContext) SetCurrentProvider(p string) {
	hc.currentProvider = p
}

func (hc *standardHandlerContext) SetRawQuery(rq string) {
	hc.rawQuery = rq
}

func (hc *standardHandlerContext) SetQuery(q string) {
	hc.query = q
}

func (hc *standardHandlerContext) SetOutfile(outFile io.Writer)       { hc.outfile = outFile }
func (hc *standardHandlerContext) SetOutErrFile(outErrFile io.Writer) { hc.outErrFile = outErrFile }

func (hc *standardHandlerContext) GetRawQuery() string                         { return hc.rawQuery }
func (hc *standardHandlerContext) GetQuery() string                            { return hc.query }
func (hc *standardHandlerContext) GetRuntimeContext() dto.RuntimeCtx           { return hc.runtimeContext }
func (hc *standardHandlerContext) GetProviders() map[string]provider.IProvider { return hc.providers }
func (hc *standardHandlerContext) GetControlAttributes() sqlcontrol.ControlAttributes {
	return hc.controlAttributes
}
func (hc *standardHandlerContext) GetCurrentProvider() string               { return hc.currentProvider }
func (hc *standardHandlerContext) GetAuthContexts() map[string]*dto.AuthCtx { return hc.authContexts }
func (hc *standardHandlerContext) GetRegistry() openapistackql.RegistryAPI  { return hc.registry }
func (hc *standardHandlerContext) GetErrorPresentation() string             { return hc.errorPresentation }
func (hc *standardHandlerContext) GetOutfile() io.Writer                    { return hc.outfile }
func (hc *standardHandlerContext) GetOutErrFile() io.Writer                 { return hc.outErrFile }
func (hc *standardHandlerContext) GetLRUCache() *lrucache.LRUCache          { return hc.lRUCache }
func (hc *standardHandlerContext) GetSQLEngine() sqlengine.SQLEngine        { return hc.sqlEngine }
func (hc *standardHandlerContext) GetSQLSystem() sql_system.SQLSystem       { return hc.sqlSystem }
func (hc *standardHandlerContext) GetGarbageCollector() garbagecollector.GarbageCollector {
	return hc.garbageCollector
}
func (hc *standardHandlerContext) GetDrmConfig() drm.DRMConfig { return hc.drmConfig }
func (hc *standardHandlerContext) GetTxnCounterMgr() txncounter.TxnCounterManager {
	return hc.txnCounterMgr
}
func (hc *standardHandlerContext) GetTxnStore() kstore.KStore { return hc.txnStore }

//	func (hc *standardHandlerContext) GetNamespaceCollection() tablenamespace.TableNamespaceCollection {
//		return hc.namespaceCollection
//	}
func (hc *standardHandlerContext) GetFormatter() sqlparser.NodeFormatter { return hc.formatter }
func (hc *standardHandlerContext) GetPGInternalRouter() dbmsinternal.DBMSInternalRouter {
	return hc.pgInternalRouter
}

func getProviderMap(providerName string, providerDesc openapistackql.ProviderDescription) map[string]interface{} {
	latestVersion, err := providerDesc.GetLatestVersion()
	if err != nil {
		latestVersion = fmt.Sprintf("could not infer latest version due to error.  Suggested action is that you wipe the local provider directory.  Error =  '%s'", err.Error())
	}
	googleMap := map[string]interface{}{
		"name":    providerName,
		"version": latestVersion,
	}
	return googleMap
}

func getProviderMapExtended(providerName string, providerDesc openapistackql.ProviderDescription) map[string]interface{} {
	return getProviderMap(providerName, providerDesc)
}

func (hc *standardHandlerContext) GetSQLDataSource(name string) (sql_datasource.SQLDataSource, bool) {
	ac, ok := hc.sqlDataSources[name]
	return ac, ok
}

func (hc *standardHandlerContext) GetSupportedProviders(extended bool) (map[string]map[string]interface{}, error) {
	retVal := make(map[string]map[string]interface{})
	provs := hc.registry.ListLocallyAvailableProviders()
	// Supporting SQL data sources
	// These will be overwritten by any documented providers with the same name
	for k, _ := range hc.sqlDataSources {
		pn := k
		retVal[pn] = map[string]interface{}{
			"name": pn,
		}

	}
	for k, pd := range provs {
		pn := k
		if pn == "googleapis.com" {
			pn = "google"
		}
		if extended {
			retVal[pn] = getProviderMapExtended(pn, pd)
		} else {
			retVal[pn] = getProviderMap(pn, pd)
		}
	}
	return retVal, nil
}

func (hc *standardHandlerContext) GetASTFormatter() sqlparser.NodeFormatter {
	return hc.formatter
}

func (hc *standardHandlerContext) GetProvider(providerName string) (provider.IProvider, error) {
	var err error
	if providerName == "" {
		providerName = hc.runtimeContext.ProviderStr
	}
	if hc.namespaceCollection.GetAnalyticsCacheTableNamespaceConfigurator().IsAllowed(providerName) {
		providerName = hc.namespaceCollection.GetAnalyticsCacheTableNamespaceConfigurator().GetObjectName(providerName)
	}
	ds, err := nomenclature.ExtractProviderDesignation(providerName)
	if err != nil {
		return nil, err
	}
	prov, ok := hc.providers[providerName]
	if !ok {
		prov, err = provider.GetProvider(hc.runtimeContext, ds.Name, ds.Tag, hc.registry, hc.sqlSystem)
		if err == nil {
			hc.providers[providerName] = prov
			return prov, err
		}
		err = fmt.Errorf("cannot find provider = '%s': %s", providerName, err.Error())
	}
	return prov, err
}

func (hc *standardHandlerContext) LogHTTPResponseMap(target interface{}) {
	if target == nil {
		hc.outErrFile.Write([]byte("processed http response body not present\n"))
		return
	}
	if hc.runtimeContext.HTTPLogEnabled {
		switch target := target.(type) {
		case map[string]interface{}, []interface{}:
			b, err := json.MarshalIndent(target, "", "  ")
			if err != nil {
				hc.outErrFile.Write([]byte(fmt.Sprintf("processed http response body map '%v' colud not be marshalled; error: %s\n", target, err.Error())))
				return
			}
			if target != nil {
				hc.outErrFile.Write([]byte(fmt.Sprintf("processed http response body object: %s\n", string(b))))
			} else {
				hc.outErrFile.Write([]byte("processed http response body not present\n"))
			}
		default:
			if target != nil {
				hc.outErrFile.Write([]byte(fmt.Sprintf("processed http response body object: %v\n", target)))
			} else {
				hc.outErrFile.Write([]byte("processed http response body not present\n"))
			}
		}
	}
}

func (hc *standardHandlerContext) GetAuthContext(providerName string) (*dto.AuthCtx, error) {
	var err error
	if providerName == "" {
		providerName = hc.runtimeContext.ProviderStr
	}
	authCtx, ok := hc.authContexts[providerName]
	if !ok {
		err = fmt.Errorf("cannot find AUTH context for provider = '%s'", providerName)
	}
	return authCtx, err
}

func (hc *standardHandlerContext) GetNamespaceCollection() tablenamespace.TableNamespaceCollection {
	return hc.namespaceCollection
}

func (hc *standardHandlerContext) GetDBMSInternalRouter() dbmsinternal.DBMSInternalRouter {
	return hc.pgInternalRouter
}

func GetRegistry(runtimeCtx dto.RuntimeCtx) (openapistackql.RegistryAPI, error) {
	return getRegistry(runtimeCtx)
}

func getRegistry(runtimeCtx dto.RuntimeCtx) (openapistackql.RegistryAPI, error) {
	var rc openapistackql.RegistryConfig
	err := yaml.Unmarshal([]byte(runtimeCtx.RegistryRaw), &rc)
	if err != nil {
		return nil, err
	}
	if rc.LocalDocRoot == "" {
		if strings.HasPrefix(rc.RegistryURL, "file:") {
			rc.LocalDocRoot = path.Clean(path.Join(strings.TrimPrefix(rc.RegistryURL, "file:"), ".."))
		} else {
			rc.LocalDocRoot = runtimeCtx.ApplicationFilesRootPath
		}
	}
	rt := netutils.GetRoundTripper(runtimeCtx, nil)
	return openapistackql.NewRegistry(rc, rt)
}

func (hc *standardHandlerContext) Clone() HandlerContext {
	rv := standardHandlerContext{
		rawQuery:            hc.rawQuery,
		runtimeContext:      hc.runtimeContext,
		providers:           hc.providers,
		authContexts:        hc.authContexts,
		registry:            hc.registry,
		controlAttributes:   hc.controlAttributes,
		errorPresentation:   hc.errorPresentation,
		lRUCache:            hc.lRUCache,
		sqlEngine:           hc.sqlEngine,
		sqlDataSources:      hc.sqlDataSources,
		sqlSystem:           hc.sqlSystem,
		garbageCollector:    hc.garbageCollector,
		outErrFile:          hc.outErrFile,
		outfile:             hc.outfile,
		txnCounterMgr:       hc.txnCounterMgr,
		txnStore:            hc.txnStore,
		namespaceCollection: hc.namespaceCollection,
		formatter:           hc.formatter,
		pgInternalRouter:    hc.pgInternalRouter,
	}
	return &rv
}

func GetHandlerCtx(cmdString string, runtimeCtx dto.RuntimeCtx, lruCache *lrucache.LRUCache, inputBundle bundle.Bundle) (HandlerContext, error) {
	reg, err := getRegistry(runtimeCtx)
	if err != nil {
		return nil, err
	}
	providers := make(map[string]provider.IProvider)
	if err != nil {
		return nil, err
	}
	controlAttributes := inputBundle.GetControlAttributes()
	sqlEngine := inputBundle.GetSQLEngine()
	rv := standardHandlerContext{
		rawQuery:            cmdString,
		runtimeContext:      runtimeCtx,
		providers:           providers,
		authContexts:        inputBundle.GetAuthContexts(),
		registry:            reg,
		controlAttributes:   controlAttributes,
		errorPresentation:   runtimeCtx.ErrorPresentation,
		lRUCache:            lruCache,
		sqlEngine:           sqlEngine,
		sqlDataSources:      inputBundle.GetSQLDataSources(),
		sqlSystem:           inputBundle.GetSQLSystem(),
		garbageCollector:    inputBundle.GetGC(),
		txnCounterMgr:       inputBundle.GetTxnCounterManager(),
		txnStore:            inputBundle.GetTxnStore(),
		namespaceCollection: inputBundle.GetNamespaceCollection(),
		formatter:           inputBundle.GetSQLSystem().GetASTFormatter(),
		pgInternalRouter:    inputBundle.GetDBMSInternalRouter(),
	}
	drmCfg, err := drm.GetDRMConfig(inputBundle.GetSQLSystem(), rv.namespaceCollection, controlAttributes)
	if err != nil {
		return nil, err
	}
	rv.drmConfig = drmCfg
	return &rv, nil
}
