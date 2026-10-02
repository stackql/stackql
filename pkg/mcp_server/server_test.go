package mcp_server //nolint:testpackage,revive // fine for now

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	if err := config.Validate(); err != nil {
		t.Fatalf("Default config validation failed: %v", err)
	}

	if config.Server.Name == "" {
		t.Error("Server name should not be empty")
	}

	if config.Server.Version == "" {
		t.Error("Server version should not be empty")
	}
}

func TestExampleBackend(t *testing.T) {
	backend := NewExampleBackend("test://localhost")
	ctx := context.Background()

	// Test Ping
	if err := backend.Ping(ctx); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	// Test Close
	if err := backend.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestMCPServerCreation(t *testing.T) {
	config := DefaultConfig()
	backend := NewExampleBackend("test://localhost")

	server, err := newMCPServer(config, backend, nil)
	if err != nil {
		t.Fatalf("NewMCPServer failed: %v", err)
	}

	if server == nil {
		t.Fatal("Server should not be nil")
	}

	// Test that server implements MCPServer interface
	var _ MCPServer = server
}

func TestDurationMarshaling(t *testing.T) {
	d := Duration(30 * time.Second)

	// Test JSON marshaling
	jsonData, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	var d2 Duration
	if err := json.Unmarshal(jsonData, &d2); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if time.Duration(d) != time.Duration(d2) {
		t.Errorf("Duration mismatch after JSON round-trip: %v != %v", d, d2)
	}
}

func TestBackendError(t *testing.T) {
	err := &BackendError{
		Code:    "TEST_ERROR",
		Message: "Test error message",
		Details: map[string]interface{}{"field": "value"},
	}

	if err.Error() != "Test error message" {
		t.Errorf("Expected error message 'Test error message', got '%s'", err.Error())
	}

	// Test Value() method for database compatibility
	val, dbErr := err.Value()
	if dbErr != nil {
		t.Fatalf("Value() failed: %v", dbErr)
	}

	if val != "Test error message" {
		t.Errorf("Expected value 'Test error message', got '%v'", val)
	}
}

func TestIsToolEnabled(t *testing.T) {
	cfg := &Config{}
	if !cfg.IsToolEnabled("anything") {
		t.Errorf("empty EnabledTools should allow all tools")
	}
	cfg.EnabledTools = []string{"foo"}
	if !cfg.IsToolEnabled("foo") {
		t.Errorf("foo should be enabled")
	}
	if cfg.IsToolEnabled("bar") {
		t.Errorf("bar should be denied")
	}
}

func TestIsPromptEnabled(t *testing.T) {
	cfg := &Config{}
	if !cfg.IsPromptEnabled("anything") {
		t.Errorf("empty EnabledPrompts should allow all prompts")
	}
	cfg.EnabledPrompts = []string{"cloud_audit"}
	if !cfg.IsPromptEnabled("cloud_audit") {
		t.Errorf("cloud_audit should be enabled")
	}
	if cfg.IsPromptEnabled("other") {
		t.Errorf("other should be denied")
	}
}

// Issue #784: server.protocol_version pins the newest advertised revision.
func TestProtocolVersionConfig(t *testing.T) {
	legacy := []string{"2025-11-25", "2025-06-18", "2025-03-26", "2024-11-05"}
	cases := []struct {
		name       string
		pinned     string
		stateless  bool
		advertised []string
		wantStless bool
	}{
		{name: "empty is auto", pinned: "", advertised: nil},
		{name: "auto", pinned: "auto", advertised: nil},
		{name: "auto keeps explicit stateless", pinned: "auto", stateless: true, wantStless: true},
		{name: "sessionless stands alone and implies stateless", pinned: "2026-07-28",
			advertised: []string{"2026-07-28"}, wantStless: true},
		{name: "legacy ceiling keeps older revisions", pinned: "2025-11-25", advertised: legacy},
		{name: "older ceiling", pinned: "2025-06-18", advertised: legacy[1:]},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Server.ProtocolVersion = tc.pinned
			cfg.Server.Stateless = tc.stateless
			if err := cfg.Validate(); err != nil {
				t.Fatalf("Validate: %v", err)
			}
			if got := cfg.AdvertisedProtocolVersions(); !reflect.DeepEqual(got, tc.advertised) {
				t.Fatalf("advertised = %v, want %v", got, tc.advertised)
			}
			if got := cfg.IsStateless(); got != tc.wantStless {
				t.Fatalf("IsStateless = %v, want %v", got, tc.wantStless)
			}
		})
	}
	cfg, err := LoadFromJSON([]byte(`{"server": {"protocol_version": "2020-01-01"}}`))
	if err == nil || !strings.Contains(err.Error(), "invalid server.protocol_version") {
		t.Fatalf("unsupported revision must fail validation, got cfg=%v err=%v", cfg, err)
	}
	cfg, err = LoadFromJSON([]byte(`{"server": {"protocol_version": "2025-11-25", "read_only": true}}`))
	if err != nil || cfg.GetProtocolVersion() != "2025-11-25" || cfg.GetMode() != "read_only" {
		t.Fatalf("wire form should carry protocol_version beside the legacy shim: cfg=%+v err=%v", cfg, err)
	}
}

func TestNewMCPServerWithExampleBackend(t *testing.T) {
	server, err := NewMCPServerWithExampleBackend(nil)
	if err != nil {
		t.Fatalf("NewMCPServerWithExampleBackend failed: %v", err)
	}

	if server == nil {
		t.Fatal("Server should not be nil")
	}
}

func TestSecureHTTPHandler(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	status := func(h http.Handler, headers map[string]string) int {
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:9876/", strings.NewReader("{}"))
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	open, err := secureHTTPHandler(ok, DefaultHTTPConfig())
	if err != nil {
		t.Fatalf("no token configured: %v", err)
	}
	if got := status(open, nil); got != http.StatusOK {
		t.Errorf("no token configured: status %d, want 200", got)
	}
	if got := status(open, map[string]string{"Origin": "https://evil.example"}); got != http.StatusForbidden {
		t.Errorf("cross-origin request: status %d, want 403", got)
	}

	cfg := DefaultHTTPConfig()
	cfg.Server.AuthTokenEnvVar = "STACKQL_MCP_TEST_TOKEN"
	if _, err = secureHTTPHandler(ok, cfg); err == nil {
		t.Error("an unset token env var must fail startup")
	}
	t.Setenv("STACKQL_MCP_TEST_TOKEN", "s3cret")
	guarded, err := secureHTTPHandler(ok, cfg)
	if err != nil {
		t.Fatalf("token configured: %v", err)
	}
	for name, tc := range map[string]struct {
		header string
		want   int
	}{
		"missing": {"", http.StatusUnauthorized},
		"wrong":   {"Bearer nope", http.StatusUnauthorized},
		"right":   {"Bearer s3cret", http.StatusOK},
	} {
		headers := map[string]string{}
		if tc.header != "" {
			headers["Authorization"] = tc.header
		}
		if got := status(guarded, headers); got != tc.want {
			t.Errorf("%s token: status %d, want %d", name, got, tc.want)
		}
	}
}

func TestIsLoopbackAddress(t *testing.T) {
	for address, want := range map[string]bool{
		"127.0.0.1:9876": true, "localhost:9876": true, "[::1]:9876": true,
		"0.0.0.0:9876": false, ":9876": false, "192.168.1.10:9876": false, "example.com:9876": false,
	} {
		if got := isLoopbackAddress(address); got != want {
			t.Errorf("isLoopbackAddress(%q) = %v, want %v", address, got, want)
		}
	}
}
