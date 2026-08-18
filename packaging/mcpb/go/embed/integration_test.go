package embed_test

// Conformance test mirroring packaging/mcpb/scripts/smoke-test.py:
// spawn the server over stdio, complete the MCP handshake, then exercise
// the github provider with null_auth (no credentials):
// initialize -> tools/list -> pull_provider -> list_services.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	stackqlmcp "github.com/stackql/stackql-mcp-go/embed"
)

// conformanceBinary returns a server binary for the current platform:
// from $STACKQL_MCP_TEST_BINARY if set, from the local test cache, or by
// downloading and pin-verifying the release bundle.
func conformanceBinary(t *testing.T) stackqlmcp.Binary {
	t.Helper()
	version := stackqlmcp.DefaultVersion
	key, err := stackqlmcp.CurrentPlatformKey()
	if err != nil {
		t.Skipf("no published binary for this platform: %v", err)
	}

	if path := os.Getenv("STACKQL_MCP_TEST_BINARY"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading STACKQL_MCP_TEST_BINARY: %v", err)
		}
		sum := sha256.Sum256(data)
		return stackqlmcp.Binary{
			Data: data, Version: version, PlatformKey: key,
			SHA256: hex.EncodeToString(sum[:]),
		}
	}

	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	cached := filepath.Join(cacheRoot, "stackql-mcp-go-test", version, "stackql-mcp-"+key)
	if data, err := os.ReadFile(cached); err == nil {
		sum := sha256.Sum256(data)
		return stackqlmcp.Binary{
			Data: data, Version: version, PlatformKey: key,
			SHA256: hex.EncodeToString(sum[:]),
		}
	}

	t.Logf("downloading stackql-mcp-%s.mcpb v%s (cached for later runs)", key, version)
	res, err := stackqlmcp.FetchBundle(nil, key)
	if err != nil {
		t.Fatalf("fetching release bundle: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(cached), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cached, res.Binary.Data, 0o755); err != nil {
		t.Fatal(err)
	}
	return res.Binary
}

func textContent(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	var sb strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			sb.WriteString(tc.Text)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func TestConformanceGithubNullAuth(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test: spawns the embedded server and reaches registry.stackql.app")
	}
	bin := conformanceBinary(t)

	// Hermetic approot and cache, as the packaging smoke test does with
	// its substituted ${HOME}: proves cwd-independence and no writes
	// outside the configured roots.
	tmp := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	client, err := stackqlmcp.StartServer(ctx, stackqlmcp.Options{
		Binary:   bin,
		Mode:     stackqlmcp.ModeReadOnly,
		AppRoot:  filepath.Join(tmp, ".stackql"),
		CacheDir: filepath.Join(tmp, "bin-cache"),
		Auth:     map[string]any{"github": map[string]any{"type": "null_auth"}},
	})
	if err != nil {
		t.Fatalf("StartServer: %v", err)
	}
	defer client.Close()

	if got := client.Session.InitializeResult(); got == nil || got.ServerInfo == nil {
		t.Fatal("initialize returned no server info")
	} else {
		t.Logf("initialize ok: server=%s %s", got.ServerInfo.Name, got.ServerInfo.Version)
	}

	tools, err := client.Session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("tools/list: %v", err)
	}
	names := map[string]bool{}
	for _, tool := range tools.Tools {
		names[tool.Name] = true
	}
	t.Logf("tools/list returned %d tools", len(tools.Tools))
	for _, required := range []string{"pull_provider", "list_services", "list_providers"} {
		if !names[required] {
			t.Errorf("missing required tool %q", required)
		}
	}
	if t.Failed() {
		t.FailNow()
	}

	pull, err := client.Session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "pull_provider",
		Arguments: map[string]any{"provider": "github"},
	})
	if err != nil {
		t.Fatalf("pull_provider: %v", err)
	}
	if pull.IsError {
		t.Fatalf("pull_provider failed: %s", textContent(t, pull))
	}
	t.Log("pull_provider github ok")

	services, err := client.Session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_services",
		Arguments: map[string]any{"provider": "github", "row_limit": 5},
	})
	if err != nil {
		t.Fatalf("list_services: %v", err)
	}
	if services.IsError {
		t.Fatalf("list_services failed: %s", textContent(t, services))
	}
	text := textContent(t, services)
	if !strings.Contains(text, "actions") && !strings.Contains(text, "apps") {
		t.Fatalf("list_services did not include expected github services: %s", text)
	}
	t.Log("list_services returned expected github services")
}

// TestReadOnlyModeRefusesMutations asserts the default mode refuses
// mutation execution (the server lists all tools regardless of mode but
// gates calls), which is the safety contract sandboxctl and other
// embedders rely on.
func TestReadOnlyModeRefusesMutations(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	bin := conformanceBinary(t)
	tmp := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client, err := stackqlmcp.StartServer(ctx, stackqlmcp.Options{
		Binary:   bin,
		AppRoot:  filepath.Join(tmp, ".stackql"),
		CacheDir: filepath.Join(tmp, "bin-cache"),
	})
	if err != nil {
		t.Fatalf("StartServer: %v", err)
	}
	defer client.Close()

	res, err := client.Session.CallTool(ctx, &mcp.CallToolParams{
		Name: "run_mutation_query",
		Arguments: map[string]any{
			"sql": "INSERT INTO github.repos.repos(data__name) SELECT 'should-never-run'",
		},
	})
	if err != nil {
		t.Fatalf("run_mutation_query call: %v", err)
	}
	text := textContent(t, res)
	if !res.IsError || !strings.Contains(text, "read_only") {
		t.Fatalf("read_only server did not refuse mutation: isError=%v text=%s", res.IsError, text)
	}
	t.Logf("mutation refused as expected: %s", strings.TrimSpace(text))
}
