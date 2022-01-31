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
	underlyingTransport http.RoundTripper
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set(
		"Authorization",
		fmt.Sprintf("%s %s", t.authType, string(t.token)),
	)
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

func oauthServiceAccount(authCtx *dto.AuthCtx, scopes []string, runtimeCtx dto.RuntimeCtx) (*http.Client, error) {
	b, err := authCtx.GetCredentialsBytes()
	if err != nil {
		return nil, fmt.Errorf("service account credentials error: %v", err)
	}
	config, errToken := google.JWTConfigFromJSON(b, scopes...)
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
	httpClient.Transport = &transport{
		token:               b,
		authType:            "SSWS",
		underlyingTransport: httpClient.Transport,
	}
	return httpClient, nil
}
