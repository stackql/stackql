package mcp_server //nolint:testpackage,revive // exercise the real HTTP auth boundary

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sirupsen/logrus"
)

const (
	oauthIssuerURL   = "https://issuer.example.test/"
	oauthAudience    = "https://mcp.example.test/api"
	oauthMetadataURL = "https://mcp.example.test/.well-known/oauth-protected-resource"
	oauthKeyID       = "integration-key"
)

type oauthIntegrationIssuer struct {
	server  *httptest.Server
	public  ed25519.PublicKey
	private ed25519.PrivateKey
}

func newOAuthIntegrationIssuer(t *testing.T) *oauthIntegrationIssuer {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	issuer := &oauthIntegrationIssuer{public: public, private: private}
	issuer.server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
			Key: issuer.public, KeyID: oauthKeyID, Algorithm: "EdDSA", Use: "sig",
		}}})
	}))
	t.Cleanup(issuer.server.Close)
	return issuer
}

func (i *oauthIntegrationIssuer) token(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["kid"] = oauthKeyID
	raw, err := token.SignedString(i.private)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestOAuthConfigLoadsAndDefaultsToOptOut(t *testing.T) {
	if DefaultHTTPConfig().Server.OAuth != nil {
		t.Fatal("OAuth must remain opt-in by default")
	}
	jsonConfig := `{"server":{"transport":"http","oauth":{"issuer":"https://issuer.example.test/","audience":"https://mcp.example.test/api","resource":"https://mcp.example.test/api","jwks_url":"https://issuer.example.test/jwks","metadata_url":"https://mcp.example.test/.well-known/oauth-protected-resource","scopes":["tools:read"],"algorithms":["EdDSA"]}}}`
	loadedJSON, err := LoadFromJSON([]byte(jsonConfig))
	if err != nil {
		t.Fatalf("LoadFromJSON with OAuth: %v", err)
	}
	if loadedJSON.Server.OAuth == nil || loadedJSON.Server.OAuth.Issuer != oauthIssuerURL {
		t.Fatalf("OAuth settings were not loaded: %+v", loadedJSON.Server.OAuth)
	}

	yamlConfig := `server:
  transport: http
  oauth:
    issuer: https://issuer.example.test/
    audience: https://mcp.example.test/api
    resource: https://mcp.example.test/api
    jwks_url: https://issuer.example.test/jwks
    metadata_url: https://mcp.example.test/.well-known/oauth-protected-resource
    scopes:
      - tools:read
    algorithms:
      - EdDSA
    cache_ttl: 2m
    clock_skew: 30s
`
	loadedYAML, err := LoadFromYAML([]byte(yamlConfig))
	if err != nil {
		t.Fatalf("LoadFromYAML with OAuth: %v", err)
	}
	if loadedYAML.Server.OAuth == nil ||
		loadedYAML.Server.OAuth.CacheTTL != Duration(2*time.Minute) ||
		loadedYAML.Server.OAuth.ClockSkew != Duration(30*time.Second) {
		t.Fatalf("OAuth duration settings were not loaded: %+v", loadedYAML.Server.OAuth)
	}
}

func TestOAuthConfigFailsClosedOnConflictingModesAndInvalidSettings(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Config)
	}{
		{"pre-shared bearer", func(c *Config) { c.Server.AuthTokenEnvVar = "MCP_TOKEN" }},
		{"unauthenticated opt-in", func(c *Config) { c.Server.AllowUnauthenticated = true }},
		{"embedding authentication hook", func(c *Config) {
			c.AuthenticateRequest = func(r *http.Request) (context.Context, error) {
				return r.Context(), nil
			}
		}},
		{"stdio transport", func(c *Config) { c.Server.Transport = serverTransportStdIO }},
		{"mTLS client authentication", func(c *Config) {
			c.TLSConfig = &tls.Config{ClientAuth: tls.RequireAndVerifyClientCert}
		}},
		{"incomplete server TLS keypair", func(c *Config) { c.Server.TLSCertFile = "server.crt" }},
		{"HTTP JWKS endpoint", func(c *Config) { c.Server.OAuth.JWKSURL = "http://issuer.example.test/jwks" }},
		{"unsafe algorithm", func(c *Config) { c.Server.OAuth.Algorithms = []string{"none"} }},
		{"missing required scopes", func(c *Config) { c.Server.OAuth.Scopes = nil }},
		{"root metadata route", func(c *Config) { c.Server.OAuth.MetadataURL = "https://mcp.example.test/" }},
		{"excessive cache TTL", func(c *Config) { c.Server.OAuth.CacheTTL = Duration(time.Hour + time.Second) }},
		{"excessive clock skew", func(c *Config) { c.Server.OAuth.ClockSkew = Duration(3 * time.Minute) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := DefaultHTTPConfig()
			config.Server.OAuth = validOAuthConfig()
			test.change(config)
			if err := config.Validate(); err == nil {
				t.Fatal("invalid OAuth configuration was accepted")
			}
		})
	}
}

