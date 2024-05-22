package discovery

import (
	"fmt"

	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/stackql/internal/stackql/docparser"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/sql_system"
	"gopkg.in/yaml.v2"

	"github.com/stackql/any-sdk/pkg/nomenclature"

	"github.com/stackql/stackql/pkg/name_mangle"
)

type IDiscoveryStore interface {
	ProcessProviderDiscoveryDoc(string, string) (anysdk.Provider, error)
	processResourcesDiscoveryDoc(
		anysdk.Provider,
		anysdk.ProviderService,
		string) (anysdk.ResourceRegister, error)
	PersistServiceShard(anysdk.Provider, anysdk.ProviderService, string) (anysdk.Service, error)
}

type TTLDiscoveryStore struct {
	sqlSystem  sql_system.SQLSystem
	runtimeCtx dto.RuntimeCtx
	registry   anysdk.RegistryAPI
}

type IDiscoveryAdapter interface {
	GetResourcesMap(prov anysdk.Provider, serviceKey string) (map[string]anysdk.Resource, error)
	GetServiceShard(prov anysdk.Provider, serviceKey, resourceKey string) (anysdk.Service, error)
	GetServiceHandlesMap(prov anysdk.Provider) (map[string]anysdk.ProviderService, error)
	GetServiceHandle(prov anysdk.Provider, serviceKey string) (anysdk.ProviderService, error)
	GetProvider(providerKey string) (anysdk.Provider, error)
	PersistStaticExternalSQLDataSource(prov anysdk.Provider) error
	getDicoveryStore() IDiscoveryStore
}

type BasicDiscoveryAdapter struct {
	alias              string
	apiDiscoveryDocURL string
	discoveryStore     IDiscoveryStore
	runtimeCtx         *dto.RuntimeCtx
	registry           anysdk.RegistryAPI
	sqlSystem          sql_system.SQLSystem
}

func NewBasicDiscoveryAdapter(
	alias string,
	apiDiscoveryDocURL string,
	discoveryStore IDiscoveryStore,
	runtimeCtx *dto.RuntimeCtx,
	registry anysdk.RegistryAPI,
	sqlSystem sql_system.SQLSystem,
) IDiscoveryAdapter {
	return &BasicDiscoveryAdapter{
		alias:              alias,
		apiDiscoveryDocURL: apiDiscoveryDocURL,
		discoveryStore:     discoveryStore,
		runtimeCtx:         runtimeCtx,
		registry:           registry,
		sqlSystem:          sqlSystem,
	}
}

func (adp *BasicDiscoveryAdapter) getDicoveryStore() IDiscoveryStore {
	return adp.discoveryStore
}

//nolint:revive // future proofing
func (adp *BasicDiscoveryAdapter) GetProvider(providerKey string) (anysdk.Provider, error) {
	return adp.discoveryStore.ProcessProviderDiscoveryDoc(adp.apiDiscoveryDocURL, adp.alias)
}

func (adp *BasicDiscoveryAdapter) GetServiceHandlesMap(
	prov anysdk.Provider,
) (map[string]anysdk.ProviderService, error) {
	return prov.GetProviderServices(), nil
}

func (adp *BasicDiscoveryAdapter) GetServiceHandle(
	prov anysdk.Provider,
	serviceKey string,
) (anysdk.ProviderService, error) {
	return prov.GetProviderService(serviceKey)
}

