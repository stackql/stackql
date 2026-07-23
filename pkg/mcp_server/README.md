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

The server publishes the following 14 tools. Each tool's rendered output is a markdown table (uniform multi-row results) or a markdown KV record (sparse / single-record / mixed-shape results). Every tool also returns a typed structured DTO for programmatic clients.

Tools carry MCP behavioural annotations derived from the policy-gate classification in `addToolWithGate` (`gate.go`), so the advertised hints and the enforced behaviour share one source of truth: statically select-classified tools get `readOnlyHint: true`, mutation/lifecycle tools an explicit `destructiveHint: true`, and `pull_provider` / `reload_credentials` are marked idempotent and non-destructive (they write only local cache / process env). SQL-carrying tools (`run_select_query` included) make no read-only claim because their effect depends on the submitted statement. Annotations are advisory per the MCP spec; the policy gate remains the enforcement point.

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
| `run_mutation_query` | KV | Execute INSERT/UPDATE/REPLACE/DELETE against the provider. **Real side effects.** Returns `{messages, timestamp}`. Gated by the server [mode](#server-modes). |
| `run_lifecycle_operation` | KV | Execute a stackql `EXEC` lifecycle operation. Returns `{messages, timestamp}`. Gated by the server [mode](#server-modes). |
| `list_registry` | Table | Providers (and their versions) available in the configured registry. Optional `provider` lists versions for that provider. |
| `pull_provider` | KV | Install a provider from the registry into the local approot cache. Requires `provider`; `version` optional. Local cache write only. |
| `reload_credentials` | Table | Re-source credentials from the backend's configured dotenv file into the process environment and report per-provider resolution status (issue #688). Never returns secret values. Optional `provider` scopes the report. Allowed in every mode. |

### Embedded Content: Instructions, Prompts and Resources

Server instructions, prompts and resources are authored as markdown files under `pkg/mcp_server/content/` and compiled into the binary with `go:embed` (issue #696). Adding or changing published content is a markdown-only edit; no Go changes are required.

- `content/instructions/*.md` - concatenated in lexical filename order (blank line separated) into the `instructions` string of the `initialize` result. No frontmatter. Suppress with the top-level `disable_instructions: true` config flag.
- `content/prompts/*.md` - one prompt per file. YAML frontmatter carries `name`, `description` and optional `arguments` (each with `name`, `description`, `required`); the body is the prompt text. `{{argument}}` placeholders in the body are substituted with caller-supplied argument values on `prompts/get`; a placeholder that is not a declared argument fails validation.
- `content/resources/*.md` - one resource per file. Frontmatter carries `name`, `description`, optional `uri` (default `stackql://docs/<filename-sans-extension>`) and optional `mime_type` (default `text/markdown`); the body is served by `resources/read`. The resources capability is declared only when at least one resource is published.

Malformed frontmatter, duplicate names and unresolved placeholders are caught at build time by the unit tests in `embedded_content_test.go`.

Currently published prompts:

- `write_safe_select` - guidance for writing safe SELECT queries against stackql resources. The prompt body explains how to use `SHOW METHODS IN <provider>.<service>.<resource>` to discover the best read method and the required `WHERE` parameters.

Currently published resources:

- `stackql_sql_dialect` (`stackql://docs/sql_dialect`) - notes on the StackQL SQL dialect for provider-backed queries.

### Restricting Published Tools, Prompts and Resources

The top-level `enabled_tools`, `enabled_prompts` and `enabled_resources` fields on `Config` are independent allowlists.

- **Omitted, `null`, or empty list** — every built-in tool (or prompt, or resource) is registered. This is the default.
- **Populated list** — only the named items are registered. Any other tool or prompt is absent from `tools/list` / `prompts/list` and the corresponding `tools/call` or `prompts/get` returns an `unknown tool`/`unknown prompt` error. Likewise for `resources/list` / `resources/read`; when every resource is filtered out the resources capability is not declared at all.

Enforcement happens at registration time in `pkg/mcp_server/server.go` via the `addToolIfEnabled` and `addPromptIfEnabled` helpers, which consult `Config.IsToolEnabled(name)` / `Config.IsPromptEnabled(name)` before delegating to the SDK (resources analogously via `Config.IsResourceEnabled(name)` in `registerEmbeddedResources`). There is no runtime cost for items that are not enabled — they are never bound to the server.

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

## Server Modes

`Config.Server.Mode` chooses one of four safety contracts.  All four allow SELECT and metadata reads; they differ in how they handle mutations and lifecycle operations.

| Mode | SELECT / metadata | INSERT / UPDATE / REPLACE | DELETE | EXEC (lifecycle) |
|---|---|---|---|---|
| `read_only` | allow | refuse | refuse | refuse |
| `safe` (default) | allow | needs approval | needs approval | needs approval |
| `delete_safe` | allow | allow | needs approval | needs approval |
| `full_access` | allow | allow | allow | allow |

`refuse` means the tool returns an error immediately.  `needs approval` means the server tries to elicit user consent via the MCP elicitation flow:

- If the client advertised the elicitation capability at initialise, the server sends an `elicitation/create` request with a short message describing the action and the SQL.  The user accepts, declines, or cancels.
- If the client did NOT advertise elicitation, the tool is refused with a message that explains the gap and points the operator at `full_access` mode.

The mode is global per server.  There is no per-tool override in this release.

### Default-mode change (breaking)

PR1 had a single `read_only: true / false` flag; the default behaviour was "no enforcement, mutations proceed."  PR2 replaces that flag with `mode: safe` as the default, which means **mutations now require user approval out of the box.**  Operators running an elicitation-capable client should see one approval prompt per mutation.  Operators running a non-elicitation client (or an automated pipeline) must explicitly opt into `full_access`.

For back-compat, the legacy `read_only: true` JSON / YAML key still parses and is treated as equivalent to `mode: read_only`.  When both are set, `mode` wins.

## Audit Log

Audit recording is **on by default** in PR2.  Every tool call produces one JSONL record with the tool name, mode, decision, query class, SQL (for query tools), input args (for hierarchy tools), duration, and error.  Result rows from SELECTs are intentionally not recorded - the audit answers "what did the agent do," not "what did the agent see."

### File sink

The only sink kind shipped in this release is `file`, which writes one JSON object per line and fsyncs after each record.  Lumberjack-style rotation by size, age, and backup count.

The sink implementation lives in [`pkg/sink`](/pkg/sink) so it can be reused outside MCP (future activity / telemetry channels, etc).  The MCP audit subsystem feeds `audit.Event` values into a generic `sink.Sink`; the sink JSON-marshals whatever payload it is given.  Adding alternative sinks (rotation policies, Kafka, S3) only requires implementing `sink.Sink` once; it benefits every subsystem that records through this path.

The generic `sink.FileConfig` requires the caller to specify *where* the file lives via either `Path` (a complete file path) or `Dir` (a directory in which the sink picks a filename via `DefaultFilename`).  The sink package never silently picks a directory; the MCP server defaults `Dir` to `.` (cwd) on the operator's behalf when neither field is set in `mcp.config`.

```yaml
server:
  mode: safe
  audit:
    disabled: false       # default false (audit is on)
    failure_mode: strict  # strict | strict_mutations | best_effort
    sink: file            # currently the only kind
    file:
      # Specify either `path` (a complete file path) or `dir` (the directory
      # in which the sink chooses a stackql_mcp_server_<UTC>.log basename).
      # When both are empty the MCP server defaults `dir` to cwd (".") for
      # back-compat; the underlying pkg/sink itself refuses to silently
      # pick a directory.
      path: ""
      dir: ""
      max_size_mb: 100
      max_backups: 5
      max_age_days: 30
```

The resolved absolute path is logged to stderr at startup as `sink file: /path/to/file.log` so operators can find the file later.

### Failure modes

When the sink returns an error, the response depends on `failure_mode`:

| failure_mode | Effect on tool call |
|---|---|
| `strict` (default) | The tool call returns the audit error to the client, even if the underlying tool succeeded.  This is intentional: better an ambiguous client response than an undetected DELETE. |
| `strict_mutations` | SELECT and metadata reads log the audit error to stderr and proceed.  Mutations and lifecycle ops surface the error. |
| `best_effort` | Always log to stderr and proceed. |

### Sequencing

The audit write happens AFTER the tool has executed (or been gated out) but BEFORE the response returns to the client.  In strict mode, an audit-write failure on a successful DELETE means the row is gone but the client receives an error - by design, so no mutation slips through unaudited.

### Audit-on change (breaking)

PR1 had no audit subsystem; nothing was logged.  PR2 enables audit by default.  To preserve PR1 behaviour, set `server.audit.disabled: true`.

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