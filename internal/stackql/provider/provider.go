package provider

import (
	"net/http"

	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/any-sdk/pkg/auth_util"
	"github.com/stackql/any-sdk/pkg/constants"
	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/any-sdk/public/discovery"
	sdk_persistence "github.com/stackql/any-sdk/public/persistence"
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

	EnhanceMetadataFilter(
		string,
		func(anysdk.ITable) (anysdk.ITable, error),
		map[string]bool) (func(anysdk.ITable) (anysdk.ITable, error), error)

	GetCurrentService() string

	GetDefaultKeyForDeleteItems() string

	GetFirstMethodForAction(
		serviceName string,
		resourceName string,
		iqlAction string,
		runtimeCtx dto.RuntimeCtx) (anysdk.StandardOperationStore, string, error)

	GetLikeableColumns(string) []string

	GetMethodForAction(
		serviceName string,
		resourceName string,
		iqlAction string,
		parameters parserutil.ColumnKeyedDatastore,
		runtimeCtx dto.RuntimeCtx) (anysdk.StandardOperationStore, string, error)

	GetMethodSelector() methodselect.IMethodSelector

	GetProvider() (anysdk.Provider, error)

	GetProviderString() string

	GetProviderServicesRedacted(
		runtimeCtx dto.RuntimeCtx,
		extended bool) (map[string]anysdk.ProviderService, error)

	GetResource(serviceKey string, resourceKey string, runtimeCtx dto.RuntimeCtx) (anysdk.Resource, error)

	GetResourcesMap(
		serviceKey string,
		runtimeCtx dto.RuntimeCtx) (map[string]anysdk.Resource, error)

	GetResourcesRedacted(
		currentService string,
		runtimeCtx dto.RuntimeCtx,
		extended bool) (map[string]anysdk.Resource, error)

	GetServiceShard(serviceKey string, resourceKey string, runtimeCtx dto.RuntimeCtx) (anysdk.Service, error)

	GetObjectSchema(serviceName string, resourceName string, schemaName string) (anysdk.Schema, error)

	GetVersion() string

	InferDescribeMethod(anysdk.Resource) (anysdk.StandardOperationStore, string, error)

	InferMaxResultsElement(anysdk.OperationStore) sdk_internal_dto.HTTPElement

	InferNextPageRequestElement(internaldto.Heirarchy) sdk_internal_dto.HTTPElement

	InferNextPageResponseElement(internaldto.Heirarchy) sdk_internal_dto.HTTPElement

	PersistStaticExternalSQLDataSource(dto.RuntimeCtx) error

	SetCurrentService(serviceKey string)

	ShowAuth(authCtx *dto.AuthCtx) (*anysdk.AuthMetadata, error)
}

func GenerateProvider(
	runtimeCtx dto.RuntimeCtx,
	providerStr,
	providerVersion string,
	reg anysdk.RegistryAPI,
	sqlSystem sql_system.SQLSystem,
	persistenceSystem sdk_persistence.PersistenceSystem,
) (IProvider, error) {
	switch providerStr { //nolint:gocritic // TODO: review
	default:
		return newGenericProvider(runtimeCtx, providerStr, providerVersion, reg, sqlSystem, persistenceSystem)
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
	reg anysdk.RegistryAPI,
	_ sql_system.SQLSystem,
	persistenceSystem sdk_persistence.PersistenceSystem,
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
		authUtil:         auth_util.NewAuthUtility(),
	}
	return gp, err
}
