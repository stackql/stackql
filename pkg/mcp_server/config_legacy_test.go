package mcp_server //nolint:testpackage,revive // exercise loader

import (
	"encoding/json"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/stackql/stackql/pkg/mcp_server/audit"
	"github.com/stackql/stackql/pkg/mcp_server/policy"
)

func TestLoadedDefaultDeadlineStopsToolCall(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cfg, err := LoadFromJSON([]byte(`{"server":{"mode":"read_only","audit":{"disabled":true}}}`))
		if err != nil {
			t.Fatal(err)
		}
		server, err := newMCPServer(cfg, newGatedBackend(), nil)
		if err != nil {
			t.Fatal(err)
		}
		serverTransport, clientTransport := mcp.NewInMemoryTransports()
		if _, err := server.(*simpleMCPServer).server.Connect(t.Context(), serverTransport, nil); err != nil {
			t.Fatal(err)
		}
		client := mcp.NewClient(&mcp.Implementation{Name: "loaded-limits-test", Version: "1"}, nil)
		session, err := client.Connect(t.Context(), clientTransport, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer session.Close()
		started := time.Now()
		result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "list_providers"})
		if err != nil || result == nil || !result.IsError ||
			!strings.Contains(firstText(t, result), "server.request_timeout of 30s exceeded") {
			t.Fatalf("loaded default deadline call = %#v, error=%v; want the default timeout tool error", result, err)
		}
		if elapsed := time.Since(started); elapsed != 30*time.Second {
			t.Fatalf("default deadline expired after %s, want 30s", elapsed)
		}
	})
}

func TestLoadConfigPreservesAbsentRequestLimits(t *testing.T) {
	for _, tc := range []struct {
		name string
		load func([]byte) (*Config, error)
		raw  string
	}{
		{"JSON empty", LoadFromJSON, `{}`},
		{"JSON server", LoadFromJSON, `{"server":{"transport":"http","mode":"read_only"}}`},
		{"YAML server", LoadFromYAML, "server:\n  transport: http\n  mode: read_only\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := tc.load([]byte(tc.raw))
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Server.MaxConcurrentRequests != 100 || cfg.Server.RequestTimeout != Duration(30*time.Second) {
				t.Fatalf("omitted request limits = %d, %s; want 100, 30s",
					cfg.Server.MaxConcurrentRequests, time.Duration(cfg.Server.RequestTimeout))
			}
			cfg.Server.Audit.Disabled = true
			server, err := newMCPServer(cfg, &testBackend{}, nil)
			if err != nil {
				t.Fatal(err)
			}
			backend, ok := server.(*simpleMCPServer).backend.(*limitedBackend)
			if !ok || backend.limit != 100 || backend.timeout != 30*time.Second {
				t.Fatalf("loaded config did not wire finite backend limits: %T", server.(*simpleMCPServer).backend)
			}
		})
	}
}

func TestUnmarshalServerRequestLimitsPreservesExplicitDisabling(t *testing.T) {
	for _, tc := range []struct {
		raw        string
		concurrent int
		timeout    Duration
	}{
		{`{"mode":"read_only"}`, 7, Duration(time.Second)},
		{`{"max_concurrent_requests":0}`, 0, Duration(time.Second)},
		{`{"request_timeout":"0s"}`, 7, 0},
		{`{"max_concurrent_requests":-1,"request_timeout":"-1s"}`, -1, Duration(-time.Second)},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			server := ServerConfig{MaxConcurrentRequests: 7, RequestTimeout: Duration(time.Second)}
			if err := json.Unmarshal([]byte(tc.raw), &server); err != nil {
				t.Fatal(err)
			}
			if server.MaxConcurrentRequests != tc.concurrent || server.RequestTimeout != tc.timeout {
				t.Fatalf("limits = %d, %s; want %d, %s", server.MaxConcurrentRequests,
					time.Duration(server.RequestTimeout), tc.concurrent, time.Duration(tc.timeout))
			}
		})
	}
}

