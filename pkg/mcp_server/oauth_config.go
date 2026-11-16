package mcp_server

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/stackql/stackql/pkg/mcp_server/oauthrs"
)

// OAuthResourceServerConfig opts the HTTP transport into OAuth JWT
// verification. All trust anchors and policy settings are required. The
// HTTPClient field is for host-constructed configurations and is never loaded
// from or written to JSON/YAML.
type OAuthResourceServerConfig struct {
	Issuer      string   `json:"issuer" yaml:"issuer"`
	Audience    string   `json:"audience" yaml:"audience"`
	Resource    string   `json:"resource" yaml:"resource"`
	JWKSURL     string   `json:"jwks_url" yaml:"jwks_url"`
	JWKSCAFile  string   `json:"jwks_ca_file,omitempty" yaml:"jwks_ca_file,omitempty"`
	MetadataURL string   `json:"metadata_url" yaml:"metadata_url"`
	Scopes      []string `json:"scopes" yaml:"scopes"`
	Algorithms  []string `json:"algorithms" yaml:"algorithms"`

	CacheTTL   Duration     `json:"cache_ttl,omitempty" yaml:"cache_ttl,omitempty"`
	ClockSkew  Duration     `json:"clock_skew,omitempty" yaml:"clock_skew,omitempty"`
	HTTPClient *http.Client `json:"-" yaml:"-"`
}

func (c *OAuthResourceServerConfig) newResourceServer() (*oauthrs.ResourceServer, error) {
	if c == nil {
		return nil, errors.New("OAuth configuration is required")
	}
	httpClient := c.HTTPClient
	if c.JWKSCAFile != "" {
		if httpClient != nil {
			return nil, errors.New("jwks_ca_file cannot be combined with a custom HTTP client")
		}
		caPEM, err := os.ReadFile(c.JWKSCAFile)
		if err != nil {
			return nil, fmt.Errorf("read jwks_ca_file: %w", err)
		}
		roots := x509.NewCertPool()
		if !roots.AppendCertsFromPEM(caPEM) {
			return nil, errors.New("jwks_ca_file contains no usable certificates")
		}
		defaultTransport, ok := http.DefaultTransport.(*http.Transport)
		if !ok {
			return nil, errors.New("jwks_ca_file requires the default HTTP transport to be an *http.Transport")
		}
		transport := defaultTransport.Clone()
		transport.TLSClientConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			RootCAs:    roots,
		}
		httpClient = &http.Client{Transport: transport}
	}
	server, err := oauthrs.NewResourceServer(oauthrs.Config{
		Issuer:      c.Issuer,
		Audience:    c.Audience,
		Resource:    c.Resource,
		JWKSURL:     c.JWKSURL,
		MetadataURL: c.MetadataURL,
		Scopes:      c.Scopes,
		Algorithms:  c.Algorithms,
		CacheTTL:    time.Duration(c.CacheTTL),
		ClockSkew:   time.Duration(c.ClockSkew),
		HTTPClient:  httpClient,
	})
	if err != nil {
		return nil, err
	}
	return server, nil
}

func (c *OAuthResourceServerConfig) validateMetadataPath() error {
	if c == nil {
		return errors.New("OAuth configuration is required")
	}
	parsed, err := url.Parse(c.MetadataURL)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" {
		return errors.New("metadata_url must be an absolute HTTPS URL")
	}
	if parsed.EscapedPath() == "" || parsed.EscapedPath() == "/" {
		return fmt.Errorf("metadata_url must identify a non-root HTTP path")
	}
	return nil
}

func (c *Config) validateOAuth() error {
	if c.Server.OAuth == nil {
		return nil
	}
	if c.GetServerTransport() != serverTransportHTTP {
		return errors.New("server.oauth is supported only when server.transport is http")
	}
	if c.Server.AuthTokenEnvVar != "" || c.Server.AllowUnauthenticated || c.AuthenticateRequest != nil {
		return errors.New("server.oauth conflicts with server.auth_token_env_var, server.allow_unauthenticated, or AuthenticateRequest")
	}
	if c.TLSConfig != nil && c.TLSConfig.ClientAuth != tls.NoClientCert {
		return errors.New("server.oauth cannot be combined with TLS client-certificate authentication")
	}
	if (c.Server.TLSCertFile == "") != (c.Server.TLSKeyFile == "") {
		return errors.New("server.oauth requires both server.tls_cert_file and server.tls_key_file when either is set")
	}
	if _, err := c.Server.OAuth.newResourceServer(); err != nil {
		return fmt.Errorf("invalid server.oauth configuration: %w", err)
	}
	if err := c.Server.OAuth.validateMetadataPath(); err != nil {
		return fmt.Errorf("invalid server.oauth configuration: %w", err)
	}
	return nil
}

func secureOAuthHTTPHandler(handler http.Handler, config *Config) (http.Handler, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid OAuth HTTP configuration: %w", err)
	}
	resourceServer, err := config.Server.OAuth.newResourceServer()
	if err != nil {
		return nil, fmt.Errorf("initialize OAuth resource server: %w", err)
	}
	metadataURL, err := url.Parse(config.Server.OAuth.MetadataURL)
	if err != nil {
		return nil, fmt.Errorf("parse protected-resource metadata URL: %w", err)
	}
	metadataPath := metadataURL.EscapedPath()

	protection := http.NewCrossOriginProtection()
	protection.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rejectRequest(w, r, http.StatusForbidden, "cross-origin request rejected")
	}))
	protectedMCP := protection.Handler(drainRejectedRequestBodyBeforeOAuthResponse(resourceServer.Middleware(handler)))
	metadataHandler := resourceServer.ProtectedResourceMetadataHandler()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() == metadataPath {
			metadataHandler.ServeHTTP(w, r)
			return
		}
		protectedMCP.ServeHTTP(w, r)
	}), nil
}

func drainRejectedRequestBodyBeforeOAuthResponse(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&oauthResponseWriter{ResponseWriter: w, request: r}, r)
	})
}

type oauthResponseWriter struct {
	http.ResponseWriter
	request     *http.Request
	wroteHeader bool
}

func (w *oauthResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	if statusCode >= 100 && statusCode < http.StatusOK && statusCode != http.StatusSwitchingProtocols {
		w.ResponseWriter.WriteHeader(statusCode)
		return
	}
	w.wroteHeader = true
	if statusCode >= http.StatusBadRequest {
		drainRejectedRequestBody(w.request)
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *oauthResponseWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(data)
}

func (w *oauthResponseWriter) Flush() {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *oauthResponseWriter) FlushError() error {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if flusher, ok := w.ResponseWriter.(interface{ FlushError() error }); ok {
		return flusher.FlushError()
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
		return nil
	}
	return http.ErrNotSupported
}

func (w *oauthResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return hijacker.Hijack()
}

func (w *oauthResponseWriter) Push(target string, options *http.PushOptions) error {
	pusher, ok := w.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, options)
}

func (w *oauthResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

var _ io.ReaderFrom = (*oauthResponseWriter)(nil)

func (w *oauthResponseWriter) ReadFrom(source io.Reader) (int64, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if readerFrom, ok := w.ResponseWriter.(io.ReaderFrom); ok {
		return readerFrom.ReadFrom(source)
	}
	return io.Copy(w.ResponseWriter, source)
}
