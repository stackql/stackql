package mcp_server //nolint:testpackage,revive // exercise loader

import (
	"strings"
	"testing"

	"github.com/stackql/stackql/pkg/mcp_server/audit"
	"github.com/stackql/stackql/pkg/mcp_server/policy"
)

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