func TestOAuthProtectsMCPOverTLSAndLeavesMetadataPublic(t *testing.T) {
	issuer := newOAuthIntegrationIssuer(t)
	config := DefaultHTTPConfig()
	config.Server.Audit.Disabled = true
	config.Server.OAuth = validOAuthConfig()
	config.Server.OAuth.JWKSURL = issuer.server.URL
	config.Server.OAuth.HTTPClient = issuer.server.Client()

	mcpServer, err := newMCPServer(config, &testBackend{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	handler := newMCPStreamableHTTPHandler(mcpServer.(*simpleMCPServer).server, config)
	handler, err = secureHTTPHandler(handler, config)
	if err != nil {
		t.Fatal(err)
	}
	logBuffer := &bytes.Buffer{}
	logger := logrus.New()
	logger.SetOutput(logBuffer)
	server := httptest.NewTLSServer(loggingHandler(handler, logger))
	defer server.Close()
	if !strings.HasPrefix(server.URL, "https://") {
		t.Fatalf("test endpoint is not TLS protected: %q", server.URL)
	}
	client := server.Client()

	metadataResponse, err := client.Get(server.URL + "/.well-known/oauth-protected-resource")
	if err != nil {
		t.Fatal(err)
	}
	defer metadataResponse.Body.Close()
	if metadataResponse.StatusCode != http.StatusOK {
		t.Fatalf("public metadata status = %d, want %d", metadataResponse.StatusCode, http.StatusOK)
	}
	var metadata struct {
		Resource string `json:"resource"`
	}
	if err := json.NewDecoder(metadataResponse.Body).Decode(&metadata); err != nil {
		t.Fatal(err)
	}
	if metadata.Resource != oauthAudience {
		t.Fatalf("metadata resource = %q, want %q", metadata.Resource, oauthAudience)
	}

	validClaims := oauthTestClaims()
	validToken := issuer.token(t, validClaims)
	_, wrongSigner, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	badSignature := jwt.NewWithClaims(jwt.SigningMethodEdDSA, validClaims)
	badSignature.Header["kid"] = oauthKeyID
	badSignatureToken, err := badSignature.SignedString(wrongSigner)
	if err != nil {
		t.Fatal(err)
	}
	unknownKey := jwt.NewWithClaims(jwt.SigningMethodEdDSA, validClaims)
	unknownKey.Header["kid"] = "not-in-jwks"
	unknownKeyToken, err := unknownKey.SignedString(issuer.private)
	if err != nil {
		t.Fatal(err)
	}
	wrongAlgorithm, err := jwt.NewWithClaims(jwt.SigningMethodHS256, validClaims).SignedString([]byte("not-a-trusted-key"))
	if err != nil {
		t.Fatal(err)
	}
	rejections := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "missing credentials", wantStatus: http.StatusUnauthorized},
		{name: "bad signature", token: badSignatureToken, wantStatus: http.StatusUnauthorized},
		{name: "wrong issuer", token: issuer.token(t, claimsWith(validClaims, "iss", "https://wrong.example.test/")), wantStatus: http.StatusUnauthorized},
		{name: "wrong audience", token: issuer.token(t, claimsWith(validClaims, "aud", "https://wrong.example.test")), wantStatus: http.StatusUnauthorized},
		{name: "expired", token: issuer.token(t, claimsWith(validClaims, "exp", time.Now().Add(-time.Minute).Unix())), wantStatus: http.StatusUnauthorized},
		{name: "missing expiration", token: issuer.token(t, claimsWithout(validClaims, "exp")), wantStatus: http.StatusUnauthorized},
		{name: "unknown key", token: unknownKeyToken, wantStatus: http.StatusUnauthorized},
		{name: "unsafe algorithm", token: wrongAlgorithm, wantStatus: http.StatusUnauthorized},
		{name: "insufficient scope", token: issuer.token(t, claimsWith(validClaims, "scope", "tools:read")), wantStatus: http.StatusForbidden},
	}
	for _, test := range rejections {
		t.Run(test.name, func(t *testing.T) {
			response := postMCPInitialize(t, client, server.URL, test.token)
			defer response.Body.Close()
			assertOAuthRejection(t, response, test.name, test.wantStatus)
		})
	}

	response := postMCPInitialize(t, client, server.URL, validToken)
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("valid OAuth initialize status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if strings.Contains(logBuffer.String(), validToken) || strings.Contains(logBuffer.String(), "invalid token") ||
		strings.Contains(logBuffer.String(), "JWKS unavailable") {
		t.Fatalf("HTTP logs exposed credential or verifier details: %s", logBuffer.String())
	}
}

func TestOAuthRejectionDrainsRequestBodyBeforeWritingResponse(t *testing.T) {
	config := DefaultHTTPConfig()
	config.Server.OAuth = validOAuthConfig()
	nextCalled := false
	handler, err := secureOAuthHTTPHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	}), config)
	if err != nil {
		t.Fatal(err)
	}

	body := strings.NewReader(`{"jsonrpc":"2.0"}`)
	request := httptest.NewRequest(http.MethodPost, oauthAudience, nil)
	request.Body = io.NopCloser(body)
	response := &oauthBodyCheckingRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		body:             body,
	}
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("missing-token status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if nextCalled {
		t.Fatal("unauthenticated request reached the MCP handler")
	}
	if !response.bodyDrainedBeforeError {
		t.Fatal("request body was not drained before the unauthorized response")
	}
}

