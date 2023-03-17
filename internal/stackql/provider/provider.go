package provider

import (
	"net/http"

	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/discovery"
	"github.com/stackql/stackql/internal/stackql/docparser"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/methodselect"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/sql_system"

	"github.com/stackql/go-openapistackql/openapistackql"
)

const (
	ambiguousServiceErrorMessage string = "More than one service exists with this name, please use the id in the object name, or unset the --usenonpreferredapis flag" //nolint:lll // long string
	SchemaDelimiter              string = docparser.SchemaDelimiter
)

var (
	DummyAuth bool = false //nolint:revive,gochecknoglobals // prefer declarative
)

type ProviderParam struct { //nolint:revive // TODO: review
	Id     string //nolint:revive,stylecheck // TODO: review
	Type   string
	Format string
}

type IProvider interface {
	Auth(authCtx *dto.AuthCtx, authTypeRequested string, enforceRevokeFirst bool) (*http.Client, error)

	AuthRevoke(authCtx *dto.AuthCtx) error

	CheckCredentialFile(authCtx *dto.AuthCtx) error

	EnhanceMetadataFilter(
		string,
		func(openapistackql.ITable) (openapistackql.ITable, error),
		map[string]bool) (func(openapistackql.ITable) (openapistackql.ITable, error), error)

	GetCurrentService() string

	GetDefaultKeyForDeleteItems() string

	GetFirstMethodForAction(
		serviceName string,
		resourceName string,
		iqlAction string,
		runtimeCtx dto.RuntimeCtx) (openapistackql.OperationStore, string, error)

	GetLikeableColumns(string) []string

	GetMethodForAction(
		serviceName string,
		resourceName string,
		iqlAction string,
		parameters parserutil.ColumnKeyedDatastore,
		runtimeCtx dto.RuntimeCtx) (openapistackql.OperationStore, string, error)

	GetMethodSelector() methodselect.IMethodSelector

	GetProvider() (openapistackql.Provider, error)

	GetProviderString() string

	GetProviderServicesRedacted(
		runtimeCtx dto.RuntimeCtx,
		extended bool) (map[string]openapistackql.ProviderService, error)

	GetResource(serviceKey string, resourceKey string, runtimeCtx dto.RuntimeCtx) (openapistackql.Resource, error)

	GetResourcesMap(
		serviceKey string,
		runtimeCtx dto.RuntimeCtx) (map[string]openapistackql.Resource, error)

	GetResourcesRedacted(
		currentService string,
		runtimeCtx dto.RuntimeCtx,
		extended bool) (map[string]openapistackql.Resource, error)

	GetServiceShard(serviceKey string, resourceKey string, runtimeCtx dto.RuntimeCtx) (openapistackql.Service, error)

	GetObjectSchema(serviceName string, resourceName string, schemaName string) (openapistackql.Schema, error)

	GetVersion() string

	InferDescribeMethod(openapistackql.Resource) (openapistackql.OperationStore, string, error)

	InferMaxResultsElement(openapistackql.OperationStore) internaldto.HTTPElement

	InferNextPageRequestElement(internaldto.Heirarchy) internaldto.HTTPElement

	InferNextPageResponseElement(internaldto.Heirarchy) internaldto.HTTPElement

	PersistStaticExternalSQLDataSource(dto.RuntimeCtx) error

	SetCurrentService(serviceKey string)

	ShowAuth(authCtx *dto.AuthCtx) (*openapistackql.AuthMetadata, error)
}

func GetProvider(
	runtimeCtx dto.RuntimeCtx,
	providerStr,
	providerVersion string,
	reg openapistackql.RegistryAPI,
	sqlSystem sql_system.SQLSystem,
) (IProvider, error) {
	switch providerStr { //nolint:gocritic // TODO: review
	default:
		return newGenericProvider(runtimeCtx, providerStr, providerVersion, reg, sqlSystem)
	}
}

func getURL(prov string) (string, error) { //nolint:unparam // TODO: review
	switch prov {
	case "google":
		return constants.GoogleV1DiscoveryDoc, nil
	default:
		return prov, nil
	}
}

func newGenericProvider(
	rtCtx dto.RuntimeCtx,
	providerStr,
	versionStr string,
	reg openapistackql.RegistryAPI,
	sqlSystem sql_system.SQLSystem,
) (IProvider, error) {
	methSel, err := methodselect.NewMethodSelector(providerStr, versionStr)
	if err != nil {
		return nil, err
	}

	rootURL, err := getURL(providerStr)
	if err != nil {
		return nil, err
	}

	da := discovery.NewBasicDiscoveryAdapter(
		providerStr,
		rootURL,
		discovery.NewTTLDiscoveryStore(
			sqlSystem,
			reg,
			rtCtx,
		),
		&rtCtx,
		reg,
		sqlSystem,
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
