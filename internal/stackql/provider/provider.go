package provider

import (
	"fmt"

	"net/http"

	"github.com/stackql/stackql/internal/stackql/config"
	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/discovery"
	"github.com/stackql/stackql/internal/stackql/docparser"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/httpexec"
	"github.com/stackql/stackql/internal/stackql/methodselect"
	"github.com/stackql/stackql/internal/stackql/sqlengine"

	"github.com/stackql/go-openapistackql/openapistackql"
)

const (
	ambiguousServiceErrorMessage string = "More than one service exists with this name, please use the id in the object name, or unset the --usenonpreferredapis flag"
	googleProviderName           string = "google"
	oktaProviderName             string = "okta"
	SchemaDelimiter              string = docparser.SchemaDelimiter
)

var DummyAuth bool = false

type ProviderParam struct {
	Id     string
	Type   string
	Format string
}

func GetSupportedProviders(extended bool) map[string]map[string]interface{} {
	retVal := make(map[string]map[string]interface{})
	if extended {
		retVal[googleProviderName] = getProviderMapExtended(googleProviderName)
		retVal[oktaProviderName] = getProviderMapExtended(oktaProviderName)
	} else {
		retVal[googleProviderName] = getProviderMap(googleProviderName)
		retVal[oktaProviderName] = getProviderMap(oktaProviderName)
	}
	return retVal
}

type IProvider interface {
	Auth(authCtx *dto.AuthCtx, authTypeRequested string, enforceRevokeFirst bool) (*http.Client, error)

	AuthRevoke(authCtx *dto.AuthCtx) error

	CheckCredentialFile(authCtx *dto.AuthCtx) error

	EnhanceMetadataFilter(string, func(openapistackql.ITable) (openapistackql.ITable, error), map[string]bool) (func(openapistackql.ITable) (openapistackql.ITable, error), error)

	GenerateHTTPRestInstruction(httpContext httpexec.IHttpContext) (httpexec.IHttpContext, error)

	GetCurrentService() string

	GetDefaultKeyForDeleteItems() string

	GetLikeableColumns(string) []string

	GetMethodForAction(serviceName string, resourceName string, iqlAction string, runtimeCtx dto.RuntimeCtx) (*openapistackql.OperationStore, string, error)

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

	InferNextPageRequestElement(*openapistackql.OperationStore) *dto.HTTPElement

	InferNextPageResponseElement(*openapistackql.OperationStore) *dto.HTTPElement

	Parameterise(httpContext httpexec.IHttpContext, method *openapistackql.OperationStore, parameters *dto.HttpParameters, requestSchema *openapistackql.Schema) (httpexec.IHttpContext, error)

	SetCurrentService(serviceKey string)

	ShowAuth(authCtx *dto.AuthCtx) (*openapistackql.AuthMetadata, error)

	GetDiscoveryGeneration(sqlengine.SQLEngine) (int, error)
}

func GetProviderFromRuntimeCtx(runtimeCtx dto.RuntimeCtx, dbEngine sqlengine.SQLEngine) (IProvider, error) {
	providerStr := runtimeCtx.ProviderStr
	return GetProvider(runtimeCtx, providerStr, "v1", dbEngine)
}

func GetProvider(runtimeCtx dto.RuntimeCtx, providerStr, providerVersion string, dbEngine sqlengine.SQLEngine) (IProvider, error) {
	switch providerStr {
	case config.GetGoogleProviderString(), config.GetOktaProviderString():
		return newGenericProvider(runtimeCtx, providerStr, providerVersion, dbEngine)
	}
	return nil, fmt.Errorf("provider %s not supported", providerStr)
}

func getUrl(prov string) (string, error) {
	switch prov {
	case "google":
		return constants.GoogleV1DiscoveryDoc, nil
	case "okta":
		return "okta", nil
	}
	return "", fmt.Errorf("cannot find root doc for provider = '%s'", prov)
}

func newGenericProvider(rtCtx dto.RuntimeCtx, providerStr, versionStr string, dbEngine sqlengine.SQLEngine) (IProvider, error) {
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
			rtCtx,
		),
		&rtCtx,
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
