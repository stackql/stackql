package oauthrs //nolint:testpackage // tests private token verification and JWKS cache behavior

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v5"
)

type testJWKS struct {
	mu       sync.Mutex
	keys     []jose.JSONWebKey
	requests int
}

func (j *testJWKS) serve(w http.ResponseWriter, _ *http.Request) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.requests++
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: j.keys})
}

func (j *testJWKS) replace(keys ...jose.JSONWebKey) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.keys = keys
}

func (j *testJWKS) requestCount() int {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.requests
}

func TestMiddlewareVerifiesClaimsAndScopes(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	jwks := &testJWKS{keys: []jose.JSONWebKey{{
		Key: pub, KeyID: "test-key", Algorithm: "EdDSA", Use: "sig",
	}}}
	jwksServer := httptest.NewTLSServer(http.HandlerFunc(jwks.serve))
	defer jwksServer.Close()

	rs, err := NewResourceServer(testConfig(jwksServer, jwksServer.Client()))
	if err != nil {
		t.Fatal(err)
	}
	handler := rs.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	validClaims := jwt.MapClaims{
		"iss":   "https://issuer.example.test/",
		"aud":   "https://mcp.example.test/api",
		"exp":   time.Now().Add(time.Minute).Unix(),
		"scope": "tools:read tools:write",
		"sub":   "service-account",
	}
	tests := []struct {
		name       string
		claims     jwt.MapClaims
		wantStatus int
	}{
		{name: "valid", claims: validClaims, wantStatus: http.StatusNoContent},
		{name: "wrong issuer", claims: claimsWith(validClaims, "iss", "https://attacker.example.test"), wantStatus: http.StatusUnauthorized},
		{name: "wrong audience", claims: claimsWith(validClaims, "aud", "https://other.example.test"), wantStatus: http.StatusUnauthorized},
		{name: "missing expiration", claims: claimsWithout(validClaims, "exp"), wantStatus: http.StatusUnauthorized},
		{name: "expired", claims: claimsWith(validClaims, "exp", time.Now().Add(-time.Minute).Unix()), wantStatus: http.StatusUnauthorized},
		{name: "insufficient scope", claims: claimsWith(validClaims, "scope", "tools:read"), wantStatus: http.StatusForbidden},
		{name: "malformed scope", claims: claimsWith(validClaims, "scope", 42), wantStatus: http.StatusUnauthorized},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token := signToken(t, priv, "test-key", test.claims)
			request := httptest.NewRequest(http.MethodGet, "/mcp", nil)
			request.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d, body: %s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
	_, otherPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	for name, token := range map[string]string{
		"bad signature": signToken(t, otherPriv, "test-key", validClaims),
		"unknown key":   signToken(t, priv, "unknown-key", validClaims),
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/mcp", nil)
			request.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
			}
		})
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/mcp", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("missing credential status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	challenge := response.Header().Get("WWW-Authenticate")
	if !strings.Contains(challenge, `resource_metadata="https://mcp.example.test/.well-known/oauth-protected-resource"`) ||
		!strings.Contains(challenge, `scope="tools:read tools:write"`) {
		t.Fatalf("unexpected bearer challenge: %q", challenge)
	}
}

