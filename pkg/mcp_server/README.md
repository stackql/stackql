# StackQL MCP Server Package

This package implements a Model Context Protocol (MCP) server for StackQL, enabling LLMs to consume StackQL as a first-class information source.

## Overview

The `mcp_server` package provides:

1. **Backend Interface Abstraction**: A clean interface for executing queries that can be implemented for in-memory, TCP, or other communication methods
2. **Configuration Management**: Comprehensive configuration structures with JSON and YAML support
3. **MCP Server Implementation**: A complete MCP server supporting multiple transports (stdio, TCP, WebSocket)

## Architecture

The package is designed with zero dependencies on StackQL internals, making it modular and reusable. The key components are:

- `Backend`: Interface for query execution and schema retrieval
- `Config`: Configuration structures with validation
- `MCPServer`: Main server implementation supporting MCP protocol
- `ExampleBackend`: Sample implementation for testing and demonstration

## Usage

### Basic Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/stackql/stackql/pkg/mcp_server"
)

func main() {
    // Create server with default configuration and example backend
    server, err := mcp_server.NewMCPServerWithExampleBackend(nil)
    if err != nil {
        log.Fatal(err)
    }
    
    // Start the server
    ctx := context.Background()
    if err := server.Start(ctx); err != nil {
        log.Fatal(err)
    }
    
    // Server will run until context is cancelled
    <-ctx.Done()
    
    // Graceful shutdown
    server.Stop(context.Background())
}
```

### Custom Configuration

```go
config := &mcp_server.Config{
    Server: mcp_server.ServerConfig{
        Name:                  "My StackQL MCP Server",
        Version:               "1.0.0",
        Description:           "Custom MCP server for StackQL",
        MaxConcurrentRequests: 50,
        RequestTimeout:        mcp_server.Duration(30 * time.Second),
    },
    Backend: mcp_server.BackendConfig{
        Type:              "stackql",
        ConnectionString:  "stackql://localhost:5432",
        MaxConnections:    20,
        ConnectionTimeout: mcp_server.Duration(10 * time.Second),
        QueryTimeout:      mcp_server.Duration(60 * time.Second),
    },
    Transport: mcp_server.TransportConfig{
        EnabledTransports: []string{"stdio", "tcp"},
        TCP: mcp_server.TCPTransportConfig{
            Address: "0.0.0.0",
            Port:    8080,
        },
    },
    Logging: mcp_server.LoggingConfig{
        Level:  "info",
        Format: "json",
        Output: "/var/log/mcp-server.log",
    },
}

server, err := mcp_server.NewMCPServer(config, backend, logger)
```

### Implementing a Custom Backend

```go
type MyBackend struct {
    // Your backend implementation
}

func (b *MyBackend) Execute(ctx context.Context, query string, params map[string]interface{}) (*mcp_server.QueryResult, error) {
    // Execute the query using your preferred method
    // Return structured results
}

func (b *MyBackend) GetSchema(ctx context.Context) (*mcp_server.Schema, error) {
    // Return schema information about available providers and resources
}

func (b *MyBackend) Ping(ctx context.Context) error {
    // Verify backend connectivity
}

func (b *MyBackend) Close() error {
    // Clean up resources
}
```

## Configuration

### JSON Configuration Example

```json
{
  "server": {
    "name": "StackQL MCP Server",
    "version": "1.0.0",
    "description": "Model Context Protocol server for StackQL",
    "max_concurrent_requests": 100,
    "request_timeout": "30s"
  },
  "backend": {
    "type": "stackql",
    "connection_string": "stackql://localhost",
    "max_connections": 10,
    "connection_timeout": "10s",
    "query_timeout": "30s",
    "retry": {
      "enabled": true,
      "max_attempts": 3,
      "initial_delay": "100ms",
      "max_delay": "5s",
      "multiplier": 2.0
    }
  },
  "transport": {
    "enabled_transports": ["stdio", "tcp"],
    "tcp": {
      "address": "localhost",
      "port": 8080,
      "max_connections": 100,
      "read_timeout": "30s",
      "write_timeout": "30s"
    }
  },
  "logging": {
    "level": "info",
    "format": "text",
    "output": "stdout",
    "enable_request_logging": false
  }
}
```

### YAML Configuration Example

```yaml
server:
  name: "StackQL MCP Server"
  version: "1.0.0"
  description: "Model Context Protocol server for StackQL"
  max_concurrent_requests: 100
  request_timeout: "30s"

backend:
  type: "stackql"
  connection_string: "stackql://localhost"
  max_connections: 10
  connection_timeout: "10s"
  query_timeout: "30s"
  retry:
    enabled: true
    max_attempts: 3
    initial_delay: "100ms"
    max_delay: "5s"
    multiplier: 2.0

transport:
  enabled_transports: ["stdio", "tcp"]
  tcp:
    address: "localhost"
    port: 8080
    max_connections: 100
    read_timeout: "30s"
    write_timeout: "30s"

logging:
  level: "info"
  format: "text"
  output: "stdout"
  enable_request_logging: false
