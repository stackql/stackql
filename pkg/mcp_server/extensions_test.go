package mcp_server //nolint:testpackage,revive // exercise internal wiring

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func marshalString(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

// With every extension off the built capabilities are the SDK default, so
// the server advertises exactly what it did before extensions existed.
func TestBuildServerCapabilitiesDisabledEqualsSDKDefault(t *testing.T) {
	got := marshalString(t, buildServerCapabilities(DefaultConfig()))
	if got != `{"logging":{}}` {
		t.Fatalf("disabled capabilities = %s, want the SDK default {\"logging\":{}}", got)
	}
	cs := connectInProcess(t, DefaultConfig(), &testBackend{})
	caps := cs.InitializeResult().Capabilities
	if caps.Extensions != nil || caps.Logging == nil || caps.Tools == nil || caps.Prompts == nil || caps.Resources == nil {
		t.Fatalf("initialize capabilities changed: %s", marshalString(t, caps))
	}
}

func TestBuildServerCapabilitiesAdvertisesEachEnabledExtension(t *testing.T) {
	cases := []struct {
		name   string
		enable func(*Config)
		want   string
	}{
		{"skills", func(c *Config) { c.Server.Extensions.Skills.Enabled = true },
			`{"extensions":{"io.modelcontextprotocol/skills":{}},"logging":{}}`},
		{"tasks", func(c *Config) { c.Server.Extensions.Tasks.Enabled = true },
			`{"extensions":{"io.modelcontextprotocol/tasks":{}},"logging":{}}`},
		{"files", func(c *Config) { c.Server.Extensions.Files.Enabled = true },
			`{"extensions":{"io.modelcontextprotocol/files":{"maxInlineBytes":1048576}},"logging":{}}`},
		{"files with limit", func(c *Config) {
			c.Server.Extensions.Files.Enabled = true
			c.Server.Extensions.Files.MaxInlineBytes = 4096
		}, `{"extensions":{"io.modelcontextprotocol/files":{"maxInlineBytes":4096}},"logging":{}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tc.enable(cfg)
			if got := marshalString(t, buildServerCapabilities(cfg)); got != tc.want {
				t.Fatalf("capabilities = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestClientDeclaresExtension(t *testing.T) {
	cases := []struct {
		name string
		caps *mcp.ClientCapabilities
		want bool
	}{
		{"nil capabilities", nil, false},
		{"no extensions", &mcp.ClientCapabilities{}, false},
		{"other extension", &mcp.ClientCapabilities{Extensions: map[string]any{"io.example/x": map[string]any{}}}, false},
		{"empty settings", &mcp.ClientCapabilities{Extensions: map[string]any{tasksExtensionID: map[string]any{}}}, true},
		{"null settings", &mcp.ClientCapabilities{Extensions: map[string]any{tasksExtensionID: nil}}, true},
		{"settings with fields", &mcp.ClientCapabilities{Extensions: map[string]any{
			filesExtensionID: map[string]any{"upload": map[string]any{}}}}, true},
		{"unknown revision", &mcp.ClientCapabilities{Extensions: map[string]any{
			filesExtensionID: map[string]any{"revision": "2"}}}, false},
		{"not an object", &mcp.ClientCapabilities{Extensions: map[string]any{tasksExtensionID: true}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id := tasksExtensionID
			if strings.Contains(tc.name, "revision") || strings.Contains(tc.name, "fields") {
				id = filesExtensionID
			}
			if got := clientDeclaresExtension(tc.caps, id); got != tc.want {
				t.Fatalf("clientDeclaresExtension = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestExtensionsConfigLoadsWithDefaults(t *testing.T) {
	cfg, err := LoadFromJSON([]byte(`{"server":{"extensions":{` +
		`"skills":{"enabled":true,"enabled_skills":["stackql-iac"]},` +
		`"tasks":{"enabled":true,"retention":"2h"},` +
		`"files":{"enabled":true}}}}`))
	if err != nil {
		t.Fatalf("LoadFromJSON: %v", err)
	}
	extensions := cfg.Server.Extensions
	if !extensions.Skills.Enabled || !extensions.Tasks.Enabled || !extensions.Files.Enabled || !extensions.AnyEnabled() {
		t.Fatalf("extensions not enabled: %+v", extensions)
	}
	if !extensions.Skills.IsSkillEnabled("stackql-iac") || extensions.Skills.IsSkillEnabled("stackql-deploy") {
		t.Fatalf("enabled_skills allowlist not applied: %+v", extensions.Skills)
	}
	if extensions.Tasks.GetTimeout() != 30*time.Minute || extensions.Tasks.GetRetention() != 2*time.Hour ||
		extensions.Tasks.GetPollInterval() != 5*time.Second {
		t.Fatalf("task durations = %v %v %v", extensions.Tasks.GetTimeout(), extensions.Tasks.GetRetention(),
			extensions.Tasks.GetPollInterval())
	}
	if extensions.Files.GetMaxInlineBytes() != 1<<20 {
		t.Fatalf("max inline bytes = %d", extensions.Files.GetMaxInlineBytes())
	}
	if DefaultConfig().Server.Extensions.AnyEnabled() {
		t.Fatal("extensions must be off by default")
	}
}

func TestExtensionsConfigRejectsInvalidValues(t *testing.T) {
	for _, raw := range []string{
		`{"server":{"extensions":{"tasks":{"timeout":"-1s"}}}}`,
		`{"server":{"extensions":{"files":{"max_inline_bytes":-1}}}}`,
		`{"server":{"extensions":{"files":{"max_inline_bytes":4194304}}}}`,
		`{"server":{"extensions":{"tasks":{"enabled":true,"unknown":1}}}}`,
		`{"server":{"extensions":{"uploads":{"enabled":true}}}}`,
	} {
		if _, err := LoadFromJSON([]byte(raw)); err == nil {
			t.Errorf("LoadFromJSON(%s) succeeded, want an error", raw)
		}
	}
}
