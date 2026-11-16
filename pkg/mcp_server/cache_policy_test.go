package mcp_server //nolint:testpackage,revive // exercise server cache policy

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCatalogueCachePolicy(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EnabledTools = []string{"server_info"}
	cs := connectInProcess(t, cfg, &testBackend{})
	ctx := context.Background()

	tools, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if tools.TTLMs != 30000 || tools.CacheScope != "private" {
		t.Fatalf("tool cache policy = %+v", tools.Cacheable)
	}
	if len(tools.Tools) != 1 || tools.Tools[0].Name != "server_info" {
		t.Fatalf("restricted catalogue = %+v", tools.Tools)
	}

	all := connectInProcess(t, DefaultConfig(), &testBackend{})
	list, err := all.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(list.Tools))
	for _, tool := range list.Tools {
		names = append(names, tool.Name)
	}
	if len(names) <= 1 || !slices.IsSorted(names) {
		t.Fatalf("full catalogue must be ordered and independent of restricted catalogue: %v", names)
	}

	resources, err := all.ListResources(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(resources.Resources) == 0 || resources.CacheScope != "private" || resources.TTLMs != 30000 {
		t.Fatalf("resource catalogue = %+v", resources)
	}
	resource, err := all.ReadResource(ctx, &mcp.ReadResourceParams{URI: resources.Resources[0].URI})
	if err != nil {
		t.Fatal(err)
	}
	if resource.CacheScope != "private" || resource.TTLMs != 0 {
		t.Fatalf("resource content cache policy = %+v", resource.Cacheable)
	}
}

func TestCatalogueCacheInvalidation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Server.Audit.Disabled = true
	server, err := newMCPServer(cfg, &testBackend{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	raw := server.(*simpleMCPServer).server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	if _, err := raw.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatal(err)
	}
	changed := make(chan struct{}, 1)
	client := mcp.NewClient(&mcp.Implementation{Name: "cache-test", Version: "1"}, &mcp.ClientOptions{
		ToolListChangedHandler: func(context.Context, *mcp.ToolListChangedRequest) {
			changed <- struct{}{}
		},
	})
	cs, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	if _, err := cs.ListTools(ctx, nil); err != nil {
		t.Fatal(err)
	}
	raw.AddTool(&mcp.Tool{Name: "cache_test_added", InputSchema: map[string]any{"type": "object"}},
		func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{}, nil
		})
	select {
	case <-changed:
	case <-ctx.Done():
		t.Fatal("catalogue change notification not received")
	}
	tools, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools.Tools {
		if tool.Name == "cache_test_added" {
			return
		}
	}
	t.Fatal("catalogue did not refresh after notification")
}
