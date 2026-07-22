// Boot a streamable HTTP MCP server with the pkg/mcp_server example backend for
// mcp-fuzzer smoke tests. Prints one JSON line with the endpoint, then blocks.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/stackql/stackql/pkg/mcp_server"
	"github.com/stackql/stackql/pkg/mcp_server/dto"
)

type fuzzBackend struct{}

func (fuzzBackend) Ping(context.Context) error { return nil }
func (fuzzBackend) Close() error               { return nil }

func (fuzzBackend) ServerInfo(context.Context, any) (dto.ServerInfoOutput, error) {
	return dto.ServerInfoOutput{
		Version:          "fuzz-fixture",
		Commit:           "local",
		Platform:         "fuzz/ci",
		Transport:        "http",
		SQLBackend:       "sqlite3",
		ProviderRegistry: "test://registry-mocked",
		ReadOnly:         true,
	}, nil
}

func (fuzzBackend) ListProviders(context.Context) ([]map[string]any, error) {
	return []map[string]any{
		{"name": "google", "version": "v25.11.00355"},
		{"name": "github", "version": "v25.07.00320"},
	}, nil
}

func (fuzzBackend) ListServices(context.Context, dto.HierarchyInput) ([]map[string]any, error) {
	return []map[string]any{{"id": "compute:v1", "name": "compute", "title": "Compute Engine API"}}, nil
}

func (fuzzBackend) ListResources(context.Context, dto.HierarchyInput) ([]map[string]any, error) {
	return []map[string]any{{"name": "networks"}}, nil
}

func (fuzzBackend) ListMethods(context.Context, dto.HierarchyInput) ([]map[string]any, error) {
	return []map[string]any{
		{"MethodName": "list", "RequiredParams": "project", "SQLVerb": "SELECT"},
	}, nil
}

func (fuzzBackend) DescribeResource(context.Context, dto.HierarchyInput) ([]map[string]any, error) {
	return []map[string]any{{"name": "name", "type": "string"}}, nil
}

func (fuzzBackend) DescribeMethod(context.Context, dto.HierarchyInput) ([]map[string]any, error) {
	return []map[string]any{{"name": "project", "type": "string", "param_type": "input_required"}}, nil
}

func (fuzzBackend) ValidateQuery(context.Context, string) ([]map[string]any, error) {
	return []map[string]any{}, nil
}

func (fuzzBackend) RunQueryJSON(context.Context, dto.QueryJSONInput) ([]map[string]any, error) {
	return []map[string]any{{"name": "fuzz-network"}}, nil
}

func (fuzzBackend) ExecQuery(context.Context, string) (map[string]any, error) {
	return map[string]any{"messages": []string{"ok"}}, nil
}

func (fuzzBackend) ListRegistry(context.Context, dto.RegistryInput) ([]map[string]any, error) {
	return []map[string]any{}, nil
}

func (fuzzBackend) PullProvider(context.Context, dto.RegistryInput) (map[string]any, error) {
	return map[string]any{}, nil
}

func main() {
	port := os.Getenv("MCP_FUZZ_PORT")
	if port == "" {
		port = "19992"
	}
	address := "127.0.0.1:" + port
	endpoint := "http://" + address

	cfg := mcp_server.DefaultHTTPConfig()
	cfg.Server.Address = address
	cfg.Server.Mode = "read_only"
	cfg.Server.Audit.Disabled = true

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	server, err := mcp_server.NewAgnosticBackendServer(fuzzBackend{}, cfg, logger)
	if err != nil {
		log.Fatalf("create mcp server: %v", err)
	}

	out, err := json.Marshal(map[string]string{"endpoint": endpoint, "address": address})
	if err != nil {
		log.Fatalf("marshal ready json: %v", err)
	}
	fmt.Println(string(out))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go func() {
		if startErr := server.Start(ctx); startErr != nil && ctx.Err() == nil {
			log.Fatalf("mcp server exited: %v", startErr)
		}
	}()

	<-ctx.Done()
	_ = server.Stop()
}
