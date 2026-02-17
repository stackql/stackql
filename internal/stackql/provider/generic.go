package provider

import (
	"errors"
	"fmt"

	"github.com/stackql/any-sdk/pkg/auth_util"
	"github.com/stackql/any-sdk/pkg/constants"
	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/any-sdk/pkg/logging"
	"github.com/stackql/any-sdk/pkg/netutils"
	"github.com/stackql/any-sdk/public/formulation"
	"github.com/stackql/stackql/internal/stackql/docparser"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/methodselect"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/relational"

	"github.com/stackql/stackql/pkg/sqltypeutil"

	sdk_internal_dto "github.com/stackql/any-sdk/pkg/internaldto"

	"net/http"
	"regexp"
	"strings"
)

var (
	//nolint:unused // prefer declarative
	gitHubLinksNextRegex *regexp.Regexp = regexp.MustCompile(`.*<(?P<nextURL>[^>]*)>;\ rel="next".*`)
)

type GenericProvider struct {
	provider         formulation.Provider
	runtimeCtx       dto.RuntimeCtx
	currentService   string
	discoveryAdapter formulation.IDiscoveryAdapter
	apiVersion       string
	methodSelector   methodselect.IMethodSelector
	authUtil         auth_util.AuthUtility
	defaultClient    *http.Client // for testing purposes only, defaulted downstream
}

func (gp *GenericProvider) GetDefaultKeyForDeleteItems() string {
	if gp.provider.GetDeleteItemsKey() != "" {
		return gp.provider.GetDeleteItemsKey()
	}
	return "items"
}

func (gp *GenericProvider) GetMethodSelector() methodselect.IMethodSelector {
	return gp.methodSelector
}

func (gp *GenericProvider) GetVersion() string {
	return gp.apiVersion
}

func (gp *GenericProvider) GetServiceShard(
	serviceKey string,
	resourceKey string,
	runtimeCtx dto.RuntimeCtx, //nolint:revive // future proofing
) (formulation.Service, error) {
	return gp.discoveryAdapter.GetServiceShard(gp.provider, serviceKey, resourceKey)
}

//nolint:revive // future proofing
func (gp *GenericProvider) PersistStaticExternalSQLDataSource(runtimeCtx dto.RuntimeCtx) error {
	return gp.discoveryAdapter.PersistStaticExternalSQLDataSource(gp.provider)
}

func (gp *GenericProvider) inferAuthType(authCtx dto.AuthCtx, authTypeRequested string) string {
	ft := strings.ToLower(authTypeRequested)
	switch ft {
	case dto.AuthAzureDefaultStr:
		return dto.AuthAzureDefaultStr
	case dto.AuthAPIKeyStr:
		return dto.AuthAPIKeyStr
	case dto.AuthBasicStr:
		return dto.AuthBasicStr
	case dto.AuthBearerStr:
		return dto.AuthBearerStr
	case dto.AuthServiceAccountStr:
		return dto.AuthServiceAccountStr
	case dto.AuthInteractiveStr:
		return dto.AuthInteractiveStr
	case dto.AuthNullStr:
		return dto.AuthNullStr
	case dto.AuthAWSSigningv4Str:
		return dto.AuthAWSSigningv4Str
	case dto.AuthCustomStr:
		return dto.AuthCustomStr
	case dto.OAuth2Str:
		return dto.OAuth2Str
	}
	if authCtx.KeyFilePath != "" || authCtx.KeyEnvVar != "" {
		return dto.AuthServiceAccountStr
	}
	return dto.AuthNullStr
}

