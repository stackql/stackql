package provider

import (
	"net/http"

	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/discovery"
	"github.com/stackql/stackql/internal/stackql/docparser"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/methodselect"
	"github.com/stackql/stackql/internal/stackql/sqlengine"

	"github.com/stackql/go-openapistackql/openapistackql"
)

const (
	ambiguousServiceErrorMessage string = "More than one service exists with this name, please use the id in the object name, or unset the --usenonpreferredapis flag"
	SchemaDelimiter              string = docparser.SchemaDelimiter
)

var DummyAuth bool = false

type ProviderParam struct {
	Id     string
	Type   string
	Format string
}

type IProvider interface {
	Auth(authCtx *dto.AuthCtx, authTypeRequested string, enforceRevokeFirst bool) (*http.Client, error)

	AuthRevoke(authCtx *dto.AuthCtx) error

	CheckCredentialFile(authCtx *dto.AuthCtx) error

	EnhanceMetadataFilter(string, func(openapistackql.ITable) (openapistackql.ITable, error), map[string]bool) (func(openapistackql.ITable) (openapistackql.ITable, error), error)

	GetCurrentService() string

	GetDefaultKeyForDeleteItems() string

	GetLikeableColumns(string) []string

	GetMethodForAction(serviceName string, resourceName string, iqlAction string, parameters map[string]interface{}, runtimeCtx dto.RuntimeCtx) (*openapistackql.OperationStore, string, map[string]interface{}, error)

	GetMethodSelector() methodselect.IMethodSelector

	GetProvider() (*openapistackql.Provider, error)

	GetProviderString() string

	GetProviderServicesRedacted(runtimeCtx dto.RuntimeCtx, extended bool) (map[string]*openapistackql.ProviderService, error)

	GetResource(serviceKey string, resourceKey string, runtimeCtx dto.RuntimeCtx) (*openapistackql.Resource, error)

	GetResourcesMap(serviceKey string, runtimeCtx dto.RuntimeCtx) (map[string]*openapistackql.Resource, error)

	GetResourcesRedacted(currentService string, runtimeCtx dto.RuntimeCtx, extended bool) (map[string]*openapistackql.Resource, error)

	GetServiceShard(serviceKey string, resourceKey string, runtimeCtx dto.RuntimeCtx) (*openapistackql.Service, error)

	GetObjectSchema(serviceName string, resourceName string, schemaName string) (*openapistackql.Schema, error)

	GetVersion() string

	InferDescribeMethod(*openapistackql.Resource) (*openapistackql.OperationStore, string, error)

	InferMaxResultsElement(*openapistackql.OperationStore) *dto.HTTPElement

	InferNextPageRequestElement(dto.Heirarchy) *dto.HTTPElement

	InferNextPageResponseElement(dto.Heirarchy) *dto.HTTPElement

	SetCurrentService(serviceKey string)

	ShowAuth(authCtx *dto.AuthCtx) (*openapistackql.AuthMetadata, error)

	GetDiscoveryGeneration(sqlengine.SQLEngine) (int, error)
}

func GetProvider(runtimeCtx dto.RuntimeCtx, providerStr, providerVersion string, reg openapistackql.RegistryAPI, dbEngine sqlengine.SQLEngine) (IProvider, error) {
	switch providerStr {
	default:
		return newGenericProvider(runtimeCtx, providerStr, providerVersion, reg, dbEngine)
	}
}

func getUrl(prov string) (string, error) {
	switch prov {
	case "google":
		return constants.GoogleV1DiscoveryDoc, nil
	default:
		return prov, nil
	}
}

func newGenericProvider(rtCtx dto.RuntimeCtx, providerStr, versionStr string, reg openapistackql.RegistryAPI, dbEngine sqlengine.SQLEngine) (IProvider, error) {
	methSel, err := methodselect.NewMethodSelector(providerStr, versionStr)
	if err != nil {
		return nil, err
	}

	rootUrl, err := getUrl(providerStr)
	if err != nil {
		return nil, err
	}

	da := discovery.NewBasicDiscoveryAdapter(
		providerStr,
		rootUrl,
		discovery.NewTTLDiscoveryStore(
			dbEngine,
			reg,
			rtCtx,
		),
		&rtCtx,
		reg,
	)

	p, err := da.GetProvider(rtCtx.ProviderStr)

	if err != nil {
		return nil, err
	}

	gp := &GenericProvider{
		provider:         p,
		runtimeCtx:       rtCtx,
		discoveryAdapter: da,
		apiVersion:       versionStr,
		methodSelector:   methSel,
	}
	return gp, err
}
