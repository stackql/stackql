package discovery

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/docparser"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"gopkg.in/yaml.v2"

	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/go-openapistackql/pkg/nomenclature"
)

type IDiscoveryStore interface {
	ProcessProviderDiscoveryDoc(string, string) (*openapistackql.Provider, error)
	ProcessResourcesDiscoveryDoc(string, *openapistackql.ProviderService, string) (*openapistackql.ResourceRegister, error)
	PersistServiceShard(*openapistackql.Provider, *openapistackql.ProviderService, string) (*openapistackql.Service, error)
}

type TTLDiscoveryStore struct {
	sqlengine  sqlengine.SQLEngine
	runtimeCtx dto.RuntimeCtx
	registry   openapistackql.RegistryAPI
}

type IDiscoveryAdapter interface {
	GetResourcesMap(prov *openapistackql.Provider, serviceKey string) (map[string]*openapistackql.Resource, error)
	GetServiceShard(prov *openapistackql.Provider, serviceKey, resourceKey string) (*openapistackql.Service, error)
	GetServiceHandlesMap(prov *openapistackql.Provider) (map[string]*openapistackql.ProviderService, error)
	GetServiceHandle(prov *openapistackql.Provider, serviceKey string) (*openapistackql.ProviderService, error)
	GetProvider(providerKey string) (*openapistackql.Provider, error)
}

type BasicDiscoveryAdapter struct {
	alias              string
	apiDiscoveryDocUrl string
	discoveryStore     IDiscoveryStore
	runtimeCtx         *dto.RuntimeCtx
	registry           openapistackql.RegistryAPI
}

func NewBasicDiscoveryAdapter(
	alias string,
	apiDiscoveryDocUrl string,
	discoveryStore IDiscoveryStore,
	runtimeCtx *dto.RuntimeCtx,
	registry openapistackql.RegistryAPI,
) IDiscoveryAdapter {
	return &BasicDiscoveryAdapter{
		alias:              alias,
		apiDiscoveryDocUrl: apiDiscoveryDocUrl,
		discoveryStore:     discoveryStore,
		runtimeCtx:         runtimeCtx,
		registry:           registry,
	}
}

func (adp *BasicDiscoveryAdapter) GetProvider(providerKey string) (*openapistackql.Provider, error) {
	return adp.discoveryStore.ProcessProviderDiscoveryDoc(adp.apiDiscoveryDocUrl, adp.alias)
}

func (adp *BasicDiscoveryAdapter) GetServiceHandlesMap(prov *openapistackql.Provider) (map[string]*openapistackql.ProviderService, error) {
	return prov.ProviderServices, nil
}

func (adp *BasicDiscoveryAdapter) GetServiceHandle(prov *openapistackql.Provider, serviceKey string) (*openapistackql.ProviderService, error) {
	ps := prov.ProviderServices
	rv, ok := ps[serviceKey]
	if !ok {
		return nil, fmt.Errorf("could not find providerService = '%s'", serviceKey)
	}
	return rv, nil
}

func (adp *BasicDiscoveryAdapter) GetServiceShard(prov *openapistackql.Provider, serviceKey, resourceKey string) (*openapistackql.Service, error) {
	serviceIdString := docparser.TranslateServiceKeyIqlToGenericProvider(serviceKey)
	sh, err := adp.GetServiceHandle(prov, serviceIdString)
	if err != nil {
		return nil, err
	}
	shard, err := adp.discoveryStore.PersistServiceShard(prov, sh, resourceKey)
	if err != nil {
		return nil, err
	}
	return shard, nil
}

