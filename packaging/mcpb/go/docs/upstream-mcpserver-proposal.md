# Proposal: a public `pkg/mcpserver` API in stackql core

Draft for a GitHub discussion on stackql/stackql. Posted as part of
stackql-mcp-go milestone 1.

## Summary

`stackql mcp` already serves MCP over stdio from the CLI. This proposal asks
for the same capability as a public Go package, so that Go programs can run
the StackQL MCP server in-process instead of spawning a child process.

stackql-mcp-go today ships an `embed` package that carries the signed
release binary inside the calling program via `go:embed`, extracts it to a
shared cache, and spawns it over stdio. That works everywhere and matches
the npm/pypi wrappers, but a public `pkg/mcpserver` would remove the
extract-and-spawn step for pure-Go consumers: one process, no temp files, no
process supervision, and the binary size cost is paid once (stackql compiled
in) rather than twice (host binary + embedded server binary).

## Proposed API

The shape below mirrors the embedding contract stackql-mcp-go already
implements, so callers can switch between the two modes without changing
anything else:

```go
package mcpserver // github.com/stackql/stackql/pkg/mcpserver

// Config mirrors the --mcp.config document accepted by `stackql mcp`.
type Config struct {
    // Mode is one of read_only, safe, delete_safe, full_access.
    // Defaults to read_only.
    Mode string
    // AuditDisabled disables session audit (an embedded server has no
    // console session to audit to).
    AuditDisabled bool
}

type Options struct {
    // AppRoot is the stackql application root. Must be absolute.
    // Defaults to ~/.stackql. No other filesystem locations are
    // written: hosts may run with an arbitrary or read-only cwd.
    AppRoot string
    // Config is the server configuration (see above).
    Config Config
    // Auth optionally overrides provider auth, equivalent to --auth.
    Auth map[string]any
    // Logger receives diagnostics. The server must never write
    // diagnostics to the transport.
    Logger *slog.Logger
}

// Serve runs the MCP server over the given transport until ctx is
// cancelled or the transport closes. For in-process use the transport is
// an io.ReadWriteCloser pair (for example net.Pipe or an in-memory
// transport from an MCP SDK); the CLI's stdio mode becomes a thin wrapper
// over the same function.
func Serve(ctx context.Context, transport io.ReadWriteCloser, opts Options) error
```

With that in place, stackql-mcp-go adds:

```go
// StartInProcess starts the server on an in-memory pipe and returns a
// connected client, with the same Options surface as StartServer.
func StartInProcess(ctx context.Context, opts Options) (*Client, error)
```

and consumers choose process isolation (embedded binary) or a single
process (in-process server) per call site.

## Behavioural contract

These are the properties stackql-mcp-go conformance-tests against the
released binaries today (ported from stackql-mcpb-packaging
scripts/smoke-test.py); the in-process server should satisfy the same
suite:

1. initialize handshake completes and reports server info
2. tools/list includes pull_provider, list_services, list_providers
3. pull_provider + list_services for the github provider succeed with
   `{"github": {"type": "null_auth"}}` (the no-credentials fixture)
4. mode gating: in read_only mode, run_mutation_query and
   run_lifecycle_operation calls are refused (`isError`, message
   `server is in 'read_only' mode`); tools are listed regardless of mode
5. cwd independence: no reads from or writes to the working directory;
   everything lives under AppRoot

## Non-goals

- No change to the CLI surface; `stackql mcp` keeps working as-is
- No new transport types in core; stdio stays in the CLI, in-process
  callers bring their own pipe
- No re-export of stackql internals beyond what Serve needs

## Why now

Agentic Go programs increasingly want a single static binary with no
runtime dependencies. The embedded-binary route ships today, but it carries
a ~90MB server binary inside an app that may already link half of the same
code. A small, stable `pkg/mcpserver` entry point is the cheaper long-term
answer, and the conformance suite to keep it honest already exists.
