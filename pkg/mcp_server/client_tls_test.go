package mcp_server //nolint:testpackage,revive // exercise the client transport

import (
	"context"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestHTTPMCPClientTLS(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "tls-test", Version: "1"}, nil)
	endpoint := httptest.NewTLSServer(mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, nil))
	defer endpoint.Close()
	certFile := filepath.Join(t.TempDir(), "server_cert.pem")
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: endpoint.Certificate().Raw})
	if err := os.WriteFile(certFile, certPEM, 0600); err != nil {
		t.Fatal(err)
	}
	defaultTransport := http.DefaultTransport
	defaultClient := http.DefaultClient
	tests := []struct {
		name      string
		config    map[string]any
		wantError bool
	}{
		{name: "untrusted", wantError: true},
		{name: "trusted", config: map[string]any{"ca_file": certFile}},
		{name: "explicit hostname", config: map[string]any{"ca_file": certFile, "server_name": "example.com"}},
		{name: "wrong hostname", config: map[string]any{"ca_file": certFile, "server_name": "wrong.example"}, wantError: true},
		{name: "untrusted after trusted", wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			genericClient, err := newHTTPMCPClient(endpoint.URL, tt.config, nil)
			if err != nil {
				t.Fatal(err)
			}
			client, ok := genericClient.(*httpMCPClient)
			if !ok {
				t.Fatal("unexpected HTTP client implementation")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			session, err := client.connect()
			if (err != nil) != tt.wantError {
				t.Fatalf("connect error = %v, want error %v", err, tt.wantError)
			}
			if err == nil {
				defer session.Close()
				if _, err := session.ListTools(ctx, nil); err != nil {
					t.Fatal(err)
				}
			}
			assertHTTPClientIsolation(t, client.httpClient, defaultClient, defaultTransport)
		})
	}
}

func assertHTTPClientIsolation(t *testing.T, client, defaultClient *http.Client, defaultTransport http.RoundTripper) {
	t.Helper()
	if http.DefaultTransport != defaultTransport || http.DefaultClient != defaultClient {
		t.Fatal("client changed the global HTTP defaults")
	}
	if client == http.DefaultClient {
		return
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("unexpected HTTP transport")
	}
	t.Cleanup(transport.CloseIdleConnections)
	if transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("TLS verification disabled")
	}
}

func TestHTTPMCPClientInvalidTLSConfig(t *testing.T) {
	invalidPEM := filepath.Join(t.TempDir(), "invalid.pem")
	if err := os.WriteFile(invalidPEM, []byte("not a certificate"), 0600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		config map[string]any
	}{
		{name: "non string CA", config: map[string]any{"ca_file": true}},
		{name: "missing CA", config: map[string]any{"ca_file": filepath.Join(t.TempDir(), "missing.pem")}},
		{name: "invalid PEM", config: map[string]any{"ca_file": invalidPEM}},
		{name: "non string hostname", config: map[string]any{"ca_file": invalidPEM, "server_name": true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := newHTTPMCPClient("https://localhost", tt.config, nil); err == nil {
				t.Fatal("invalid TLS configuration accepted")
			}
		})
	}
}