func (gp *GenericProvider) Auth(
	authCtx *dto.AuthCtx,
	authTypeRequested string,
	enforceRevokeFirst bool,
) (*http.Client, error) {
	authCtx = authCtx.Clone()
	at := gp.inferAuthType(*authCtx, authTypeRequested)
	switch at {
	case dto.AuthAPIKeyStr:
		return gp.authUtil.ApiTokenAuth(authCtx, gp.runtimeCtx, false)
	case dto.AuthBearerStr:
		return gp.authUtil.ApiTokenAuth(authCtx, gp.runtimeCtx, true)
	case dto.AuthServiceAccountStr:
		scopes := authCtx.Scopes
		return gp.authUtil.GoogleOauthServiceAccount(gp.GetProviderString(), authCtx, scopes, gp.runtimeCtx)
	case dto.OAuth2Str:
		if authCtx.GrantType == dto.ClientCredentialsStr {
			scopes := authCtx.Scopes
			return gp.authUtil.GenericOauthClientCredentials(authCtx, scopes, gp.runtimeCtx)
		}
	case dto.AuthBasicStr:
		return gp.authUtil.BasicAuth(authCtx, gp.runtimeCtx)
	case dto.AuthCustomStr:
		return gp.authUtil.CustomAuth(authCtx, gp.runtimeCtx)
	case dto.AuthAzureDefaultStr:
		return gp.authUtil.AzureDefaultAuth(authCtx, gp.runtimeCtx)
	case dto.AuthInteractiveStr:
		return gp.authUtil.GCloudOAuth(gp.runtimeCtx, authCtx, enforceRevokeFirst)
	case dto.AuthAWSSigningv4Str:
		return gp.authUtil.AwsSigningAuth(authCtx, gp.runtimeCtx)
	case dto.AuthNullStr:
		return netutils.GetHTTPClient(gp.runtimeCtx, http.DefaultClient), nil
	}
	return nil, fmt.Errorf("could not infer auth type")
}

func (gp *GenericProvider) AuthRevoke(authCtx *dto.AuthCtx) error {
	return gp.authUtil.AuthRevoke(authCtx)
}

func (gp *GenericProvider) GetMethodForAction(
	serviceName string,
	resourceName string,
	iqlAction string,
	parameters parserutil.ColumnKeyedDatastore,
	runtimeCtx dto.RuntimeCtx,
) (formulation.StandardOperationStore, string, error) {
	rsc, err := gp.GetResource(serviceName, resourceName, runtimeCtx)
	if err != nil {
		return nil, "", err
	}
	return gp.methodSelector.GetMethodForAction(rsc, iqlAction, parameters)
}

func (gp *GenericProvider) GetDefaultHTTPClient() *http.Client {
	return gp.defaultClient
}

func (gp *GenericProvider) GetFirstMethodForAction(
	serviceName string,
	resourceName string,
	iqlAction string,
	runtimeCtx dto.RuntimeCtx,
) (formulation.StandardOperationStore, string, error) {
	rsc, err := gp.GetResource(serviceName, resourceName, runtimeCtx)
	if err != nil {
		return nil, "", err
	}
	rv, str, ok := rsc.GetFirstMethodFromSQLVerb(iqlAction)
	if !ok {
		return nil, "", fmt.Errorf("cannot locate method for action '%s'", iqlAction)
	}
	return rv, str, nil
}

func (gp *GenericProvider) InferDescribeMethod(
	rsc formulation.Resource,
) (formulation.StandardOperationStore, string, error) {
	if rsc == nil {
		return nil, "", fmt.Errorf("cannot infer describe method from nil resource")
	}
	m, mk, ok := rsc.GetFirstMethodFromSQLVerb("select")
	if ok {
		return m, mk, nil
	}
	return nil, "", fmt.Errorf(
		//nolint:lll // long message
		"SELECT not supported for this resource, use SHOW METHODS to view available operations for the resource and then invoke a supported method using the EXEC command")
}

func (gp *GenericProvider) GetObjectSchema(
	serviceName string,
	resourceName string,
	schemaName string,
) (formulation.Schema, error) {
	svc, err := gp.GetServiceShard(serviceName, resourceName, gp.runtimeCtx)
	if err != nil {
		return nil, err
	}
	return svc.GetSchema(schemaName)
}

func (gp *GenericProvider) ShowAuth(authCtx *dto.AuthCtx) (*formulation.AuthMetadata, error) {
	var err error
	var retVal formulation.AuthMetadata
	var authObj formulation.AuthMetadata
	if authCtx == nil {
		return nil, errors.New(constants.NotAuthenticatedShowStr) //nolint:stylecheck // happy with message
	}
	switch gp.inferAuthType(*authCtx, authCtx.Type) {
	case dto.AuthServiceAccountStr:
		sa, saErr := gp.authUtil.ParseServiceAccountFile(authCtx)
		if saErr == nil {
			authObj = formulation.AuthMetadata{
				Principal: sa.Email,
				Type:      strings.ToUpper(dto.AuthServiceAccountStr),
				Source:    authCtx.GetCredentialsSourceDescriptorString(),
			}
			retVal = authObj
			gp.authUtil.ActivateAuth(authCtx, sa.Email, dto.AuthServiceAccountStr)
		}
	case dto.AuthInteractiveStr:
		principal, sdkErr := gp.authUtil.GetCurrentGCloudOauthUser()
		if sdkErr == nil {
			principalStr := string(principal)
			if principalStr != "" {
				authObj = formulation.AuthMetadata{
					Principal: principalStr,
					Type:      strings.ToUpper(dto.AuthInteractiveStr),
					Source:    "OAuth",
				}
				retVal = authObj
				gp.authUtil.ActivateAuth(authCtx, principalStr, dto.AuthInteractiveStr)
			} else {
				err = errors.New(constants.NotAuthenticatedShowStr) //nolint:stylecheck // happy with message
			}
		} else {
			logging.GetLogger().Infoln(sdkErr)
			err = errors.New(constants.NotAuthenticatedShowStr) //nolint:stylecheck // happy with message
		}
	default:
		err = errors.New(constants.NotAuthenticatedShowStr) //nolint:stylecheck // happy with message
	}
	return &retVal, err
}

