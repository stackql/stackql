package discovery

import (
	"fmt"
	"io"

	"net/http"

	"github.com/stackql/stackql/internal/stackql/docparser"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/netutils"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"gopkg.in/yaml.v2"

	"github.com/stackql/go-openapistackql/openapistackql"
)

type IDiscoveryStore interface {
	ProcessProviderDiscoveryDoc(string, string) (*openapistackql.Provider, error)
	ProcessResourcesDiscoveryDoc(string, *openapistackql.ProviderService, string) (*openapistackql.ResourceRegister, error)
	ProcessServiceDiscoveryDoc(string, *openapistackql.ProviderService, string) (*openapistackql.Service, error)
	PersistServiceShard(*openapistackql.Provider, *openapistackql.ProviderService, string) (*openapistackql.Service, error)
}

type TTLDiscoveryStore struct {
	sqlengine  sqlengine.SQLEngine
	runtimeCtx dto.RuntimeCtx
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
}

func NewBasicDiscoveryAdapter(
	alias string,
	apiDiscoveryDocUrl string,
	discoveryStore IDiscoveryStore,
	runtimeCtx *dto.RuntimeCtx,
) IDiscoveryAdapter {
	return &BasicDiscoveryAdapter{
		alias:              alias,
		apiDiscoveryDocUrl: apiDiscoveryDocUrl,
		discoveryStore:     discoveryStore,
		runtimeCtx:         runtimeCtx,
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
	rr, err := component.GetResourcesShallow()
	if err != nil {
		svc, err := component.GetService()
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

func NewTTLDiscoveryStore(sqlengine sqlengine.SQLEngine, runtimeCtx dto.RuntimeCtx) IDiscoveryStore {
	return &TTLDiscoveryStore{
		sqlengine:  sqlengine,
		runtimeCtx: runtimeCtx,
	}
}

func DownloadDiscoveryDoc(url string, runtimeCtx dto.RuntimeCtx) (io.ReadCloser, error) {
	httpClient := netutils.GetHttpClient(runtimeCtx, nil)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, getErr := httpClient.Do(req)
	if getErr != nil {
		return nil, err
	}
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("discovery doc download for '%s' failed with code = %d", url, res.StatusCode)
	}
	return res.Body, nil
}

func (store *TTLDiscoveryStore) ProcessProviderDiscoveryDoc(url string, alias string) (*openapistackql.Provider, error) {
	switch url {
	case "https://www.googleapis.com/discovery/v1/apis":
		return openapistackql.LoadProviderByName("google")
	case "okta":
		return openapistackql.LoadProviderByName("okta")
	}
	return nil, fmt.Errorf("cannot process provider discovery doc url = '%s'", url)
}

func (store *TTLDiscoveryStore) ProcessServiceDiscoveryDoc(providerKey string, serviceHandle *openapistackql.ProviderService, alias string) (*openapistackql.Service, error) {
	// k := fmt.Sprintf("%s.%s", providerKey, serviceHandle.Name)
	switch providerKey {
	case "googleapis.com", "google":
		k := fmt.Sprintf("services.%s.%s", "google", serviceHandle.Name)
		b, err := store.sqlengine.CacheStoreGet(k)
		if b != nil && err == nil {
			return openapistackql.LoadServiceDocFromBytes(b)
		}
		// pr, err := openapistackql.LoadProviderByName("google")
		// if err != nil {
		// 	return nil, err
		// }
		svc, err := serviceHandle.GetServiceFragment(serviceHandle.Name)
		if err != nil {
			svc, err = serviceHandle.GetServiceFragment(serviceHandle.ID)
		}
		if err != nil {
			return nil, err
		}
		bt, err := svc.ToYaml()
		if err != nil {
			return nil, err
		}
		err = store.sqlengine.CacheStorePut(k, bt, "", 0)
		if err != nil {
			return nil, err
		}
		// err = docparser.OpenapiStackQLServiceDiscoveryDocPersistor(pr, svc, store.sqlengine, pr.Name)
		return svc, err
	default:
		k := fmt.Sprintf("%s.%s", providerKey, serviceHandle.Name)
		b, err := store.sqlengine.CacheStoreGet(k)
		if b != nil && err == nil {
			return openapistackql.LoadServiceDocFromBytes(b)
		}
		pr, err := openapistackql.LoadProviderByName(providerKey)
		if err != nil {
			return nil, err
		}
		svc, err := pr.GetService(serviceHandle.Name)
		if err != nil {
			svc, err = pr.GetService(serviceHandle.ID)
		}
		if err != nil {
			return nil, err
		}
		bt, err := svc.ToYaml()
		if err != nil {
			return nil, err
		}
		err = store.sqlengine.CacheStorePut(k, bt, "", 0)
		if err != nil {
			return nil, err
		}
		// err = docparser.OpenapiStackQLServiceDiscoveryDocPersistor(pr, svc, store.sqlengine, pr.Name)
		return svc, err
	}
}

func (store *TTLDiscoveryStore) PersistServiceShard(pr *openapistackql.Provider, serviceHandle *openapistackql.ProviderService, resourceKey string) (*openapistackql.Service, error) {
	k := fmt.Sprintf("services.%s.%s", "google", serviceHandle.Name)
	svc, ok := serviceHandle.PeekServiceFragment(resourceKey)
	if ok && svc != nil {
		return svc, nil
	}
	b, err := store.sqlengine.CacheStoreGet(k)
	if b != nil && err == nil {
		return openapistackql.LoadServiceDocFromBytes(b)
	}
	shard, err := serviceHandle.GetServiceFragment(resourceKey)
	if err != nil {
		return nil, err
	}
	// err = docparser.OpenapiStackQLServiceDiscoveryDocPersistor(pr, shard, store.sqlengine, pr.Name)
	if err != nil {
		return nil, err
	}
	serviceHandle.ServiceRef.Value = shard
	return shard, err
}

func (store *TTLDiscoveryStore) ProcessResourcesDiscoveryDoc(providerKey string, serviceHandle *openapistackql.ProviderService, alias string) (*openapistackql.ResourceRegister, error) {
	// k := fmt.Sprintf("%s.%s", providerKey, serviceHandle.Name)
	switch providerKey {
	case "googleapis.com", "google":
		k := fmt.Sprintf("resources.%s.%s", "google", serviceHandle.Name)
		b, err := store.sqlengine.CacheStoreGet(k)
		if b != nil && err == nil {
			return openapistackql.LoadResourcesShallow(b)
		}
		rr, err := serviceHandle.GetResourcesShallow()
		if err != nil {
			rr, err = serviceHandle.GetResourcesShallow()
		}
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
		// err = docparser.OpenapiStackQLServiceDiscoveryDocPersistor(pr, svc, store.sqlengine, pr.Name)
		return rr, err
	default:
		k := fmt.Sprintf("%s.%s", providerKey, serviceHandle.Name)
		b, err := store.sqlengine.CacheStoreGet(k)
		if b != nil && err == nil {
			return openapistackql.LoadResourcesShallow(b)
		}
		rr, err := serviceHandle.GetResourcesShallow()
		if err != nil {
			rr, err = serviceHandle.GetResourcesShallow()
		}
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

// func (store *TTLDiscoveryStore) GetService(providerKey string, serviceHandle *openapistackql.ProviderService, alias string) (*openapistackql.Service, error) {

// }
