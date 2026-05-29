package mcp_server //nolint:revive // fine for now

import (
	"context"
	"time"

	"github.com/stackql/stackql/pkg/mcp_server/dto"
)

const (
	// ExplainerPromptWriteSafeSelectTool is the static body of the write_safe_select prompt.
	ExplainerPromptWriteSafeSelectTool = `In order to ascertain the best safe select query, the correct query form is:
	>   SHOW methods IN <provider>.<service>.<resource>;
	From the output, one can infer the best access method for the SQL "select" verb and the **required** WHERE clause attributes.`
)

// ExampleBackend is a simple implementation of the Backend interface for demonstration purposes.
// This shows how to implement the Backend interface without depending on StackQL internals.
type ExampleBackend struct {
	connectionString string
	connected        bool
}

func (b *ExampleBackend) ServerInfo(ctx context.Context, _ any) (dto.ServerInfoOutput, error) {
	return dto.ServerInfoOutput{}, nil
}

func (b *ExampleBackend) RunQueryJSON(ctx context.Context, input dto.QueryJSONInput) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

func (b *ExampleBackend) ValidateQuery(ctx context.Context, query string) ([]map[string]any, error) {
	return []map[string]any{}, nil
}

func (b *ExampleBackend) ListMethods(ctx context.Context, hI dto.HierarchyInput) ([]map[string]any, error) {
	return []map[string]any{}, nil
}

func (b *ExampleBackend) DescribeResource(ctx context.Context, hI dto.HierarchyInput) ([]map[string]any, error) {
	return []map[string]any{}, nil
}

func (b *ExampleBackend) DescribeMethod(ctx context.Context, hI dto.HierarchyInput) ([]map[string]any, error) {
	return []map[string]any{}, nil
}

func (b *ExampleBackend) ExecQuery(ctx context.Context, query string) (map[string]any, error) {
	return map[string]any{}, nil
}

func (b *ExampleBackend) ListProviders(ctx context.Context) ([]map[string]any, error) {
	return []map[string]any{}, nil
}

func (b *ExampleBackend) ListServices(ctx context.Context, hI dto.HierarchyInput) ([]map[string]any, error) {
	return []map[string]any{}, nil
}

func (b *ExampleBackend) ListResources(ctx context.Context, hI dto.HierarchyInput) ([]map[string]any, error) {
	return []map[string]any{}, nil
}

func (b *ExampleBackend) ListRegistry(ctx context.Context, input dto.RegistryInput) ([]map[string]any, error) {
	return []map[string]any{}, nil
}

func (b *ExampleBackend) PullProvider(ctx context.Context, input dto.RegistryInput) (map[string]any, error) {
	return map[string]any{}, nil
}

// NewExampleBackend creates a new example backend instance.
func NewExampleBackend(connectionString string) Backend {
	return &ExampleBackend{
		connectionString: connectionString,
		connected:        false,
	}
}

// Ping implements the Backend interface.
func (b *ExampleBackend) Ping(ctx context.Context) error {
	if !b.connected {
		b.connected = true
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Millisecond):
		return nil
	}
}

// Close implements the Backend interface.
func (b *ExampleBackend) Close() error {
	b.connected = false
	return nil
}

// NewMCPServerWithExampleBackend creates a new MCP server with an example backend.
// This is a convenience function for testing and demonstration purposes.
func NewMCPServerWithExampleBackend(config *Config) (MCPServer, error) {
	if config == nil {
		config = DefaultConfig()
	}

	backend := NewExampleBackend(config.Backend.ConnectionString)

	return newMCPServer(config, backend, nil)
}