func (gp *GenericProvider) GetLikeableColumns(tableName string) []string {
	var retVal []string
	switch tableName {
	case "SERVICES":
		return []string{
			"id",
			"name",
		}
	case "RESOURCES":
		return []string{
			"id",
			"name",
		}
	case "METHODS":
		return []string{
			"id",
			"name",
		}
	case "PROVIDERS":
		return []string{
			"name",
		}
	}
	return retVal
}

func (gp *GenericProvider) EnhanceMetadataFilter(
	metadataType string,
	metadataFilter func(formulation.ITable) (formulation.ITable, error),
	colsVisited map[string]bool,
) (func(formulation.ITable) (formulation.ITable, error), error) {
	typeVisited, typeOk := colsVisited["type"]
	preferredVisited, preferredOk := colsVisited["preferred"]
	sqlTrue, sqlTrueErr := sqltypeutil.InterfaceToSQLType(true)
	sqlCloudStr, sqlCloudStrErr := sqltypeutil.InterfaceToSQLType("cloud")
	equalsOperator, operatorErr := relational.GetOperatorPredicate("=")
	if sqlTrueErr != nil || sqlCloudStrErr != nil || operatorErr != nil {
		return nil,
			//nolint:revive,stylecheck // exclamation marks for egregious error
			fmt.Errorf("typing and operator system broken!!!")
	}
	switch metadataType { //nolint:gocritic // happy to have this as a switch
	case "service":
		if typeOk && typeVisited && preferredOk && preferredVisited {
			return metadataFilter, nil
		}
		if typeOk && typeVisited {
			return relational.AndTableFilters(
				metadataFilter,
				relational.ConstructTablePredicateFilter("preferred", sqlTrue, equalsOperator),
			), nil
		}
		if preferredOk && preferredVisited {
			return relational.AndTableFilters(
				metadataFilter,
				relational.ConstructTablePredicateFilter("type", sqlCloudStr, equalsOperator),
			), nil
		}
		return relational.AndTableFilters(
			relational.AndTableFilters(
				metadataFilter,
				relational.ConstructTablePredicateFilter("cloud", sqlCloudStr, equalsOperator),
			),
			relational.ConstructTablePredicateFilter("preferred", sqlTrue, equalsOperator),
		), nil
	}
	return metadataFilter, nil
}

func (gp *GenericProvider) getProviderServices() (map[string]formulation.ProviderService, error) {
	retVal := make(map[string]formulation.ProviderService)
	disDoc, err := gp.discoveryAdapter.GetServiceHandlesMap(gp.provider)
	if err != nil {
		return nil, err
	}
	for k, item := range disDoc {
		retVal[docparser.TranslateServiceKeyGenericProviderToIql(k)] = item
	}
	return retVal, nil
}

//nolint:revive // future proofing
func (gp *GenericProvider) GetProviderServicesRedacted(
	runtimeCtx dto.RuntimeCtx,
	extended bool,
) (map[string]formulation.ProviderService, error) {
	return gp.getProviderServices()
}

//nolint:revive // future proofing
func (gp *GenericProvider) GetResourcesRedacted(
	currentService string,
	runtimeCtx dto.RuntimeCtx,
	extended bool,
) (map[string]formulation.Resource, error) {
	svcDiscDocMap, err := gp.discoveryAdapter.GetResourcesMap(gp.provider, currentService)
	return svcDiscDocMap, err
}

