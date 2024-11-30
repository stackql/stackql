package provider

import (
	"errors"
	"fmt"

	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/discovery"
	"github.com/stackql/stackql/internal/stackql/docparser"
	"github.com/stackql/stackql/internal/stackql/dto"
	google_sdk "github.com/stackql/stackql/internal/stackql/google_sdk"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/methodselect"
	"github.com/stackql/stackql/internal/stackql/netutils"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/relational"

	"github.com/stackql/stackql/pkg/sqltypeutil"

	sdk_internal_dto "github.com/stackql/any-sdk/pkg/internaldto"

	"net/http"
	"regexp"
	"strings"
)

var (
	//nolint:revive,unused // prefer declarative
	gitHubLinksNextRegex *regexp.Regexp = regexp.MustCompile(`.*<(?P<nextURL>[^>]*)>;\ rel="next".*`)
)

type GenericProvider struct {
	provider         anysdk.Provider
	runtimeCtx       dto.RuntimeCtx
	currentService   string
	discoveryAdapter discovery.IDiscoveryAdapter
	apiVersion       string
	methodSelector   methodselect.IMethodSelector
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
) (anysdk.Service, error) {
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
		return gp.apiTokenFileAuth(authCtx, false)
	case dto.AuthBearerStr:
		return gp.apiTokenFileAuth(authCtx, true)
	case dto.AuthServiceAccountStr:
		return gp.googleKeyFileAuth(authCtx)
	case dto.OAuth2Str:
		if authCtx.GrantType == dto.ClientCredentialsStr {
			return gp.clientCredentialsAuth(authCtx)
		}
	case dto.AuthBasicStr:
		return gp.basicAuth(authCtx)
	case dto.AuthCustomStr:
		return gp.customAuth(authCtx)
	case dto.AuthAzureDefaultStr:
		return gp.azureDefaultAuth(authCtx)
	case dto.AuthInteractiveStr:
		return gp.oAuth(authCtx, enforceRevokeFirst)
	case dto.AuthAWSSigningv4Str:
		return gp.awsSigningAuth(authCtx)
	case dto.AuthNullStr:
		return netutils.GetHTTPClient(gp.runtimeCtx, http.DefaultClient), nil
	}
	return nil, fmt.Errorf("could not infer auth type")
}

func (gp *GenericProvider) AuthRevoke(authCtx *dto.AuthCtx) error {
	switch strings.ToLower(authCtx.Type) {
	case dto.AuthServiceAccountStr:
		return errors.New(constants.ServiceAccountRevokeErrStr)
	case dto.AuthInteractiveStr:
		err := google_sdk.RevokeGoogleAuth()
		if err == nil {
			deactivateAuth(authCtx)
		}
		return err
	}
	return fmt.Errorf(`Auth revoke for Google Failed; improper auth method: "%s" specified`, authCtx.Type)
}

func (gp *GenericProvider) GetMethodForAction(
	serviceName string,
	resourceName string,
	iqlAction string,
	parameters parserutil.ColumnKeyedDatastore,
	runtimeCtx dto.RuntimeCtx,
) (anysdk.OperationStore, string, error) {
	rsc, err := gp.GetResource(serviceName, resourceName, runtimeCtx)
	if err != nil {
		return nil, "", err
	}
	return gp.methodSelector.GetMethodForAction(rsc, iqlAction, parameters)
}

func (gp *GenericProvider) GetFirstMethodForAction(
	serviceName string,
	resourceName string,
	iqlAction string,
	runtimeCtx dto.RuntimeCtx,
) (anysdk.OperationStore, string, error) {
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
	rsc anysdk.Resource,
) (anysdk.OperationStore, string, error) {
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
) (anysdk.Schema, error) {
	svc, err := gp.GetServiceShard(serviceName, resourceName, gp.runtimeCtx)
	if err != nil {
		return nil, err
	}
	return svc.GetSchema(schemaName)
}