func TestLoadConfigPreservesExplicitRequestLimits(t *testing.T) {
	for _, tc := range []struct {
		name       string
		load       func([]byte) (*Config, error)
		raw        string
		concurrent int
		timeout    Duration
	}{
		{"JSON zero concurrency", LoadFromJSON, `{"server":{"max_concurrent_requests":0}}`, 0, Duration(30 * time.Second)},
		{"JSON zero timeout", LoadFromJSON, `{"server":{"request_timeout":"0s"}}`, 100, 0},
		{"JSON disabled", LoadFromJSON, `{"server":{"max_concurrent_requests":0,"request_timeout":"0s"}}`, 0, 0},
		{"JSON custom", LoadFromJSON, `{"server":{"max_concurrent_requests":3,"request_timeout":"2s"}}`, 3, Duration(2 * time.Second)},
		{"YAML zero concurrency", LoadFromYAML, "server:\n  max_concurrent_requests: 0\n", 0, Duration(30 * time.Second)},
		{"YAML zero timeout", LoadFromYAML, "server:\n  request_timeout: 0s\n", 100, 0},
		{"YAML disabled", LoadFromYAML, "server:\n  max_concurrent_requests: -1\n  request_timeout: -1s\n", -1, Duration(-time.Second)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := tc.load([]byte(tc.raw))
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Server.MaxConcurrentRequests != tc.concurrent || cfg.Server.RequestTimeout != tc.timeout {
				t.Fatalf("limits = %d, %s; want %d, %s", cfg.Server.MaxConcurrentRequests,
					time.Duration(cfg.Server.RequestTimeout), tc.concurrent, time.Duration(tc.timeout))
			}
			cfg.Server.Audit.Disabled = true
			inner := &testBackend{}
			server, err := newMCPServer(cfg, inner, nil)
			if err != nil {
				t.Fatal(err)
			}
			backend := server.(*simpleMCPServer).backend
			if tc.concurrent <= 0 && tc.timeout <= 0 && backend != inner {
				t.Fatalf("disabled limits still wrapped backend: %T", backend)
			}
		})
	}
}

func TestLoadFromJSONRejectsMalformedSecurityConfig(t *testing.T) {
	for _, raw := range []string{
		``, `null`, `[]`, `config.json`, `server: {mode: read_only}`, `{} {}`,
		`{"server":{"mode":"read_only","request_timeout":123}}`,
		`{"server":{"request_timeout":"not-a-duration"}}`,
		`{"server":{"mode":"unknown"}}`, `{"server":{"transport":"unknown"}}`,
		`{"server":{"read_only":"true"}}`, `{"server":null}`,
		`{"enabled_tools":"server_info"}`, `{"enabled_tools":[123]}`,
		`{"enabled_tools":["server_info"],}`, `{"enabled_tool":["server_info"]}`,
		`{"server":{"read_ony":true}}`, `{"server":{"audit":{"disable":true}}}`,
		`{"backend":{"connection_string":"postgres://localhost"}}`,
	} {
		t.Run(raw, func(t *testing.T) {
			cfg, err := LoadFromJSON([]byte(raw))
			if err == nil || cfg != nil {
				t.Fatalf("accepted malformed config: cfg=%+v err=%v", cfg, err)
			}
		})
	}
}