func (gp *GenericProvider) CheckCredentialFile(authCtx *dto.AuthCtx) error {
	switch authCtx.Type {
	case dto.AuthServiceAccountStr:
		_, err := gp.authUtil.ParseServiceAccountFile(authCtx)
		return err
	case dto.AuthAPIKeyStr:
		_, err := authCtx.GetCredentialsBytes()
		return err
	}
	return fmt.Errorf("auth type = '%s' not supported", authCtx.Type)
}

func (gp *GenericProvider) SetCurrentService(serviceKey string) {
	gp.currentService = serviceKey
}

func (gp *GenericProvider) GetCurrentService() string {
	return gp.currentService
}

//nolint:revive // future proofing
func (gp *GenericProvider) GetResourcesMap(
	serviceKey string,
	runtimeCtx dto.RuntimeCtx,
) (map[string]formulation.Resource, error) {
	return gp.discoveryAdapter.GetResourcesMap(gp.provider, serviceKey)
}

func (gp *GenericProvider) GetResource(
	serviceKey string,
	resourceKey string,
	runtimeCtx dto.RuntimeCtx,
) (formulation.Resource, error) {
	svc, err := gp.GetServiceShard(serviceKey, resourceKey, runtimeCtx)
	if err != nil {
		return nil, err
	}
	return svc.GetResource(resourceKey)
}

func (gp *GenericProvider) GetProviderString() string {
	return gp.provider.GetName()
}

func (gp *GenericProvider) GetProvider() (formulation.Provider, error) {
	if gp.provider == nil {
		return nil, fmt.Errorf("nil provider object")
	}
	return gp.provider, nil
}

func (gp *GenericProvider) InferMaxResultsElement(formulation.OperationStore) sdk_internal_dto.HTTPElement {
	return sdk_internal_dto.NewHTTPElement(
		sdk_internal_dto.QueryParam,
		"maxResults",
	)
}

func (gp *GenericProvider) InferNextPageRequestElement(ho internaldto.Heirarchy) sdk_internal_dto.HTTPElement {
	st, ok := gp.getPaginationRequestTokenSemantic(ho)
	if ok {
		if tp, err := sdk_internal_dto.ExtractHTTPElement(st.GetLocation()); err == nil {
			rv := sdk_internal_dto.NewHTTPElement(
				tp,
				st.GetKey(),
			)
			transformer, tErr := st.GetTransformer()
			if tErr == nil && transformer != nil {
				rv.SetTransformer(transformer)
			}
			return rv
		}
	}
	switch gp.GetProviderString() {
	case "github", "okta":
		return sdk_internal_dto.NewHTTPElement(
			sdk_internal_dto.RequestString,
			"",
		)
	default:
		return sdk_internal_dto.NewHTTPElement(
			sdk_internal_dto.QueryParam,
			"pageToken",
		)
	}
}

func (gp *GenericProvider) getPaginationRequestTokenSemantic(
	ho internaldto.Heirarchy,
) (formulation.TokenSemantic, bool) {
	if ho.GetMethod() == nil {
		return nil, false
	}
	return ho.GetMethod().GetPaginationRequestTokenSemantic()
}

func (gp *GenericProvider) getPaginationResponseTokenSemantic(
	ho internaldto.Heirarchy,
) (formulation.TokenSemantic, bool) {
	if ho.GetMethod() == nil {
		return nil, false
	}
	return ho.GetMethod().GetPaginationResponseTokenSemantic()
}

func (gp *GenericProvider) InferNextPageResponseElement(ho internaldto.Heirarchy) sdk_internal_dto.HTTPElement {
	st, ok := gp.getPaginationResponseTokenSemantic(ho)
	if ok {
		if tp, err := sdk_internal_dto.ExtractHTTPElement(st.GetLocation()); err == nil {
			rv := sdk_internal_dto.NewHTTPElement(
				tp,
				st.GetKey(),
			)
			transformer, tErr := st.GetTransformer()
			if tErr == nil && transformer != nil {
				rv.SetTransformer(transformer)
			}
			return rv
		}
	}
	switch gp.GetProviderString() {
	case "github", "okta":
		rv := sdk_internal_dto.NewHTTPElement(
			sdk_internal_dto.Header,
			"Link",
		)
		rv.SetTransformer(formulation.DefaultLinkHeaderTransformer)
		return rv
	default:
		return sdk_internal_dto.NewHTTPElement(
			sdk_internal_dto.BodyAttribute,
			"nextPageToken",
		)
	}
}
