package accessmanagement

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stackql-labs/omnisdk/public/authentication"
	"github.com/stackql-labs/omnisdk/public/authorization"
	"github.com/stackql-labs/omnisdk/public/principal"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
)

// Config is host-owned configuration, never accepted from SQL or provider predicates.
type Config struct {
	Type                   string               `json:"type"`
	Tokens                 []BearerToken        `json:"tokens"`
	ClientCAPEM            string               `json:"client_ca_pem"`
	CertificateIdentity    string               `json:"certificate_identity"`
	Issuer                 string               `json:"issuer"`
	Audience               string               `json:"audience"`
	Keys                   []VerificationKey    `json:"keys"`
	Rules                  []authorization.Rule `json:"rules"`
	EnforceSessionDeadline bool                 `json:"enforce_session_deadline"`
	MaxSessionSeconds      int                  `json:"max_session_seconds"`
}
type VerificationKey struct {
	ID        string `json:"kid"`
	Algorithm string `json:"algorithm"`
	PEM       string `json:"public_key_pem"`
}

// BearerToken stores only the digest of a provisioned credential.
type BearerToken struct {
	Subject string `json:"subject"`
	SHA256  string `json:"sha256"`
}

// Manager establishes a fresh directory snapshot for every statement.
type Manager interface {
	Mode() string
	TLSConfig() *tls.Config
	Establish(context.Context) (authorization.Session, context.Context, context.CancelFunc, error)
}
type manager struct {
	mode            string
	tlsConfig       *tls.Config
	authenticator   authentication.Authenticator
	directory       principal.Directory
	authorizer      authorization.Authorizer
	lifetime        time.Duration
	enforceDeadline bool
}

const defaultSessionLifetime = 5 * time.Minute

type evidenceKey struct{}

// WithEvidence marks a transport boundary, including when no credential was supplied.
// Remote transports must never fall back to the server process's environment token.
func WithEvidence(ctx context.Context, evidence authentication.Evidence) context.Context {
	return context.WithValue(ctx, evidenceKey{}, evidence)
}

//nolint:nilnil // No configuration explicitly disables caller protection.
func NewManager(raw string, engine sqlengine.SQLEngine) (Manager, error) {
	if raw == "" {
		return nil, nil
	}
	path, err := configurationPath(raw)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.New("cannot read caller access configuration")
	}
	var cfg Config
	if err = decodeStrict(data, &cfg); err != nil {
		return nil, err
	}

	directory, err := NewDirectory(engine)
	if err != nil {
		return nil, err
	}
	return newManager(cfg, directory)
}

func configurationPath(raw string) (string, error) {
	if !strings.HasPrefix(strings.TrimSpace(raw), "{") {
		return raw, nil
	}
	var source struct {
		Path string `json:"path"`
	}
	if err := decodeStrict([]byte(raw), &source); err != nil {
		return "", err
	}
	if strings.TrimSpace(source.Path) == "" {
		return "", errors.New("caller access configuration requires a nonempty path")
	}
	return source.Path, nil
}

func decodeStrict(data []byte, target any) error {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return errors.New("invalid caller access configuration")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("invalid trailing caller access configuration")
	}
	return nil
}

func newManager(cfg Config, directory principal.Directory) (Manager, error) {
	if len(cfg.Rules) == 0 || cfg.MaxSessionSeconds < 0 || cfg.MaxSessionSeconds > 86400 {
		return nil, errors.New("caller access requires policy and a valid session lifetime")
	}
	if cfg.Type == "" {
		cfg.Type = "jwt"
	}
	verifier, transport, err := configuredAuthenticator(cfg)
	if err != nil {
		return nil, err
	}
	policy, err := authorization.NewCasbin(cfg.Rules)
	if err != nil {
		return nil, err
	}
	lifetime := time.Duration(cfg.MaxSessionSeconds) * time.Second
	if lifetime == 0 {
		lifetime = defaultSessionLifetime
	}
	return &manager{
		mode: cfg.Type, tlsConfig: transport, authenticator: verifier,
		directory: directory, authorizer: policy, lifetime: lifetime, enforceDeadline: cfg.EnforceSessionDeadline,
	}, nil
}

func configuredAuthenticator(cfg Config) (authentication.Authenticator, *tls.Config, error) {
	switch cfg.Type {
	case "jwt":
		if len(cfg.Tokens) != 0 || cfg.ClientCAPEM != "" || cfg.CertificateIdentity != "" {
			return nil, nil, errors.New("JWT configuration contains another authentication mode")
		}
		verifier, err := jwtAuthenticator(cfg)
		return verifier, nil, err
	case "bearer":
		if len(cfg.Keys) != 0 || cfg.Audience != "" || cfg.ClientCAPEM != "" || cfg.CertificateIdentity != "" {
			return nil, nil, errors.New("bearer configuration contains another authentication mode")
		}
		tokens := make([]authentication.Token, len(cfg.Tokens))
		for i, token := range cfg.Tokens {
			digest, err := hex.DecodeString(token.SHA256)
			if err != nil {
				return nil, nil, errors.New("invalid bearer SHA-256 digest")
			}
			tokens[i] = authentication.Token{Subject: token.Subject, Digest: digest}
		}
		verifier, err := authentication.NewBearer(cfg.Issuer, tokens)
		return verifier, nil, err
	case "mtls":
		if len(cfg.Keys) != 0 || len(cfg.Tokens) != 0 || cfg.Audience != "" {
			return nil, nil, errors.New("mTLS configuration contains another authentication mode")
		}
		return mtlsAuthenticator(cfg)
	case "local_os":
		if len(cfg.Keys) != 0 || len(cfg.Tokens) != 0 || cfg.Audience != "" ||
			cfg.ClientCAPEM != "" || cfg.CertificateIdentity != "" {
			return nil, nil, errors.New("local OS configuration contains another authentication mode")
		}
		verifier, err := authentication.NewLocalOS(cfg.Issuer)
		return verifier, nil, err
	default:
		return nil, nil, errors.New("unsupported caller authentication type")
	}
}

