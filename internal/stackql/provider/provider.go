package provider

import (
	"net/http"

	"github.com/stackql/any-sdk/pkg/auth_util"
	"github.com/stackql/any-sdk/pkg/constants"
	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/any-sdk/public/formulation"
	"github.com/stackql/stackql/internal/stackql/docparser"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/methodselect"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/sql_system"

	sdk_internal_dto "github.com/stackql/any-sdk/pkg/internaldto"
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

	GetDefaultHTTPClient() *http.Client

	EnhanceMetadataFilter(
		string,
		func(formulation.ITable) (formulation.ITable, error),
		map[string]bool) (func(formulation.ITable) (formulation.ITable, error), error)

	GetCurrentService() string

	GetDefaultKeyForDeleteItems() string

	GetFirstMethodForAction(
		serviceName string,
		resourceName string,
		iqlAction string,
		runtimeCtx dto.RuntimeCtx) (formulation.StandardOperationStore, string, error)

	GetLikeableColumns(string) []string

	GetMethodForAction(
		serviceName string,
		resourceName string,
		iqlAction string,
		parameters parserutil.ColumnKeyedDatastore,
		runtimeCtx dto.RuntimeCtx) (formulation.StandardOperationStore, string, error)

	GetMethodSelector() methodselect.IMethodSelector

	GetProvider() (formulation.Provider, error)

	GetProviderString() string

	GetProviderServicesRedacted(
		runtimeCtx dto.RuntimeCtx,
		extended bool) (map[string]formulation.ProviderService, error)

	GetResource(serviceKey string, resourceKey string, runtimeCtx dto.RuntimeCtx) (formulation.Resource, error)

	GetResourcesMap(
		serviceKey string,
		runtimeCtx dto.RuntimeCtx) (map[string]formulation.Resource, error)

	GetResourcesRedacted(
		currentService string,
		runtimeCtx dto.RuntimeCtx,
		extended bool) (map[string]formulation.Resource, error)

	GetServiceShard(serviceKey string, resourceKey string, runtimeCtx dto.RuntimeCtx) (formulation.Service, error)

	GetObjectSchema(serviceName string, resourceName string, schemaName string) (formulation.Schema, error)

	GetVersion() string

	InferDescribeMethod(formulation.Resource) (formulation.StandardOperationStore, string, error)

	InferMaxResultsElement(formulation.OperationStore) sdk_internal_dto.HTTPElement

	InferNextPageRequestElement(internaldto.Heirarchy) sdk_internal_dto.HTTPElement

	InferNextPageResponseElement(internaldto.Heirarchy) sdk_internal_dto.HTTPElement

	PersistStaticExternalSQLDataSource(dto.RuntimeCtx) error

	SetCurrentService(serviceKey string)

	ShowAuth(authCtx *dto.AuthCtx) (*formulation.AuthMetadata, error)
}

//nolint:revive // TODO: review
func GenerateProvider(
	runtimeCtx dto.RuntimeCtx,
	providerStr,
	providerVersion string,
	reg formulation.RegistryAPI,
	sqlSystem sql_system.SQLSystem,
	persistenceSystem formulation.PersistenceSystem,
	defaultHTTPClient *http.Client,
) (IProvider, error) {
	switch providerStr { //nolint:gocritic // TODO: review
	default:
		return newGenericProvider(
			runtimeCtx, providerStr, providerVersion, reg, sqlSystem, persistenceSystem, defaultHTTPClient)
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

//nolint:revive // TODO: review
func newGenericProvider(
	rtCtx dto.RuntimeCtx,
	providerStr,
	versionStr string,
	reg formulation.RegistryAPI,
	_ sql_system.SQLSystem,
	persistenceSystem formulation.PersistenceSystem,
	defaultHTTPClient *http.Client,
) (IProvider, error) {
	methSel, err := methodselect.NewMethodSelector(providerStr, versionStr)
	if err != nil {
		return nil, err
	}

	rootURL, err := getURL(providerStr)
	if err != nil {
		return nil, err
	}

	da := formulation.NewBasicDiscoveryAdapter(
		providerStr,
		rootURL,
		formulation.NewTTLDiscoveryStore(
			persistenceSystem,
			reg,
			rtCtx,
		),
		&rtCtx,
		reg,
		persistenceSystem,
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
		authUtil:         auth_util.NewAuthUtility(defaultHTTPClient),
		defaultClient:    defaultHTTPClient,
	}
	return gp, err
}