func TestLoadFromJSONPreservesRestrictionsAndZeroLimits(t *testing.T) {
	cfg, err := LoadFromJSON([]byte(`{"server":{"read_only":true,"request_timeout":"0s","max_concurrent_requests":0,"transport_cfg":{"custom":{"setting":true}}},"enabled_tools":["server_info"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GetMode() != policy.ModeReadOnly || !cfg.IsToolEnabled("server_info") || cfg.IsToolEnabled("run_mutation_query") {
		t.Fatalf("restrictions lost: %+v", cfg)
	}
	if cfg.Server.RequestTimeout != 0 || cfg.Server.MaxConcurrentRequests != 0 {
		t.Fatal("zero limits changed")
	}
}

func TestLoadFromJSON_LegacyReadOnlyTrueMapsToReadOnlyMode(t *testing.T) {
	data := []byte(`{"server":{"transport":"http","read_only":true}}`)
	cfg, err := LoadFromJSON(data)
	if err != nil {
		t.Fatalf("LoadFromJSON: %v", err)
	}
	if cfg.Server.Mode != policy.ModeReadOnly {
		t.Errorf("expected Mode=%q, got %q", policy.ModeReadOnly, cfg.Server.Mode)
	}
}

func TestLoadFromJSON_ModeWinsOverLegacyReadOnly(t *testing.T) {
	data := []byte(`{"server":{"transport":"http","read_only":true,"mode":"full_access"}}`)
	cfg, err := LoadFromJSON(data)
	if err != nil {
		t.Fatalf("LoadFromJSON: %v", err)
	}
	if cfg.Server.Mode != policy.ModeFullAccess {
		t.Errorf("expected Mode=%q, got %q", policy.ModeFullAccess, cfg.Server.Mode)
	}
}

func TestLoadFromJSON_LegacyReadOnlyFalseDoesNotForceMode(t *testing.T) {
	data := []byte(`{"server":{"transport":"http","read_only":false}}`)
	cfg, err := LoadFromJSON(data)
	if err != nil {
		t.Fatalf("LoadFromJSON: %v", err)
	}
	if cfg.Server.Mode != "" {
		t.Errorf("read_only:false should not force a mode, got %q", cfg.Server.Mode)
	}
	// GetMode falls back to safe.
	if cfg.GetMode() != policy.ModeSafe {
		t.Errorf("empty mode should fall back to safe, got %q", cfg.GetMode())
	}
}

func TestLoadFromJSON_RejectsUnknownMode(t *testing.T) {
	data := []byte(`{"server":{"transport":"http","mode":"yolo"}}`)
	_, err := LoadFromJSON(data)
	if err == nil {
		t.Fatal("expected validation error for unknown mode")
	}
	if !strings.Contains(err.Error(), "mode") {
		t.Errorf("error should mention mode, got %v", err)
	}
}

func TestLoadFromYAML_LegacyReadOnlyTrueMapsToReadOnlyMode(t *testing.T) {
	data := []byte("server:\n  transport: http\n  read_only: true\n")
	cfg, err := LoadFromYAML(data)
	if err != nil {
		t.Fatalf("LoadFromYAML: %v", err)
	}
	if cfg.Server.Mode != policy.ModeReadOnly {
		t.Errorf("expected Mode=%q, got %q", policy.ModeReadOnly, cfg.Server.Mode)
	}
}

func TestLoadFromJSON_AuditDisabledFlagParses(t *testing.T) {
	data := []byte(`{"server":{"transport":"http","audit":{"disabled":true}}}`)
	cfg, err := LoadFromJSON(data)
	if err != nil {
		t.Fatalf("LoadFromJSON: %v", err)
	}
	if !cfg.Server.Audit.Disabled {
		t.Errorf("expected audit.disabled=true")
	}
	if cfg.IsAuditEnabled() {
		t.Errorf("IsAuditEnabled should be false when audit.disabled=true")
	}
}

func TestValidate_RejectsUnknownAuditFailureMode(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Server.Audit.FailureMode = "explode"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for bad failure_mode")
	}
}

func TestValidate_AcceptsAllLegalFailureModes(t *testing.T) {
	for _, m := range []string{"", audit.FailureModeStrict, audit.FailureModeStrictMutations, audit.FailureModeBestEffort} {
		cfg := DefaultConfig()
		cfg.Server.Audit.FailureMode = m
		if err := cfg.Validate(); err != nil {
			t.Errorf("validate rejected legal failure_mode %q: %v", m, err)
		}
	}
}

func TestGetMode_EmptyFallsBackToSafe(t *testing.T) {
	cfg := &Config{}
	if cfg.GetMode() != policy.ModeSafe {
		t.Errorf("empty mode should map to safe, got %q", cfg.GetMode())
	}
}
