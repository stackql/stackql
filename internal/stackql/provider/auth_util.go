package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/netutils"

	"net/http"
	"regexp"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
)

var (
	storageObjectsRegex *regexp.Regexp = regexp.MustCompile(`^storage\.objects\..*$`)
)

type serviceAccount struct {
	Email      string `json:"client_email"`
	PrivateKey string `json:"private_key"`
}

type transport struct {
	token               []byte
	authType            string
	authValuePrefix     string
	tokenLocation       string
	key                 string
	underlyingTransport http.RoundTripper
}

func newTransport(token []byte, authType, authValuePrefix, tokenLocation, key string, underlyingTransport http.RoundTripper) (*transport, error) {
	switch authType {
	case authTypeBasic, authTypeBearer, authTypeSSWS:
		if len(token) < 1 {
			return nil, fmt.Errorf("no credentials provided for auth type = '%s'", authType)
		}
		if tokenLocation != locationHeader {
			return nil, fmt.Errorf("improper location provided for auth type = '%s', provided = '%s', expected = '%s'", authType, tokenLocation, locationHeader)
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
	return &transport{
		token:               token,
		authType:            authType,
		authValuePrefix:     authValuePrefix,
		tokenLocation:       tokenLocation,
		key:                 key,
		underlyingTransport: underlyingTransport,
	}, nil
}

const (
	locationHeader string = "header"
	locationQuery  string = "query"
	authTypeBasic  string = "BASIC"
	authTypeBearer string = "Bearer"
	authTypeSSWS   string = "SSWS"
)

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch t.tokenLocation {
	case locationHeader:
		switch t.authType {
		case authTypeBasic, authTypeBearer, authTypeSSWS:
			authValuePrefix := t.authValuePrefix
			if t.authValuePrefix == "" {
				authValuePrefix = fmt.Sprintf("%s ", t.authType)
			}
			req.Header.Set(
				"Authorization",
				fmt.Sprintf("%s%s", authValuePrefix, string(t.token)),
			)
		default:
			req.Header.Set(
				t.key,
				string(t.token),
			)
		}
	case locationQuery:
		qv := req.URL.Query()
		qv.Set(
			t.key, string(t.token),
		)
		req.URL.RawQuery = qv.Encode()
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
		return c, fmt.Errorf(constants.ServiceAccountPathErrStr)
	}
	return c, json.Unmarshal(b, &c)
}

func getJWTConfig(provider string, credentialsBytes []byte, scopes []string) (*jwt.Config, error) {
	switch provider {
	case "google":
		return google.JWTConfigFromJSON(credentialsBytes, scopes...)
	default:
		return nil, fmt.Errorf("service account auth for provider = '%s' currently not supported", provider)
	}
}

func oauthServiceAccount(provider string, authCtx *dto.AuthCtx, scopes []string, runtimeCtx dto.RuntimeCtx) (*http.Client, error) {
	b, err := authCtx.GetCredentialsBytes()
	if err != nil {
		return nil, fmt.Errorf("service account credentials error: %v", err)
	}
	config, errToken := getJWTConfig(provider, b, scopes)
	if errToken != nil {
		return nil, errToken
	}
	activateAuth(authCtx, "", dto.AuthServiceAccountStr)
	httpClient := netutils.GetHttpClient(runtimeCtx, http.DefaultClient)
	if DummyAuth {
		// return httpClient, nil
	}
	return config.Client(context.WithValue(oauth2.NoContext, oauth2.HTTPClient, httpClient)), nil
}

func apiTokenAuth(authCtx *dto.AuthCtx, runtimeCtx dto.RuntimeCtx) (*http.Client, error) {
	b, err := authCtx.GetCredentialsBytes()
	if err != nil {
		return nil, fmt.Errorf("credentials error: %v", err)
	}
	activateAuth(authCtx, "", "api_key")
	httpClient := netutils.GetHttpClient(runtimeCtx, http.DefaultClient)
	tr, err := newTransport(b, authTypeSSWS, authCtx.ValuePrefix, locationHeader, "", httpClient.Transport)
	if err != nil {
		return nil, err
	}
	httpClient.Transport = tr
	return httpClient, nil
}

func basicAuth(authCtx *dto.AuthCtx, runtimeCtx dto.RuntimeCtx) (*http.Client, error) {
	b, err := authCtx.GetCredentialsBytes()
	if err != nil {
		return nil, fmt.Errorf("credentials error: %v", err)
	}
	activateAuth(authCtx, "", "basic")
	httpClient := netutils.GetHttpClient(runtimeCtx, http.DefaultClient)
	tr, err := newTransport(b, authTypeBasic, authCtx.ValuePrefix, locationHeader, "", httpClient.Transport)
	if err != nil {
		return nil, err
	}
	httpClient.Transport = tr
	return httpClient, nil
}