func mtlsAuthenticator(cfg Config) (authentication.Authenticator, *tls.Config, error) {
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM([]byte(cfg.ClientCAPEM)) {
		return nil, nil, errors.New("mTLS requires trusted client CA certificates")
	}
	if cfg.CertificateIdentity != "uri_san" && cfg.CertificateIdentity != "dns_san" {
		return nil, nil, errors.New("mTLS certificate_identity must be uri_san or dns_san")
	}
	verifier, err := authentication.NewMTLS(cfg.Issuer, roots, func(cert *x509.Certificate) (string, error) {
		if cfg.CertificateIdentity == "uri_san" && len(cert.URIs) == 1 {
			return cert.URIs[0].String(), nil
		}
		if cfg.CertificateIdentity == "dns_san" && len(cert.DNSNames) == 1 {
			return cert.DNSNames[0], nil
		}
		return "", errors.New("client certificate must have exactly one configured identity SAN")
	})
	return verifier, &tls.Config{
		MinVersion: tls.VersionTLS12, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: roots,
	}, err
}

func jwtAuthenticator(cfg Config) (authentication.Authenticator, error) {
	if len(cfg.Keys) == 0 {
		return nil, errors.New("JWT requires trusted keys")
	}
	keys := map[string]any{}
	algorithms := map[string]string{}
	var allowed []string
	for _, spec := range cfg.Keys {
		if spec.ID == "" || keys[spec.ID] != nil {
			return nil, errors.New("verification keys require unique nonempty IDs")
		}
		var key any
		var err error
		switch spec.Algorithm {
		case "EdDSA":
			key, err = jwt.ParseEdPublicKeyFromPEM([]byte(spec.PEM))
		case "RS256", "RS384", "RS512", "PS256", "PS384", "PS512":
			key, err = jwt.ParseRSAPublicKeyFromPEM([]byte(spec.PEM))
		case "ES256", "ES384", "ES512":
			key, err = jwt.ParseECPublicKeyFromPEM([]byte(spec.PEM))
		default:
			return nil, errors.New("unsupported caller JWT algorithm")
		}
		if err != nil {
			return nil, errors.New("invalid caller verification key")
		}
		if value, ok := key.(*rsa.PublicKey); ok && value.N.BitLen() < 2048 {
			return nil, errors.New("RSA key must have at least 2048 bits")
		}

		keys[spec.ID], algorithms[spec.ID] = key, spec.Algorithm
		allowed = append(allowed, spec.Algorithm)
	}
	verifier, err := authentication.NewJWT(cfg.Issuer, cfg.Audience, allowed, func(token *jwt.Token) (any, error) {
		id, ok := token.Header["kid"].(string)
		if !ok || keys[id] == nil || algorithms[id] != token.Method.Alg() {
			return nil, errors.New("untrusted caller verification key")
		}
		return keys[id], nil
	})
	if err != nil {
		return nil, err
	}
	return verifier, nil
}

func (m *manager) Mode() string { return m.mode }
func (m *manager) TLSConfig() *tls.Config {
	if m.tlsConfig == nil {
		return nil
	}
	return m.tlsConfig.Clone()
}

func (m *manager) Establish(ctx context.Context) (authorization.Session, context.Context, context.CancelFunc, error) {
	evidence, remote := ctx.Value(evidenceKey{}).(authentication.Evidence)
	if remote && m.mode == "local_os" {
		return nil, nil, nil, errors.New("local OS authentication is unavailable to remote callers")
	}
	if !remote && m.mode != "local_os" {
		evidence.Token = os.Getenv("STACKQL_CALLER_TOKEN")
	}
	identity, err := m.authenticator.Authenticate(ctx, evidence)
	if err != nil {
		return nil, nil, nil, errors.New("caller authentication failed")
	}
	execution, cancel, err := m.executionContext(ctx, evidence)
	if err != nil {
		return nil, nil, nil, err
	}
	caller, err := authorization.NewSession(execution, identity, m.directory, m.authorizer)
	if err != nil {
		cancel()
		return nil, nil, nil, errors.New("caller identity resolution failed")
	}
	return caller, caller.Context(execution), cancel, nil
}

// executionContext preserves caller timeouts; credential-derived limits are opt-in.
func (m *manager) executionContext(
	ctx context.Context,
	evidence authentication.Evidence,
) (context.Context, context.CancelFunc, error) {
	if !m.enforceDeadline {
		execution, cancel := context.WithCancel(ctx)
		return execution, cancel, nil
	}
	deadline := time.Now().Add(m.lifetime)
	if m.mode == "jwt" {
		// Inspect expiry only after the verifier authenticates this exact token.
		claims := jwt.RegisteredClaims{}
		if _, _, err := jwt.NewParser().ParseUnverified(evidence.Token, &claims); err != nil || claims.ExpiresAt == nil {
			return nil, nil, errors.New("caller expiry is required")
		}
		if claims.ExpiresAt.Time.Before(deadline) {
			deadline = claims.ExpiresAt.Time
		}
	}
	if m.mode == "mtls" && evidence.TLS.PeerCertificates[0].NotAfter.Before(deadline) {
		deadline = evidence.TLS.PeerCertificates[0].NotAfter
	}
	execution, cancel := context.WithDeadline(ctx, deadline)
	return execution, cancel, nil
}