func (gp *GenericProvider) ShowAuth(authCtx *dto.AuthCtx) (*anysdk.AuthMetadata, error) {
	var err error
	var retVal *anysdk.AuthMetadata
	var authObj anysdk.AuthMetadata
	if authCtx == nil {
		return nil, errors.New(constants.NotAuthenticatedShowStr) //nolint:stylecheck // happy with message
	}
	switch gp.inferAuthType(*authCtx, authCtx.Type) {
	case dto.AuthServiceAccountStr:
		var sa serviceAccount
		sa, err = parseServiceAccountFile(authCtx)
		if err == nil {
			authObj = anysdk.AuthMetadata{
				Principal: sa.Email,
				Type:      strings.ToUpper(dto.AuthServiceAccountStr),
				Source:    authCtx.GetCredentialsSourceDescriptorString(),
			}
			retVal = &authObj
			activateAuth(authCtx, sa.Email, dto.AuthServiceAccountStr)
		}
	case dto.AuthInteractiveStr:
		principal, sdkErr := google_sdk.GetCurrentAuthUser()
		if sdkErr == nil {
			principalStr := string(principal)
			if principalStr != "" {
				authObj = anysdk.AuthMetadata{
					Principal: principalStr,
					Type:      strings.ToUpper(dto.AuthInteractiveStr),
					Source:    "OAuth",
				}
				retVal = &authObj
				activateAuth(authCtx, principalStr, dto.AuthInteractiveStr)
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
	return retVal, err
}

func (gp *GenericProvider) oAuth(authCtx *dto.AuthCtx, enforceRevokeFirst bool) (*http.Client, error) {
	var err error
	var tokenBytes []byte
	tokenBytes, err = google_sdk.GetAccessToken()
	if enforceRevokeFirst && authCtx.Type == dto.AuthInteractiveStr && err == nil {
		return nil, fmt.Errorf(constants.OAuthInteractiveAuthErrStr) //nolint:stylecheck // happy with message
	}
	if err != nil {
		err = google_sdk.OAuthToGoogle()
		if err == nil {
			tokenBytes, err = google_sdk.GetAccessToken()
		}
	}
	if err != nil {
		return nil, err
	}
	activateAuth(authCtx, "", dto.AuthInteractiveStr)
	client := netutils.GetHTTPClient(gp.runtimeCtx, nil)
	tr, err := newTransport(tokenBytes, authTypeBearer, authCtx.ValuePrefix, locationHeader, "", client.Transport)
	if err != nil {
		return nil, err
	}
	client.Transport = tr
	return client, nil
}

func (gp *GenericProvider) googleKeyFileAuth(authCtx *dto.AuthCtx) (*http.Client, error) {
	scopes := authCtx.Scopes
	return googleOauthServiceAccount(gp.GetProviderString(), authCtx, scopes, gp.runtimeCtx)
}

func (gp *GenericProvider) clientCredentialsAuth(authCtx *dto.AuthCtx) (*http.Client, error) {
	scopes := authCtx.Scopes
	return genericOauthClientCredentials(authCtx, scopes, gp.runtimeCtx)
}

func (gp *GenericProvider) apiTokenFileAuth(authCtx *dto.AuthCtx, enforceBearer bool) (*http.Client, error) {
	return apiTokenAuth(authCtx, gp.runtimeCtx, enforceBearer)
}

func (gp *GenericProvider) awsSigningAuth(authCtx *dto.AuthCtx) (*http.Client, error) {
	return awsSigningAuth(authCtx, gp.runtimeCtx)
}

func (gp *GenericProvider) basicAuth(authCtx *dto.AuthCtx) (*http.Client, error) {
	return basicAuth(authCtx, gp.runtimeCtx)
}

func (gp *GenericProvider) customAuth(authCtx *dto.AuthCtx) (*http.Client, error) {
	return customAuth(authCtx, gp.runtimeCtx)
}

func (gp *GenericProvider) azureDefaultAuth(authCtx *dto.AuthCtx) (*http.Client, error) {
	return azureDefaultAuth(authCtx, gp.runtimeCtx)
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
	metadataFilter func(anysdk.ITable) (anysdk.ITable, error),
	colsVisited map[string]bool,
) (func(anysdk.ITable) (anysdk.ITable, error), error) {
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

func (gp *GenericProvider) getProviderServices() (map[string]anysdk.ProviderService, error) {
	retVal := make(map[string]anysdk.ProviderService)
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
) (map[string]anysdk.ProviderService, error) {
	return gp.getProviderServices()
}

//nolint:revive // future proofing
func (gp *GenericProvider) GetResourcesRedacted(
	currentService string,
	runtimeCtx dto.RuntimeCtx,
	extended bool,
) (map[string]anysdk.Resource, error) {
	svcDiscDocMap, err := gp.discoveryAdapter.GetResourcesMap(gp.provider, currentService)
	return svcDiscDocMap, err
}

func (gp *GenericProvider) CheckCredentialFile(authCtx *dto.AuthCtx) error {
	switch authCtx.Type {
	case dto.AuthServiceAccountStr:
		_, err := parseServiceAccountFile(authCtx)
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
) (map[string]anysdk.Resource, error) {
	return gp.discoveryAdapter.GetResourcesMap(gp.provider, serviceKey)
}

func (gp *GenericProvider) GetResource(
	serviceKey string,
	resourceKey string,
	runtimeCtx dto.RuntimeCtx,
) (anysdk.Resource, error) {
	svc, err := gp.GetServiceShard(serviceKey, resourceKey, runtimeCtx)
	if err != nil {
		return nil, err
	}
	return svc.GetResource(resourceKey)
}

func (gp *GenericProvider) GetProviderString() string {
	return gp.provider.GetName()
}

func (gp *GenericProvider) GetProvider() (anysdk.Provider, error) {
	if gp.provider == nil {
		return nil, fmt.Errorf("nil provider object")
	}
	return gp.provider, nil
}

func (gp *GenericProvider) InferMaxResultsElement(anysdk.OperationStore) sdk_internal_dto.HTTPElement {
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
) (anysdk.TokenSemantic, bool) {
	if ho.GetMethod() == nil {
		return nil, false
	}
	return ho.GetMethod().GetPaginationRequestTokenSemantic()
}

func (gp *GenericProvider) getPaginationResponseTokenSemantic(
	ho internaldto.Heirarchy,
) (anysdk.TokenSemantic, bool) {
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
		rv.SetTransformer(anysdk.DefaultLinkHeaderTransformer)
		return rv
	default:
		return sdk_internal_dto.NewHTTPElement(
			sdk_internal_dto.BodyAttribute,
			"nextPageToken",
		)
	}
}
