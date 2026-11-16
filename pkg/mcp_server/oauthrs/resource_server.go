// Package oauthrs provides an opt-in OAuth JWT resource-server boundary for
// MCP HTTP handlers. It verifies locally signed access tokens using a
// preconfigured issuer and JWKS endpoint; it does not acquire tokens, perform
// introspection, or provide revocation, DPoP, or mTLS-bound token checks.
// Callers remain responsible for serving the protected endpoint over HTTPS.
package oauthrs

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v5"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

const (
	defaultCacheTTL       = 5 * time.Minute
	minimumCacheTTL       = 10 * time.Second
	maximumCacheTTL       = time.Hour
	maximumClockSkew      = 2 * time.Minute
	maximumJWKSBytes      = 1 << 20
	maximumJWKSKeys       = 64
	maximumTokenSize      = 16 << 10
	maximumRSAModulusBits = 4096
	jwksTimeout           = 5 * time.Second
	refreshCooldown       = time.Second
)

var safeAlgorithms = []string{
	jwt.SigningMethodRS256.Alg(),
	jwt.SigningMethodRS384.Alg(),
	jwt.SigningMethodRS512.Alg(),
	jwt.SigningMethodPS256.Alg(),
	jwt.SigningMethodPS384.Alg(),
	jwt.SigningMethodPS512.Alg(),
	jwt.SigningMethodES256.Alg(),
	jwt.SigningMethodES384.Alg(),
	jwt.SigningMethodES512.Alg(),
	jwt.SigningMethodEdDSA.Alg(),
}

// ErrJWKSUnavailable identifies a failure to obtain or use the configured
// JWKS document. The MCP SDK middleware maps it to a server error, not a 401.
var ErrJWKSUnavailable = errors.New("JWKS unavailable")

// Config contains trusted resource-server settings. Issuer, Audience, Resource,
// JWKSURL, MetadataURL, Scopes, and Algorithms must be configured explicitly.
// Only HTTPS URLs are accepted. HTTPClient may provide a custom trust store for
// private authorization servers, but redirects are always disabled.
type Config struct {
	Issuer      string
	Audience    string
	Resource    string
	JWKSURL     string
	MetadataURL string
	Scopes      []string
	Algorithms  []string

	// CacheTTL defaults to five minutes and, when set, must be between
	// ten seconds and one hour.
	CacheTTL time.Duration
	// ClockSkew defaults to zero and cannot exceed two minutes.
	ClockSkew  time.Duration
	HTTPClient *http.Client
}

// ResourceServer holds immutable verification settings and a synchronized
// JWKS cache. Construct it with New before publishing its handlers.
type ResourceServer struct {
	issuer      string
	audience    string
	resource    string
	jwksURL     string
	metadataURL string
	scopes      []string
	algorithms  []string
	cacheTTL    time.Duration
	clockSkew   time.Duration
	parser      *jwt.Parser
	client      *http.Client

	keysMu    sync.Mutex
	keys      map[string]verifiedKey
	loadedAt  time.Time
	lastFetch time.Time
}

type verifiedKey struct {
	key       crypto.PublicKey
	algorithm string
}