func TestUnknownKeyRefreshesJWKSForRotation(t *testing.T) {
	pub1, priv1, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pub2, priv2, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	jwks := &testJWKS{keys: []jose.JSONWebKey{{Key: pub1, KeyID: "old", Algorithm: "EdDSA", Use: "sig"}}}
	server := httptest.NewTLSServer(http.HandlerFunc(jwks.serve))
	defer server.Close()
	rs, err := NewResourceServer(testConfig(server, server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	handler := rs.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	claims := jwt.MapClaims{
		"iss": "https://issuer.example.test/", "aud": "https://mcp.example.test/api",
		"exp": time.Now().Add(time.Minute).Unix(), "scope": "tools:read tools:write",
	}
	call := func(token string) int {
		request := httptest.NewRequest(http.MethodGet, "/mcp", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response.Code
	}
	if status := call(signToken(t, priv1, "old", claims)); status != http.StatusNoContent {
		t.Fatalf("initial key status = %d", status)
	}
	jwks.replace(jose.JSONWebKey{Key: pub2, KeyID: "new", Algorithm: "EdDSA", Use: "sig"})
	rs.keysMu.Lock()
	rs.lastFetch = time.Now().Add(-2 * refreshCooldown)
	rs.keysMu.Unlock()
	if status := call(signToken(t, priv2, "new", claims)); status != http.StatusNoContent {
		t.Fatalf("rotated key status = %d, want %d", status, http.StatusNoContent)
	}
	if got := jwks.requestCount(); got != 2 {
		t.Fatalf("JWKS requests = %d, want initial fetch plus rotation refresh", got)
	}
}

func TestUnavailableJWKSFailsClosedAndBoundsRetries(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	rs, err := NewResourceServer(testConfig(server, server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	handler := rs.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("request reached protected handler without verification")
	}))
	token := signToken(t, priv, "not-present", jwt.MapClaims{
		"iss": "https://issuer.example.test/", "aud": "https://mcp.example.test/api",
		"exp": time.Now().Add(time.Minute).Unix(), "scope": "tools:read tools:write",
	})
	for range 2 {
		request := httptest.NewRequest(http.MethodGet, "/mcp", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("JWKS service failure status = %d, want %d", response.Code, http.StatusInternalServerError)
		}
		if !strings.Contains(response.Body.String(), "JWKS unavailable") {
			t.Fatalf("JWKS service failure was not surfaced: %q", response.Body.String())
		}
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("JWKS requests = %d, want one request within the refresh cooldown", got)
	}
}

func TestJWKSRedirectIsNotFollowed(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	var redirectedRequests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/keys", http.StatusFound)
			return
		}
		redirectedRequests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
			Key: pub, KeyID: "test-key", Algorithm: "EdDSA", Use: "sig",
		}}})
	}))
	defer server.Close()
	config := testConfig(server, server.Client())
	config.JWKSURL = server.URL + "/redirect"
	rs, err := NewResourceServer(config)
	if err != nil {
		t.Fatal(err)
	}
	handler := rs.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	token := signToken(t, priv, "test-key", jwt.MapClaims{
		"iss": "https://issuer.example.test/", "aud": "https://mcp.example.test/api",
		"exp": time.Now().Add(time.Minute).Unix(), "scope": "tools:read tools:write",
	})
	request := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("JWKS redirect status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if got := redirectedRequests.Load(); got != 0 {
		t.Fatalf("redirect target requests = %d, want none", got)
	}
}

func TestRSAModulusBounds(t *testing.T) {
	tests := []struct {
		bits int
		want bool
	}{
		{bits: 2047, want: false},
		{bits: 2048, want: true},
		{bits: maximumRSAModulusBits, want: true},
		{bits: maximumRSAModulusBits + 1, want: false},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%d_bits", test.bits), func(t *testing.T) {
			modulus := new(big.Int).Lsh(big.NewInt(1), uint(test.bits))
			modulus.Sub(modulus, big.NewInt(1))
			key := &rsa.PublicKey{N: modulus, E: 65537}
			if got := keySupportsAlgorithm(key, "RS256"); got != test.want {
				t.Fatalf("keySupportsAlgorithm = %v, want %v", got, test.want)
			}
		})
	}
}

func TestProtectedResourceMetadataAndConfigurationBounds(t *testing.T) {
	server := httptest.NewTLSServer(http.NotFoundHandler())
	defer server.Close()
	rs, err := NewResourceServer(testConfig(server, server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	rs.ProtectedResourceMetadataHandler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metadata", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("metadata status = %d", response.Code)
	}
	var metadata struct {
		Resource             string   `json:"resource"`
		AuthorizationServers []string `json:"authorization_servers"`
		ScopesSupported      []string `json:"scopes_supported"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata.Resource != "https://mcp.example.test/api" ||
		len(metadata.AuthorizationServers) != 1 || metadata.AuthorizationServers[0] != "https://issuer.example.test/" ||
		len(metadata.ScopesSupported) != 2 {
		t.Fatalf("unexpected protected-resource metadata: %+v", metadata)
	}

	config := testConfig(server, server.Client())
	config.JWKSURL = "http://issuer.example.test/jwks"
	if _, err := NewResourceServer(config); err == nil {
		t.Fatal("expected insecure JWKS URL to be rejected")
	}
	config = testConfig(server, server.Client())
	config.Algorithms = []string{"none"}
	if _, err := NewResourceServer(config); err == nil {
		t.Fatal("expected unsafe algorithm to be rejected")
	}
	config = testConfig(server, server.Client())
	config.Scopes = nil
	if _, err := NewResourceServer(config); err == nil {
		t.Fatal("expected empty required scopes to be rejected")
	}
	config = testConfig(server, server.Client())
	config.CacheTTL = maximumCacheTTL + time.Second
	if _, err := NewResourceServer(config); err == nil {
		t.Fatal("expected unbounded cache TTL to be rejected")
	}
}

func testConfig(server *httptest.Server, client *http.Client) Config {
	return Config{
		Issuer:      "https://issuer.example.test/",
		Audience:    "https://mcp.example.test/api",
		Resource:    "https://mcp.example.test/api",
		JWKSURL:     server.URL,
		MetadataURL: "https://mcp.example.test/.well-known/oauth-protected-resource",
		Scopes:      []string{"tools:read", "tools:write"},
		Algorithms:  []string{"EdDSA"},
		HTTPClient:  client,
	}
}

func signToken(t *testing.T, key ed25519.PrivateKey, kid string, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["kid"] = kid
	raw, err := token.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return raw
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