func (adp *BasicDiscoveryAdapter) GetResourcesMap(prov *openapistackql.Provider, serviceKey string) (map[string]*openapistackql.Resource, error) {
	component, err := adp.GetServiceHandle(prov, serviceKey)
	if component == nil || err != nil {
		return nil, err
	}
	if component.ResourcesRef != nil && component.ResourcesRef.Ref != "" {
		disDoc, err := adp.discoveryStore.ProcessResourcesDiscoveryDoc(prov.Name, component, fmt.Sprintf("%s.%s", adp.alias, serviceKey))
		if err != nil {
			return nil, err
		}
		return disDoc.Resources, nil
	}
	rr, err := adp.registry.GetResourcesShallowFromProviderService(component)
	if err != nil {
		svc, err := adp.registry.GetServiceFromProviderService(component)
		if err != nil {
			return nil, err
		}
		return svc.GetResources()
	} else {
		if rr.Resources == nil {
			return nil, fmt.Errorf("no resources found for provider = '%s' and service = '%s'", prov.Name, serviceKey)
		}
		return rr.Resources, nil
	}

}

func NewTTLDiscoveryStore(sqlengine sqlengine.SQLEngine, registry openapistackql.RegistryAPI, runtimeCtx dto.RuntimeCtx) IDiscoveryStore {
	return &TTLDiscoveryStore{
		sqlengine:  sqlengine,
		runtimeCtx: runtimeCtx,
		registry:   registry,
	}
}

func (store *TTLDiscoveryStore) ProcessProviderDiscoveryDoc(url string, alias string) (*openapistackql.Provider, error) {
	switch url {
	case "https://www.googleapis.com/discovery/v1/apis":
		ver, err := store.registry.GetLatestAvailableVersion("google")
		if err != nil {
			return nil, fmt.Errorf("locally stored providers not viable. Please try a pull from the registry.  Error: %s", err.Error())
		}
		return store.registry.LoadProviderByName("google", ver)
	default:
		ds, err := nomenclature.ExtractProviderDesignation(url)
		if err != nil {
			return nil, err
		}
		ver, err := store.registry.GetLatestAvailableVersion(ds.Name)
		if err != nil {
			return nil, fmt.Errorf("locally stored providers not viable. Please try a pull from the registry.  Error: %s", err.Error())
		}
		return store.registry.LoadProviderByName(ds.Name, ver)
	}
}

func (store *TTLDiscoveryStore) PersistServiceShard(pr *openapistackql.Provider, serviceHandle *openapistackql.ProviderService, resourceKey string) (*openapistackql.Service, error) {
	k := fmt.Sprintf("services.%s.%s", pr.Name, serviceHandle.Name)
	svc, ok := serviceHandle.PeekServiceFragment(resourceKey)
	if ok && svc != nil {
		return svc, nil
	}
	b, err := store.sqlengine.CacheStoreGet(k)
	if b != nil && err == nil {
		return openapistackql.LoadServiceDocFromBytes(b)
	}
	shard, err := store.registry.GetServiceFragment(serviceHandle, resourceKey)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	serviceHandle.ServiceRef.Value = shard
	return shard, err
}

func (store *TTLDiscoveryStore) ProcessResourcesDiscoveryDoc(providerKey string, serviceHandle *openapistackql.ProviderService, alias string) (*openapistackql.ResourceRegister, error) {
	switch providerKey {
	case "googleapis.com", "google":
		k := fmt.Sprintf("resources.%s.%s", "google", serviceHandle.Name)
		b, err := store.sqlengine.CacheStoreGet(k)
		if b != nil && err == nil {
			return openapistackql.LoadResourcesShallow(b)
		}
		rr, err := store.registry.GetResourcesShallowFromProviderService(serviceHandle)
		if err != nil {
			return nil, err
		}
		bt, err := yaml.Marshal(rr)
		if err != nil {
			return nil, err
		}
		err = store.sqlengine.CacheStorePut(k, bt, "", 0)
		if err != nil {
			return nil, err
		}
		return rr, err
	default:
		k := fmt.Sprintf("%s.%s", providerKey, serviceHandle.Name)
		b, err := store.sqlengine.CacheStoreGet(k)
		if b != nil && err == nil {
			return openapistackql.LoadResourcesShallow(b)
		}
		rr, err := store.registry.GetResourcesShallowFromProviderService(serviceHandle)
		if err != nil {
			return nil, err
		}
		bt, err := yaml.Marshal(rr)
		if err != nil {
			return nil, err
		}
		err = store.sqlengine.CacheStorePut(k, bt, "", 0)
		if err != nil {
			return nil, err
		}
		return rr, nil
	}
}