func (adp *BasicDiscoveryAdapter) GetServiceShard(
	prov anysdk.Provider,
	serviceKey,
	resourceKey string,
) (anysdk.Service, error) {
	serviceIDString := docparser.TranslateServiceKeyIqlToGenericProvider(serviceKey)
	sh, err := adp.GetServiceHandle(prov, serviceIDString)
	if err != nil {
		return nil, err
	}
	shard, err := adp.discoveryStore.PersistServiceShard(prov, sh, resourceKey)
	if err != nil {
		return nil, err
	}
	rsc, err := shard.GetResource(resourceKey)
	if err != nil && resourceKey != "" {
		return nil, err
	}
	viewNameMangler := name_mangle.NewViewNameMangler()
	viewCollection, viewCollectionPresent := rsc.GetViewsForSqlDialect(adp.sqlSystem.GetName())
	if viewCollectionPresent {
		for i, view := range viewCollection {
			viewNameNaive := view.GetNameNaive()
			viewName := viewNameMangler.MangleName(viewNameNaive, i)
			_, viewExists := adp.sqlSystem.GetViewByName(viewName)
			if !viewExists {
				// TODO: resolve any possible data race
				err = adp.sqlSystem.CreateView(viewName, view.GetDDL(), true, view.GetRequiredParamNames())
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return shard, nil
}

func (adp *BasicDiscoveryAdapter) PersistStaticExternalSQLDataSource(prov anysdk.Provider) error {
	stackqlConfig, ok := prov.GetStackQLConfig()
	if !ok || len(stackqlConfig.GetExternalTables()) < 1 {
		return fmt.Errorf("no external tables supplied")
	}
	providerName := prov.GetName()
	externalTables := stackqlConfig.GetExternalTables()
	for _, tbl := range externalTables {
		err := adp.sqlSystem.RegisterExternalTable(
			providerName,
			tbl,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (adp *BasicDiscoveryAdapter) GetResourcesMap(
	prov anysdk.Provider,
	serviceKey string,
) (map[string]anysdk.Resource, error) {
	component, err := adp.GetServiceHandle(prov, serviceKey)
	if component == nil || err != nil {
		return nil, err
	}
	if component.GetResourcesRefRef() != "" {
		disDoc, docErr := adp.discoveryStore.processResourcesDiscoveryDoc(
			prov,
			component,
			fmt.Sprintf("%s.%s", adp.alias, serviceKey))
		if docErr != nil {
			return nil, docErr
		}
		return disDoc.GetResources(), nil
	}
	rr, err := adp.registry.GetResourcesShallowFromProviderService(component)
	if err != nil {
		svc, svcErr := adp.registry.GetServiceFromProviderService(component)
		if svcErr != nil {
			return nil, svcErr
		}
		return svc.GetResources()
	}
	if len(rr.GetResources()) == 0 {
		return nil, fmt.Errorf("no resources found for provider = '%s' and service = '%s'", prov.GetName(), serviceKey)
	}
	return rr.GetResources(), nil
}

func NewTTLDiscoveryStore(
	sqlSystem sql_system.SQLSystem,
	registry anysdk.RegistryAPI,
	runtimeCtx dto.RuntimeCtx,
) IDiscoveryStore {
	return &TTLDiscoveryStore{
		sqlSystem:  sqlSystem,
		runtimeCtx: runtimeCtx,
		registry:   registry,
	}
}

//nolint:revive // future proofing
func (store *TTLDiscoveryStore) ProcessProviderDiscoveryDoc(url string, alias string) (anysdk.Provider, error) {
	switch url {
	case "https://www.googleapis.com/discovery/v1/apis":
		ver, err := store.registry.GetLatestAvailableVersion("google")
		if err != nil {
			return nil, fmt.Errorf(
				"locally stored providers not viable. Please try a pull from the registry.  Error: %w", err)
		}
		return store.registry.LoadProviderByName("google", ver)
	default:
		ds, err := nomenclature.ExtractProviderDesignation(url)
		if err != nil {
			return nil, err
		}
		ver, err := store.registry.GetLatestAvailableVersion(ds.Name)
		if err != nil {
			return nil, fmt.Errorf(
				"locally stored providers not viable. Please try a pull from the registry.  Error: %w", err)
		}
		return store.registry.LoadProviderByName(ds.Name, ver)
	}
}

func (store *TTLDiscoveryStore) PersistServiceShard(
	pr anysdk.Provider,
	serviceHandle anysdk.ProviderService,
	resourceKey string,
) (anysdk.Service, error) {
	k := fmt.Sprintf("services.%s.%s", pr.GetName(), serviceHandle.GetName())
	svc, ok := serviceHandle.PeekServiceFragment(resourceKey)
	if ok && svc != nil {
		return svc, nil
	}
	b, err := store.sqlSystem.GetSQLEngine().CacheStoreGet(k)
	if b != nil && err == nil {
		return anysdk.LoadServiceDocFromBytes(serviceHandle, b)
	}
	shard, err := store.registry.GetServiceFragment(serviceHandle, resourceKey)
	if err != nil {
		return nil, err
	}
	serviceHandle.SetServiceRefVal(shard)
	return shard, err
}

//nolint:revive // complexity is fine
func (store *TTLDiscoveryStore) processResourcesDiscoveryDoc(
	prov anysdk.Provider,
	serviceHandle anysdk.ProviderService,
	alias string,
) (anysdk.ResourceRegister, error) {
	providerKey := prov.GetName()
	switch providerKey {
	case "googleapis.com", "google":
		k := fmt.Sprintf("resources.%s.%s", "google", serviceHandle.GetName())
		b, err := store.sqlSystem.GetSQLEngine().CacheStoreGet(k)
		if b != nil && err == nil {
			return anysdk.LoadResourcesShallow(serviceHandle, b)
		}
		rr, err := store.registry.GetResourcesShallowFromProviderService(serviceHandle)
		if err != nil {
			return nil, err
		}
		bt, err := yaml.Marshal(rr)
		if err != nil {
			return nil, err
		}
		err = store.sqlSystem.GetSQLEngine().CacheStorePut(k, bt, "", 0)
		if err != nil {
			return nil, err
		}
		return rr, err
	default:
		k := fmt.Sprintf("%s.%s", providerKey, serviceHandle.GetName())
		b, err := store.sqlSystem.GetSQLEngine().CacheStoreGet(k)
		if b != nil && err == nil {
			return anysdk.LoadResourcesShallow(serviceHandle, b)
		}
		rr, err := store.registry.GetResourcesShallowFromProviderService(serviceHandle)
		if err != nil {
			return nil, err
		}
		bt, err := yaml.Marshal(rr)
		if err != nil {
			return nil, err
		}
		err = store.sqlSystem.GetSQLEngine().CacheStorePut(k, bt, "", 0)
		if err != nil {
			return nil, err
		}
		return rr, nil
	}
}
