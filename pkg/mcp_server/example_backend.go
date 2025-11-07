package mcp_server //nolint:revive // fine for now

import (
	"context"
	"time"

	"github.com/stackql/stackql/pkg/mcp_server/dto"
)

const (
	ExplainerForeignKeyStackql         = "At present, foreign keys are not meaningfully supported in stackql."
	ExplainerFindRelationships         = "At present, relationship finding is not meaningfully supported in stackql."
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

// Stub all Backend interface methods below

func (b *ExampleBackend) Greet(ctx context.Context, args dto.GreetInput) (string, error) {
	return "Hi " + args.Name, nil
}

func (b *ExampleBackend) ServerInfo(ctx context.Context, _ any) (dto.ServerInfoOutput, error) {
	return dto.ServerInfoOutput{
		Name:       "Stackql explorer",
		Info:       "This is an example server.",
		IsReadOnly: false,
	}, nil
}

// Please adjust all below to sensible signatures in keeping with what is above.
// Do it now!
func (b *ExampleBackend) DBIdentity(ctx context.Context, _ any) (map[string]any, error) {
	return map[string]any{
		"identity": "stub",
	}, nil
}

func (b *ExampleBackend) RunQuery(ctx context.Context, args dto.QueryInput) (string, error) {
	return "stub", nil
}

func (b *ExampleBackend) RunQueryJSON(ctx context.Context, input dto.QueryJSONInput) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

// func (b *ExampleBackend) ListTableResources(ctx context.Context, hI dto.HierarchyInput) ([]string, error) {
// 	return []string{}, nil
// }

func (b *ExampleBackend) ReadTableResource(ctx context.Context, hI dto.HierarchyInput) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

func (b *ExampleBackend) PromptWriteSafeSelectTool(ctx context.Context, args dto.HierarchyInput) (string, error) {
	return ExplainerPromptWriteSafeSelectTool, nil
}

// func (b *ExampleBackend) PromptExplainPlanTipsTool(ctx context.Context) (string, error) {
// 	return "stub", nil
// }

func (b *ExampleBackend) ListTablesJSON(ctx context.Context, input dto.ListTablesInput) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

func (b *ExampleBackend) ListTablesJSONPage(ctx context.Context, input dto.ListTablesPageInput) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (b *ExampleBackend) ListTables(ctx context.Context, hI dto.HierarchyInput) (string, error) {
	return "stub", nil
}

func (b *ExampleBackend) ListMethods(ctx context.Context, hI dto.HierarchyInput) (string, error) {
	return "stub", nil
}

func (b *ExampleBackend) DescribeTable(ctx context.Context, hI dto.HierarchyInput) (string, error) {
	return "stub", nil
}

func (b *ExampleBackend) GetForeignKeys(ctx context.Context, hI dto.HierarchyInput) (string, error) {
	return ExplainerForeignKeyStackql, nil
}

func (b *ExampleBackend) FindRelationships(ctx context.Context, hI dto.HierarchyInput) (string, error) {
	return ExplainerFindRelationships, nil
}

func (b *ExampleBackend) ListProviders(ctx context.Context) (string, error) {
	return "stub", nil
}

func (b *ExampleBackend) ListServices(ctx context.Context, hI dto.HierarchyInput) (string, error) {
	return "stub", nil
}

func (b *ExampleBackend) ListResources(ctx context.Context, hI dto.HierarchyInput) (string, error) {
	return "stub", nil
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
		// Simulate connection establishment
		b.connected = true
	}

	// Simulate a ping operation
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
