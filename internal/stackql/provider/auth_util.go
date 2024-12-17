package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/stackql/any-sdk/pkg/litetemplate"
	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/netutils"
	"github.com/stackql/stackql/pkg/awssign"
	"github.com/stackql/stackql/pkg/azureauth"

	"net/http"
	"regexp"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
)

var (
	storageObjectsRegex *regexp.Regexp = regexp.MustCompile(`^storage\.objects\..*$`) //nolint:unused,revive,nolintlint,lll // prefer declarative
)

type serviceAccount struct {
	Email      string `json:"client_email"`
	PrivateKey string `json:"private_key"`
}

type tokenCfg struct {
	token           []byte
	authType        string
	authValuePrefix string
	tokenLocation   string
	key             string
}

func newTokenConfig(
	token []byte,
	authType,
	authValuePrefix,
	tokenLocation,
	key string,
) *tokenCfg {
	return &tokenCfg{
		token:           token,
		authType:        authType,
		authValuePrefix: authValuePrefix,
		tokenLocation:   tokenLocation,
		key:             key,
	}
}

type transport struct {
	tokenConfigs        []*tokenCfg
	underlyingTransport http.RoundTripper
}

func newTransport(
	token []byte,
	authType,
	authValuePrefix,
	tokenLocation,
	key string,
	underlyingTransport http.RoundTripper,
) (*transport, error) {
	switch authType {
	case authTypeBasic, authTypeBearer, authTypeAPIKey:
		if len(token) < 1 {
			return nil, fmt.Errorf("no credentials provided for auth type = '%s'", authType)
		}
		if tokenLocation != locationHeader {
			return nil, fmt.Errorf(
				"improper location provided for auth type = '%s', provided = '%s', expected = '%s'",
				authType, tokenLocation, locationHeader)
		}
	default:
		switch tokenLocation {
		case locationHeader:
		case locationQuery:
			if key == "" {
				return nil, fmt.Errorf("key required for query param based auth")
			}
		default:
			return nil, fmt.Errorf("token location not supported: '%s'", tokenLocation)
		}
	}
	tokenConfigObj := newTokenConfig(token, authType, authValuePrefix, tokenLocation, key)
	return &transport{
		tokenConfigs:        []*tokenCfg{tokenConfigObj},
		underlyingTransport: underlyingTransport,
	}, nil
}

//nolint:unparam // future proofing
func (t *transport) addTokenCfg(tokenConfig *tokenCfg) error {
	t.tokenConfigs = append(t.tokenConfigs, tokenConfig)
	return nil
}

const (
	locationHeader string = "header"
	locationQuery  string = "query"
	authTypeBasic  string = "BASIC"
	authTypeCustom string = "custom"
	authTypeBearer string = "Bearer"
	authTypeAPIKey string = "api_key"
)

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	for _, tc := range t.tokenConfigs {
		tokenConfig := tc
		switch tokenConfig.tokenLocation {
		case locationHeader:
			switch tokenConfig.authType {
			case authTypeBasic, authTypeBearer, authTypeAPIKey:
				authValuePrefix := tokenConfig.authValuePrefix
				if tokenConfig.authValuePrefix == "" {
					authValuePrefix = fmt.Sprintf("%s ", tokenConfig.authType)
				}
				req.Header.Set(
					"Authorization",
					fmt.Sprintf("%s%s", authValuePrefix, string(tokenConfig.token)),
				)
			default:
				req.Header.Set(
					tokenConfig.key,
					string(tokenConfig.token),
				)
			}
		case locationQuery:
			qv := req.URL.Query()
			qv.Set(
				tokenConfig.key, string(tokenConfig.token),
			)
			req.URL.RawQuery = qv.Encode()
		}
	}
	return t.underlyingTransport.RoundTrip(req)
}

func activateAuth(authCtx *dto.AuthCtx, principal string, authType string) {
	authCtx.Active = true
	authCtx.Type = authType
	if principal != "" {
		authCtx.ID = principal
	}
}

func deactivateAuth(authCtx *dto.AuthCtx) {
	authCtx.Active = false
}