// NewResourceServer validates configuration and returns a fail-closed
// resource-server verifier.
func NewResourceServer(config Config) (*ResourceServer, error) {
	for label, value := range map[string]string{
		"issuer": config.Issuer, "audience": config.Audience, "resource": config.Resource,
		"JWKS URL": config.JWKSURL, "metadata URL": config.MetadataURL,
	} {
		if value == "" {
			return nil, fmt.Errorf("%s is required", label)
		}
	}
	for label, value := range map[string]string{
		"issuer": config.Issuer, "resource": config.Resource,
		"JWKS URL": config.JWKSURL, "metadata URL": config.MetadataURL,
	} {
		if err := validateHTTPSURL(value); err != nil {
			return nil, fmt.Errorf("invalid %s: %w", label, err)
		}
	}
	if strings.TrimSpace(config.Audience) != config.Audience {
		return nil, errors.New("audience must not contain surrounding whitespace")
	}
	if err := validateTokenPolicy(config.Scopes, config.Algorithms); err != nil {
		return nil, err
	}
	config.Scopes = slices.Clone(config.Scopes)
	slices.Sort(config.Scopes)
	config.Scopes = slices.Compact(config.Scopes)

	config.Algorithms = slices.Clone(config.Algorithms)
	slices.Sort(config.Algorithms)
	config.Algorithms = slices.Compact(config.Algorithms)

	if config.CacheTTL == 0 {
		config.CacheTTL = defaultCacheTTL
	} else if config.CacheTTL < minimumCacheTTL || config.CacheTTL > maximumCacheTTL {
		return nil, fmt.Errorf("JWKS cache TTL must be between %s and %s", minimumCacheTTL, maximumCacheTTL)
	}
	if config.ClockSkew < 0 || config.ClockSkew > maximumClockSkew {
		return nil, fmt.Errorf("clock skew must be between zero and %s", maximumClockSkew)
	}

	client := &http.Client{}
	if config.HTTPClient != nil {
		*client = *config.HTTPClient
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	return &ResourceServer{
		issuer:      config.Issuer,
		audience:    config.Audience,
		resource:    config.Resource,
		jwksURL:     config.JWKSURL,
		metadataURL: config.MetadataURL,
		scopes:      config.Scopes,
		algorithms:  config.Algorithms,
		cacheTTL:    config.CacheTTL,
		clockSkew:   config.ClockSkew,
		parser: jwt.NewParser(
			jwt.WithValidMethods(config.Algorithms),
			jwt.WithIssuer(config.Issuer),
			jwt.WithAudience(config.Audience),
			jwt.WithExpirationRequired(),
			jwt.WithLeeway(config.ClockSkew),
		),
		client: client,
	}, nil
}

func validateTokenPolicy(scopes, algorithms []string) error {
	if len(scopes) == 0 {
		return errors.New("at least one required scope is required")
	}
	for _, scope := range scopes {
		if !validScope(scope) {
			return fmt.Errorf("invalid required scope %q", scope)
		}
	}
	if len(algorithms) == 0 {
		return errors.New("at least one JWT signing algorithm is required")
	}
	for _, algorithm := range algorithms {
		if !slices.Contains(safeAlgorithms, algorithm) {
			return fmt.Errorf("unsupported or unsafe JWT algorithm %q", algorithm)
		}
	}
	return nil
}

// Middleware verifies every bearer token and applies the configured scope and
// expiration requirements using the MCP SDK's OAuth middleware.
func (rs *ResourceServer) Middleware(next http.Handler) http.Handler {
	return auth.RequireBearerToken(rs.verifyToken, &auth.RequireBearerTokenOptions{
		ResourceMetadataURL: rs.metadataURL,
		Scopes:              slices.Clone(rs.scopes),
		ClockSkew:           rs.clockSkew,
	})(next)
}

// ProtectedResourceMetadataHandler serves RFC 9728 metadata for this resource.
// Mount it at the resource metadata URL configured in Config.
func (rs *ResourceServer) ProtectedResourceMetadataHandler() http.Handler {
	return auth.ProtectedResourceMetadataHandler(&oauthex.ProtectedResourceMetadata{
		Resource:               rs.resource,
		AuthorizationServers:   []string{rs.issuer},
		ScopesSupported:        slices.Clone(rs.scopes),
		BearerMethodsSupported: []string{"header"},
	})
}

func (rs *ResourceServer) verifyToken(ctx context.Context, raw string, _ *http.Request) (*auth.TokenInfo, error) {
	if len(raw) == 0 || len(raw) > maximumTokenSize {
		return nil, auth.ErrInvalidToken
	}

	jwksUnavailable := false
	token, err := rs.parser.Parse(raw, func(token *jwt.Token) (any, error) {
		key, keyErr := rs.tokenKey(ctx, token)
		if keyErr != nil {
			if errors.Is(keyErr, ErrJWKSUnavailable) {
				jwksUnavailable = true
			}
			return nil, keyErr
		}
		return key, nil
	})
	if err != nil || token == nil || !token.Valid {
		if jwksUnavailable {
			return nil, ErrJWKSUnavailable
		}
		return nil, auth.ErrInvalidToken
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, auth.ErrInvalidToken
	}
	expiration, err := claims.GetExpirationTime()
	if err != nil || expiration == nil {
		return nil, auth.ErrInvalidToken
	}
	scopes, err := tokenScopes(claims)
	if err != nil {
		return nil, auth.ErrInvalidToken
	}
	subject, _ := claims["sub"].(string)
	return &auth.TokenInfo{
		Scopes:     scopes,
		Expiration: expiration.Time,
		UserID:     subject,
	}, nil
}

func (rs *ResourceServer) tokenKey(ctx context.Context, token *jwt.Token) (crypto.PublicKey, error) {
	for _, name := range []string{"crit", "b64", "jku", "x5u", "jwk", "x5c"} {
		if _, present := token.Header[name]; present {
			return nil, auth.ErrInvalidToken
		}
	}
	kid, ok := token.Header["kid"].(string)
	if !ok || kid == "" || len(kid) > 128 {
		return nil, auth.ErrInvalidToken
	}
	return rs.keyFor(ctx, kid, token.Method.Alg())
}

func tokenScopes(claims jwt.MapClaims) ([]string, error) {
	var scopes []string
	if raw, present := claims["scope"]; present {
		scope, ok := raw.(string)
		if !ok {
			return nil, errors.New("invalid scope claim")
		}
		scopes = append(scopes, strings.Fields(scope)...)
	}
	if raw, present := claims["scp"]; present {
		values, ok := raw.([]any)
		if !ok {
			return nil, errors.New("invalid scp claim")
		}
		for _, value := range values {
			scope, isString := value.(string)
			if !isString || !validScope(scope) {
				return nil, errors.New("invalid scp claim")
			}
			scopes = append(scopes, scope)
		}
	}
	return slices.Compact(slices.Sorted(slices.Values(scopes))), nil
}

func (rs *ResourceServer) keyFor(ctx context.Context, kid, algorithm string) (crypto.PublicKey, error) {
	rs.keysMu.Lock()
	defer rs.keysMu.Unlock()

	now := time.Now()
	if rs.keys == nil || !now.Before(rs.loadedAt.Add(rs.cacheTTL)) {
		if !rs.lastFetch.IsZero() && now.Sub(rs.lastFetch) < refreshCooldown {
			return nil, ErrJWKSUnavailable
		}
		if err := rs.refreshKeysLocked(ctx, now); err != nil {
			return nil, ErrJWKSUnavailable
		}
	} else if _, found := rs.keys[kid]; !found && now.Sub(rs.lastFetch) >= refreshCooldown {
		if err := rs.refreshKeysLocked(ctx, now); err != nil {
			return nil, ErrJWKSUnavailable
		}
	}
	key, found := rs.keys[kid]
	if !found || (key.algorithm != "" && key.algorithm != algorithm) ||
		!keySupportsAlgorithm(key.key, algorithm) {
		return nil, errors.New("no trusted signing key")
	}
	return key.key, nil
}

func (rs *ResourceServer) refreshKeysLocked(ctx context.Context, now time.Time) error {
	rs.lastFetch = now
	fetchCtx, cancel := context.WithTimeout(ctx, jwksTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(fetchCtx, http.MethodGet, rs.jwksURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json, application/jwk-set+json")
	resp, err := rs.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint returned HTTP %d", resp.StatusCode)
	}
	if contentType := resp.Header.Get("Content-Type"); contentType != "" {
		mediaType, _, parseErr := mime.ParseMediaType(contentType)
		if parseErr != nil || (mediaType != "application/json" && mediaType != "application/jwk-set+json") {
			return errors.New("JWKS endpoint returned an unsupported content type")
		}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maximumJWKSBytes+1))
	if err != nil || len(data) > maximumJWKSBytes {
		return errors.New("JWKS response is unreadable or exceeds the size limit")
	}
	var set jose.JSONWebKeySet
	if decodeErr := json.Unmarshal(data, &set); decodeErr != nil || len(set.Keys) == 0 || len(set.Keys) > maximumJWKSKeys {
		return errors.New("JWKS response is invalid or exceeds the key limit")
	}

	keys := make(map[string]verifiedKey)
	for _, jwk := range set.Keys {
		key, usable := rs.usableSigningKey(jwk)
		if !usable {
			continue
		}
		if _, duplicate := keys[jwk.KeyID]; duplicate {
			return errors.New("JWKS contains duplicate usable key identifiers")
		}
		keys[jwk.KeyID] = key
	}
	if len(keys) == 0 {
		return errors.New("JWKS contains no usable signing keys")
	}
	rs.keys = keys
	rs.loadedAt = now
	return nil
}

func (rs *ResourceServer) usableSigningKey(jwk jose.JSONWebKey) (verifiedKey, bool) {
	if jwk.KeyID == "" || (jwk.Use != "" && jwk.Use != "sig") ||
		(jwk.Algorithm != "" && !slices.Contains(rs.algorithms, jwk.Algorithm)) ||
		jwk.CertificatesURL != nil || !jwk.IsPublic() || !jwk.Valid() {
		return verifiedKey{}, false
	}
	key, ok := jwk.Key.(crypto.PublicKey)
	if !ok || !slices.Contains(rs.algorithms, algorithmForKey(key, rs.algorithms)) {
		return verifiedKey{}, false
	}
	if jwk.Algorithm != "" && !keySupportsAlgorithm(key, jwk.Algorithm) {
		return verifiedKey{}, false
	}
	return verifiedKey{key: key, algorithm: jwk.Algorithm}, true
}

func algorithmForKey(key crypto.PublicKey, allowed []string) string {
	for _, algorithm := range allowed {
		if keySupportsAlgorithm(key, algorithm) {
			return algorithm
		}
	}
	return ""
}

func keySupportsAlgorithm(key crypto.PublicKey, algorithm string) bool {
	switch k := key.(type) {
	case *rsa.PublicKey:
		return k.N != nil && k.N.BitLen() >= 2048 && k.N.BitLen() <= maximumRSAModulusBits &&
			k.N.Bit(0) == 1 &&
			k.E >= 3 && k.E&1 == 1 &&
			(strings.HasPrefix(algorithm, "RS") || strings.HasPrefix(algorithm, "PS"))
	case *ecdsa.PublicKey:
		if k.Curve == nil || k.X == nil || k.Y == nil || !k.Curve.IsOnCurve(k.X, k.Y) {
			return false
		}
		switch algorithm {
		case "ES256":
			return k.Curve == elliptic.P256()
		case "ES384":
			return k.Curve == elliptic.P384()
		case "ES512":
			return k.Curve == elliptic.P521()
		}
	case ed25519.PublicKey:
		return algorithm == jwt.SigningMethodEdDSA.Alg() && len(k) == ed25519.PublicKeySize
	}
	return false
}

func validateHTTPSURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" ||
		parsed.User != nil || parsed.Opaque != "" || parsed.RawQuery != "" ||
		parsed.Fragment != "" || strings.ContainsAny(raw, " \t\r\n") {
		return errors.New("must be an absolute HTTPS URL without user info, query, or fragment")
	}
	return nil
}

func validScope(scope string) bool {
	if scope == "" {
		return false
	}
	for _, r := range scope {
		if r < 0x21 || r > 0x7e || r == '"' || r == '\\' {
			return false
		}
	}
	return true
}
