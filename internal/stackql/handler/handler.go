package handler

import (
	"fmt"
	"io"

	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/go-openapistackql/pkg/nomenclature"
	"github.com/stackql/stackql/internal/pkg/txncounter"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/netutils"
	"github.com/stackql/stackql/internal/stackql/provider"
	"github.com/stackql/stackql/internal/stackql/sqlengine"

	"gopkg.in/yaml.v2"
	lrucache "vitess.io/vitess/go/cache"
)

var (
	drmConfig drm.DRMConfig = drm.GetGoogleV1SQLiteConfig()
)

type HandlerContext struct {
	RawQuery          string
	Query             string
	RuntimeContext    dto.RuntimeCtx
	providers         map[string]provider.IProvider
	CurrentProvider   string
	authContexts      map[string]*dto.AuthCtx
	Registry          openapistackql.RegistryAPI
	ErrorPresentation string
	Outfile           io.Writer
	OutErrFile        io.Writer
	LRUCache          *lrucache.LRUCache
	SQLEngine         sqlengine.SQLEngine
	DrmConfig         drm.DRMConfig
	TxnCounterMgr     *txncounter.TxnCounterManager
}

func getProviderMap(providerName string) map[string]interface{} {
	googleMap := map[string]interface{}{
		"name": providerName,
	}
	return googleMap
}

func getProviderMapExtended(providerName string) map[string]interface{} {
	return getProviderMap(providerName)
}

func (hc *HandlerContext) GetSupportedProviders(extended bool) map[string]map[string]interface{} {
	retVal := make(map[string]map[string]interface{})
	for pn := range hc.Registry.ListLocallyAvailableProviders() {
		if pn == "googleapis.com" {
			pn = "google"
		}
		if extended {
			retVal[pn] = getProviderMapExtended(pn)
		} else {
			retVal[pn] = getProviderMap(pn)
		}
	}
	return retVal
}

func (hc *HandlerContext) GetProvider(providerName string) (provider.IProvider, error) {
	var err error
	if providerName == "" {
		providerName = hc.RuntimeContext.ProviderStr
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
		rc.LocalDocRoot = runtimeCtx.ApplicationFilesRootPath
	}
	rt := netutils.GetRoundTripper(runtimeCtx, nil)
	return openapistackql.NewRegistry(rc, rt)
}

func GetHandlerCtx(cmdString string, runtimeCtx dto.RuntimeCtx, lruCache *lrucache.LRUCache, sqlEng sqlengine.SQLEngine) (HandlerContext, error) {

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
	return HandlerContext{
		RawQuery:          cmdString,
		RuntimeContext:    runtimeCtx,
		providers:         providers,
		authContexts:      ac,
		Registry:          reg,
		ErrorPresentation: runtimeCtx.ErrorPresentation,
		LRUCache:          lruCache,
		SQLEngine:         sqlEng,
		DrmConfig:         drmConfig,
		TxnCounterMgr:     nil,
	}, nil
}