func parseServiceAccountFile(ac *dto.AuthCtx) (serviceAccount, error) {
	b, err := ac.GetCredentialsBytes()
	var c serviceAccount
	if err != nil {
		return c, fmt.Errorf(constants.ServiceAccountPathErrStr) //nolint:stylecheck //TODO: review
	}
	return c, json.Unmarshal(b, &c)
}

func getGoogleJWTConfig(
	provider string,
	credentialsBytes []byte,
	scopes []string,
	subject string,
) (*jwt.Config, error) {
	switch provider {
	case "google", "googleads", "googleanalytics",
		"googledevelopers", "googlemybusiness", "googleworkspace",
		"youtube", "googleadmin":
		if scopes == nil {
			scopes = []string{
				"https://www.googleapis.com/auth/cloud-platform",
			}
		}
		rv, err := google.JWTConfigFromJSON(credentialsBytes, scopes...)
		if err != nil {
			return nil, err
		}
		if subject != "" {
			rv.Subject = subject
		}
		return rv, nil
	default:
		return nil, fmt.Errorf("service account auth for provider = '%s' currently not supported", provider)
	}
}

func getGenericClientCredentialsConfig(authCtx *dto.AuthCtx, scopes []string) (*clientcredentials.Config, error) {
	clientID, clientIDErr := authCtx.GetClientID()
	if clientIDErr != nil {
		return nil, clientIDErr
	}
	clientSecret, secretErr := authCtx.GetClientSecret()
	if secretErr != nil {
		return nil, secretErr
	}
	templatedTokenURL, templateErr := litetemplate.RenderTemplateFromSerializable(authCtx.GetTokenURL(), authCtx)
	if templateErr != nil {
		return nil, fmt.Errorf("incorrect token url templating %w", templateErr)
	}
	rv := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       scopes,
		TokenURL:     templatedTokenURL,
	}
	if len(authCtx.GetValues()) > 0 {
		rv.EndpointParams = authCtx.GetValues()
	}
	if authCtx.GetAuthStyle() > 0 {
		rv.AuthStyle = oauth2.AuthStyle(authCtx.GetAuthStyle())
	}
	return rv, nil
}

func googleOauthServiceAccount(
	provider string,
	authCtx *dto.AuthCtx,
	scopes []string,
	runtimeCtx dto.RuntimeCtx,
) (*http.Client, error) {
	b, err := authCtx.GetCredentialsBytes()
	if err != nil {
		return nil, fmt.Errorf("service account credentials error: %w", err)
	}
	config, errToken := getGoogleJWTConfig(provider, b, scopes, authCtx.Subject)
	if errToken != nil {
		return nil, errToken
	}
	activateAuth(authCtx, "", dto.AuthServiceAccountStr)
	httpClient := netutils.GetHTTPClient(runtimeCtx, http.DefaultClient)
	return config.Client(context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)), nil
}

func genericOauthClientCredentials(
	authCtx *dto.AuthCtx,
	scopes []string,
	runtimeCtx dto.RuntimeCtx,
) (*http.Client, error) {
	config, errToken := getGenericClientCredentialsConfig(authCtx, scopes)
	if errToken != nil {
		return nil, errToken
	}
	activateAuth(authCtx, "", dto.ClientCredentialsStr)
	httpClient := netutils.GetHTTPClient(runtimeCtx, http.DefaultClient)
	return config.Client(context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)), nil
}

func apiTokenAuth(authCtx *dto.AuthCtx, runtimeCtx dto.RuntimeCtx, enforceBearer bool) (*http.Client, error) {
	b, err := authCtx.GetCredentialsBytes()
	if err != nil {
		return nil, fmt.Errorf("credentials error: %w", err)
	}
	activateAuth(authCtx, "", "api_key")
	httpClient := netutils.GetHTTPClient(runtimeCtx, http.DefaultClient)
	valPrefix := authCtx.ValuePrefix
	if enforceBearer {
		valPrefix = "Bearer "
	}
	tr, err := newTransport(b, authTypeAPIKey, valPrefix, locationHeader, "", httpClient.Transport)
	if err != nil {
		return nil, err
	}
	httpClient.Transport = tr
	return httpClient, nil
}