```

### Published Tools

The server publishes the following 11 tools. Each tool's rendered output is a markdown table (uniform multi-row results) or a markdown KV record (sparse / single-record / mixed-shape results). Every tool also returns a typed structured DTO for programmatic clients.

| Tool | Renderer | Description |
|---|---|---|
| `server_info` | KV | Server identity and runtime: stackql version, backing SQL engine, provider registry location, read-only flag. Call once at session start. |
| `list_providers` | Table | Available cloud/SaaS providers (top of the hierarchy). No inputs. |
| `list_services` | Table | Services under a provider. Requires `provider`. |
| `list_resources` | Table | Resources under a `provider`.`service`. Requires `provider` and `service`. |
| `list_methods` | Table | Access methods (HTTP operations) for a resource. Call before writing any query. Requires `provider`, `service`, `resource`. |
| `describe_resource` | KV | Output fields for a resource's primary read method. Requires `provider`, `service`, `resource`. |
| `describe_method` | KV | Full I/O contract for one method. Requires `provider`, `service`, `resource`, `method`. |
| `validate_select_query` | KV | Parse and plan a SELECT without executing. Returns `{valid, errors}`. SELECT only. |
| `run_select_query` | Table | Execute a SELECT. Returns `{rows}`. Reads only. |
| `run_mutation_query` | KV | Execute INSERT/UPDATE/REPLACE/DELETE against the provider. **Real side effects.** Returns `{messages, timestamp}`. Refused in read-only mode. |
| `run_lifecycle_operation` | KV | Execute a stackql `EXEC` lifecycle operation. Returns `{messages, timestamp}`. Refused in read-only mode. |

### Published Prompts

The server publishes one static prompt:

- `write_safe_select` — guidance for writing safe SELECT queries against stackql resources. The prompt body explains how to use `SHOW METHODS IN <provider>.<service>.<resource>` to discover the best read method and the required `WHERE` parameters.

### Restricting Published Tools and Prompts

The top-level `enabled_tools` and `enabled_prompts` fields on `Config` are independent allowlists.

- **Omitted, `null`, or empty list** — every built-in tool (or prompt) is registered. This is the default.
- **Populated list** — only the named items are registered. Any other tool or prompt is absent from `tools/list` / `prompts/list` and the corresponding `tools/call` or `prompts/get` returns an `unknown tool`/`unknown prompt` error.

Enforcement happens at registration time in `pkg/mcp_server/server.go` via the `addToolIfEnabled` and `addPromptIfEnabled` helpers, which consult `Config.IsToolEnabled(name)` / `Config.IsPromptEnabled(name)` before delegating to the SDK. There is no runtime cost for items that are not enabled — they are never bound to the server.

JSON example — a single-purpose server that exposes only `server_info`:

```json
{
  "server": {
    "transport": "http",
    "address": "127.0.0.1:9915"
  },
  "enabled_tools": ["server_info"]
}
```

When the server is launched via the `stackql mcp` (or `stackql srv --mcp.server.type=...`) command, these fields are parsed from the same `--mcp.config` JSON blob as the rest of the configuration — no additional flag is required.  For example, `stackql mcp --mcp.config='{"server": { "transport": "http",    "address": "127.0.0.1:9915"}, "enabled_tools": ["server_info"]}'`.

## MCP Protocol Support

The server implements the Model Context Protocol specification and supports:

- **Initialization**: Capability negotiation with MCP clients
- **Resources**: Listing and reading StackQL resources (providers, services, resources)
- **Tools**: Query execution tool for running StackQL queries
- **Multiple Transports**: stdio, TCP, and WebSocket (WebSocket implementation is placeholder)

### Supported MCP Methods

- `initialize`: Server initialization and capability negotiation
- `resources/list`: List available StackQL resources
- `resources/read`: Read specific resource data
- `tools/list`: List available tools (StackQL query execution)
- `tools/call`: Execute StackQL queries

## Transport Support

### Stdio Transport
- Primary transport for command-line integration
- JSON-RPC over stdin/stdout
- Ideal for shell integrations and CLI tools

### TCP Transport
- HTTP-based JSON-RPC
- Suitable for network-based integrations
- Configurable address, port, and connection limits

### WebSocket Transport (Placeholder)
- Real-time bidirectional communication
- Suitable for web applications
- Currently implemented as placeholder

## Development

### Testing

The package includes an example backend for testing:

```bash
go test ./pkg/mcp_server/...
```

### Integration with StackQL

To integrate with actual StackQL:

1. Implement the `Backend` interface using StackQL's query execution engine
2. Map StackQL's schema information to the `Schema` structure
3. Handle StackQL-specific error types and convert them to `BackendError`

## Dependencies

The package uses minimal external dependencies:
- `github.com/gorilla/mux`: HTTP routing (already available in StackQL)
- `golang.org/x/sync`: Concurrency utilities (already available in StackQL)
- `gopkg.in/yaml.v2`: YAML configuration support (already available in StackQL)

No MCP SDK dependency is required as the package implements the MCP protocol directly.

## Future Enhancements

1. **Full WebSocket Implementation**: Complete WebSocket transport support
2. **Stdio Transport**: Complete stdio JSON-RPC implementation
3. **Authentication**: Add authentication and authorization support
4. **Streaming**: Support for streaming large query results
5. **Caching**: Query result caching for improved performance
6. **Metrics**: Prometheus metrics for monitoring and observability