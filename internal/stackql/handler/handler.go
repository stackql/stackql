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
	"github.com/stackql/stackql/internal/stackql/dbmsinternal"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/garbagecollector"
	"github.com/stackql/stackql/internal/stackql/kstore"
	"github.com/stackql/stackql/internal/stackql/netutils"
	"github.com/stackql/stackql/internal/stackql/provider"
	"github.com/stackql/stackql/internal/stackql/sqlcontrol"
	"github.com/stackql/stackql/internal/stackql/sqldialect"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
	"github.com/stackql/stackql/pkg/txncounter"

	"gopkg.in/yaml.v2"
	lrucache "vitess.io/vitess/go/cache"
	"vitess.io/vitess/go/vt/sqlparser"
)

type HandlerContext struct {
	RawQuery            string
	Query               string
	RuntimeContext      dto.RuntimeCtx
	providers           map[string]provider.IProvider
	ControlAttributes   sqlcontrol.ControlAttributes
	CurrentProvider     string
	authContexts        map[string]*dto.AuthCtx
	Registry            openapistackql.RegistryAPI
	ErrorPresentation   string
	Outfile             io.Writer
	OutErrFile          io.Writer
	LRUCache            *lrucache.LRUCache
	SQLEngine           sqlengine.SQLEngine
	SQLDialect          sqldialect.SQLDialect
	GarbageCollector    garbagecollector.GarbageCollector
	DrmConfig           drm.DRMConfig
	TxnCounterMgr       txncounter.TxnCounterManager
	TxnStore            kstore.KStore
	namespaceCollection tablenamespace.TableNamespaceCollection
	formatter           sqlparser.NodeFormatter
	pgInternalRouter    dbmsinternal.DBMSInternalRouter
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

func (hc *HandlerContext) GetSupportedProviders(extended bool) map[string]map[string]interface{} {
	retVal := make(map[string]map[string]interface{})
	provs := hc.Registry.ListLocallyAvailableProviders()
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
	return retVal
}

func (hc *HandlerContext) GetASTFormatter() sqlparser.NodeFormatter {
	return hc.formatter
}

func (hc *HandlerContext) GetProvider(providerName string) (provider.IProvider, error) {
	var err error
	if providerName == "" {
		providerName = hc.RuntimeContext.ProviderStr
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
		prov, err = provider.GetProvider(hc.RuntimeContext, ds.Name, ds.Tag, hc.Registry, hc.SQLEngine)
		if err == nil {
			hc.providers[providerName] = prov
			return prov, err
		}
		err = fmt.Errorf("cannot find provider = '%s': %s", providerName, err.Error())
	}
	return prov, err
}

func (hc *HandlerContext) LogHTTPResponseMap(target interface{}) {
	if target == nil {
		hc.OutErrFile.Write([]byte("processed http response body not present\n"))
		return
	}
	if hc.RuntimeContext.HTTPLogEnabled {
		switch target := target.(type) {
		case map[string]interface{}, []interface{}:
			b, err := json.MarshalIndent(target, "", "  ")
			if err != nil {
				hc.OutErrFile.Write([]byte(fmt.Sprintf("processed http response body map '%v' colud not be marshalled; error: %s\n", target, err.Error())))
				return
			}
			if target != nil {
				hc.OutErrFile.Write([]byte(fmt.Sprintf("processed http response body object: %s\n", string(b))))
			} else {
				hc.OutErrFile.Write([]byte("processed http response body not present\n"))
			}
		default:
			if target != nil {
				hc.OutErrFile.Write([]byte(fmt.Sprintf("processed http response body object: %v\n", target)))
			} else {
				hc.OutErrFile.Write([]byte("processed http response body not present\n"))
			}
		}
	}
}

func (hc *HandlerContext) GetAuthContext(providerName string) (*dto.AuthCtx, error) {
	var err error
	if providerName == "" {
		providerName = hc.RuntimeContext.ProviderStr
	}
	authCtx, ok := hc.authContexts[providerName]
	if !ok {
		err = fmt.Errorf("cannot find AUTH context for provider = '%s'", providerName)
	}
	return authCtx, err
}

func (hc *HandlerContext) GetNamespaceCollection() tablenamespace.TableNamespaceCollection {
	return hc.namespaceCollection
}

func (hc *HandlerContext) GetDBMSInternalRouter() dbmsinternal.DBMSInternalRouter {
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

func (hc *HandlerContext) initNamespaces() error {
	cfgs, err := dto.GetNamespaceCfg(hc.RuntimeContext.NamespaceCfgRaw)
	if err != nil {
		return err
	}
	namespaces, err := tablenamespace.NewStandardTableNamespaceCollection(cfgs, hc.SQLEngine)
	if err != nil {
		return err
	}
	hc.namespaceCollection = namespaces
	return nil
}

func GetHandlerCtx(cmdString string, runtimeCtx dto.RuntimeCtx, lruCache *lrucache.LRUCache, inputBundle bundle.Bundle) (HandlerContext, error) {

	ac := make(map[string]*dto.AuthCtx)
	err := yaml.Unmarshal([]byte(runtimeCtx.AuthRaw), ac)
	if err != nil {
		return HandlerContext{}, err
	}
	reg, err := getRegistry(runtimeCtx)
	if err != nil {
		return HandlerContext{}, err
	}
	providers := make(map[string]provider.IProvider)
	if err != nil {
		return HandlerContext{}, err
	}
	controlAttributes := inputBundle.GetControlAttributes()
	sqlEngine := inputBundle.GetSQLEngine()
	rv := HandlerContext{
		RawQuery:            cmdString,
		RuntimeContext:      runtimeCtx,
		providers:           providers,
		authContexts:        ac,
		Registry:            reg,
		ControlAttributes:   controlAttributes,
		ErrorPresentation:   runtimeCtx.ErrorPresentation,
		LRUCache:            lruCache,
		SQLEngine:           sqlEngine,
		SQLDialect:          inputBundle.GetSQLDialect(),
		GarbageCollector:    inputBundle.GetGC(),
		TxnCounterMgr:       inputBundle.GetTxnCounterManager(),
		TxnStore:            inputBundle.GetTxnStore(),
		namespaceCollection: inputBundle.GetNamespaceCollection(),
		formatter:           inputBundle.GetSQLDialect().GetASTFormatter(),
		pgInternalRouter:    inputBundle.GetDBMSInternalRouter(),
	}
	drmCfg, err := drm.GetDRMConfig(inputBundle.GetSQLDialect(), rv.namespaceCollection, controlAttributes)
	if err != nil {
		return HandlerContext{}, err
	}
	rv.DrmConfig = drmCfg
	return rv, nil
}