type oauthBodyCheckingRecorder struct {
	*httptest.ResponseRecorder
	body                   *strings.Reader
	bodyDrainedBeforeError bool
}

func (r *oauthBodyCheckingRecorder) WriteHeader(statusCode int) {
	if statusCode >= http.StatusBadRequest {
		r.bodyDrainedBeforeError = r.body.Len() == 0
	}
	r.ResponseRecorder.WriteHeader(statusCode)
}

func assertOAuthRejection(t *testing.T, response *http.Response, name string, wantStatus int) {
	t.Helper()
	if response.StatusCode != wantStatus {
		t.Fatalf("status = %d, want %d", response.StatusCode, wantStatus)
	}
	if name == "missing credentials" {
		challenge := response.Header.Get("WWW-Authenticate")
		if !strings.Contains(challenge, oauthMetadataURL) || !strings.Contains(challenge, "tools:read") {
			t.Fatalf("missing protected-resource challenge parameters: %q", challenge)
		}
	}
}

func TestOAuthDoesNotChangeLegacyBearerOrStdio(t *testing.T) {
	t.Setenv("STACKQL_MCP_OAUTH_LEGACY_TOKEN", "legacy-token")
	config := DefaultHTTPConfig()
	config.Server.AuthTokenEnvVar = "STACKQL_MCP_OAUTH_LEGACY_TOKEN"
	server := mcp.NewServer(&mcp.Implementation{Name: "legacy-auth", Version: "1"}, nil)
	handler, err := secureHTTPHandler(newMCPStreamableHTTPHandler(server, config), config)
	if err != nil {
		t.Fatal(err)
	}
	testServer := httptest.NewServer(handler)
	defer testServer.Close()
	client := testServer.Client()
	if response := postMCPInitialize(t, client, testServer.URL, ""); response.StatusCode != http.StatusUnauthorized {
		_ = response.Body.Close()
		t.Fatalf("legacy missing-token status = %d, want %d", response.StatusCode, http.StatusUnauthorized)
	} else {
		_ = response.Body.Close()
	}
	response := postMCPInitialize(t, client, testServer.URL, "legacy-token")
	if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()
		t.Fatalf("legacy bearer status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	_ = response.Body.Close()

	session := connectInProcess(t, DefaultConfig(), &testBackend{})
	if err := session.Close(); err != nil {
		t.Fatalf("stdio session close: %v", err)
	}
}

func validOAuthConfig() *OAuthResourceServerConfig {
	return &OAuthResourceServerConfig{
		Issuer:      oauthIssuerURL,
		Audience:    oauthAudience,
		Resource:    oauthAudience,
		JWKSURL:     "https://issuer.example.test/jwks",
		MetadataURL: oauthMetadataURL,
		Scopes:      []string{"tools:read", "tools:write"},
		Algorithms:  []string{"EdDSA"},
	}
}

func oauthTestClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"iss":   oauthIssuerURL,
		"aud":   oauthAudience,
		"exp":   time.Now().Add(time.Minute).Unix(),
		"scope": "tools:read tools:write",
		"sub":   "test-client",
	}
}

func postMCPInitialize(t *testing.T, client *http.Client, endpoint, bearer string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"oauth-test","version":"1"}}}`,
	))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func claimsWith(claims jwt.MapClaims, name string, value any) jwt.MapClaims {
	cloned := make(jwt.MapClaims, len(claims))
	for key, existing := range claims {
		cloned[key] = existing
	}
	cloned[name] = value
	return cloned
}

func claimsWithout(claims jwt.MapClaims, name string) jwt.MapClaims {
	cloned := make(jwt.MapClaims, len(claims))
	for key, existing := range claims {
		if key != name {
			cloned[key] = existing
		}
	}
	return cloned
}