func awsSigningAuth(authCtx *dto.AuthCtx, runtimeCtx dto.RuntimeCtx) (*http.Client, error) {
	// Retrieve the AWS access key and secret key.
	credentialsBytes, err := authCtx.GetCredentialsBytes()
	if err != nil {
		return nil, fmt.Errorf("credentials error: %w", err)
	}
	keyStr := string(credentialsBytes)

	// Retrieve the AWS access key ID.
	keyID, err := authCtx.GetKeyIDString()
	if err != nil {
		return nil, err
	}

	// Validate that both keyID and keyStr are not empty.
	if keyStr == "" || keyID == "" {
		return nil, fmt.Errorf("cannot compose AWS signing credentials")
	}

	// Retrieve the optional session token. Note: No error handling for missing session token.
	sessionToken, _ := authCtx.GetAwsSessionTokenString()

	// Mark the authentication context as active for AWS signing.
	activateAuth(authCtx, "", dto.AuthAWSSigningv4Str)

	// Get the HTTP client from the runtime context.
	httpClient := netutils.GetHTTPClient(runtimeCtx, http.DefaultClient)

	// Initialize the AWS signing transport with credentials and optional session token.
	tr, err := awssign.NewAwsSignTransport(httpClient.Transport, keyID, keyStr, sessionToken)
	if err != nil {
		return nil, err
	}

	// Set the custom AWS signing transport as the client's transport.
	httpClient.Transport = tr

	return httpClient, nil
}

func basicAuth(authCtx *dto.AuthCtx, runtimeCtx dto.RuntimeCtx) (*http.Client, error) {
	b, err := authCtx.GetCredentialsBytes()
	if err != nil {
		return nil, fmt.Errorf("credentials error: %w", err)
	}
	activateAuth(authCtx, "", "basic")
	httpClient := netutils.GetHTTPClient(runtimeCtx, http.DefaultClient)
	tr, err := newTransport(b, authTypeBasic, authCtx.ValuePrefix, locationHeader, "", httpClient.Transport)
	if err != nil {
		return nil, err
	}
	httpClient.Transport = tr
	return httpClient, nil
}

func customAuth(authCtx *dto.AuthCtx, runtimeCtx dto.RuntimeCtx) (*http.Client, error) {
	b, err := authCtx.GetCredentialsBytes()
	if err != nil {
		return nil, fmt.Errorf("credentials error: %w", err)
	}
	activateAuth(authCtx, "", "custom")
	httpClient := netutils.GetHTTPClient(runtimeCtx, http.DefaultClient)
	tr, err := newTransport(b, authTypeCustom, authCtx.ValuePrefix, authCtx.Location, authCtx.Name, httpClient.Transport)
	if err != nil {
		return nil, err
	}
	successor, successorExists := authCtx.GetSuccessor()
	for {
		if successorExists {
			successorCredentialsBytes, sbErr := successor.GetCredentialsBytes()
			if sbErr != nil {
				return nil, fmt.Errorf("successor credentials error: %w", sbErr)
			}
			successorTokenConfig := newTokenConfig(
				successorCredentialsBytes,
				authTypeCustom,
				successor.ValuePrefix,
				successor.Location,
				successor.Name,
			)
			addTknErr := tr.addTokenCfg(successorTokenConfig)
			if addTknErr != nil {
				return nil, addTknErr
			}
			successor, successorExists = successor.GetSuccessor()
		} else {
			break
		}
	}
	httpClient.Transport = tr
	return httpClient, nil
}

func azureDefaultAuth(authCtx *dto.AuthCtx, runtimeCtx dto.RuntimeCtx) (*http.Client, error) {
	azureTokenSource, err := azureauth.NewDefaultCredentialAzureTokenSource()
	if err != nil {
		return nil, fmt.Errorf("azure default credentials error: %w", err)
	}
	token, err := azureTokenSource.GetToken(context.Background())
	if err != nil {
		return nil, fmt.Errorf("azure default credentials token error: %w", err)
	}
	tokenString := token.Token
	activateAuth(authCtx, "", "azure_default")
	httpClient := netutils.GetHTTPClient(runtimeCtx, http.DefaultClient)
	tr, err := newTransport([]byte(tokenString), authTypeBearer, "Bearer ", locationHeader, "", httpClient.Transport)
	if err != nil {
		return nil, err
	}
	httpClient.Transport = tr
	return httpClient, nil
}
