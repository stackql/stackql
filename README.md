# stackql-mcp-go

An embedded [StackQL](https://github.com/stackql/stackql) MCP server for Go
programs: cloud queries and provisioning over SQL, compiled into your
binary. No runtime dependencies, no npx, no separate install.

Your program carries the signed stackql release binary via `go:embed`. At
startup this module extracts it to a shared on-disk cache (verifying its
sha256), spawns it as an MCP stdio server, and hands you a connected client
from the official
[modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk).

## Quick start

Add the module:

```sh
go get github.com/stackql/stackql-mcp-go
```

Add a `go:generate` directive where you want the server embedded:

```go
//go:generate go run github.com/stackql/stackql-mcp-go/cmd/stackql-mcp-fetch -platform auto -package main
```

`go generate` downloads the release `.mcpb` bundle for your platform,
verifies it against the published sha256 pin, extracts the server binary,
and writes the `go:embed` glue (a `StackqlMCPBinary()` function). Use
`-platform all` to generate for every supported platform behind build tags;
the binaries belong in `.gitignore`, regenerate them in CI.

Then start the server and use the MCP session directly:

```go
import (
    "context"

    "github.com/modelcontextprotocol/go-sdk/mcp"
    stackqlmcp "github.com/stackql/stackql-mcp-go/embed"
)

func main() {
    ctx := context.Background()
    client, err := stackqlmcp.StartServer(ctx, stackqlmcp.Options{
        Binary: StackqlMCPBinary(), // generated
    })
    if err != nil {
        // ...
    }
    defer client.Close()

    res, err := client.Session.CallTool(ctx, &mcp.CallToolParams{
        Name:      "run_select_query",
        Arguments: map[string]any{"sql": "SELECT name FROM google.compute.regions WHERE project = 'my-project'"},
    })
    // ...
}
```

## Safety modes

The server starts in `read_only` mode by default: metadata and SELECT
tools work, mutations are refused at call time. Escalation is an explicit
caller decision:

```go
client, err := stackqlmcp.StartServer(ctx, stackqlmcp.Options{
    Binary: StackqlMCPBinary(),
    Mode:   stackqlmcp.ModeSafe, // or ModeDeleteSafe, ModeFullAccess
})
```

| Mode | Allows |
|---|---|
| `read_only` (default) | metadata, SELECT |
| `safe` | + non-destructive mutations (INSERT/UPDATE) |
| `delete_safe` | + DELETE |
| `full_access` | everything |

## Trust chain

- Binaries come from the official release `.mcpb` bundles on
  [stackql/stackql releases](https://github.com/stackql/stackql/releases),
  built and signed by
  [stackql/stackql-mcpb-packaging](https://github.com/stackql/stackql-mcpb-packaging)
- `stackql-mcp-fetch` verifies the downloaded bundle against the published
  per-version sha256 pin before extracting, and records the extracted
  binary's sha256 in the generated code
- At runtime, extraction re-verifies that sha256 before the binary is
  executed; a tampered or corrupted cache entry is replaced

Supported platforms: `linux-x64`, `linux-arm64`, `windows-x64`,
`darwin-universal` (both Mac architectures).

The extraction cache is `~/.stackql/mcp-server-bin/<version>/<platform>/`,
shared with the StackQL npm and pypi MCP wrappers, so multiple embedders on
one machine extract once. Launch arguments are cwd-independent: the server
only touches its approot (`~/.stackql` by default), never the working
directory.

## Demo: sandboxctl

[cmd/sandboxctl](cmd/sandboxctl) is the demo app: an on-demand
infrastructure concierge in a single compiled binary, combining the
embedded stackql MCP server with a Claude agent loop
([anthropic-sdk-go](https://github.com/anthropics/anthropic-sdk-go)).

```sh
export ANTHROPIC_API_KEY=...
export GOOGLE_CREDENTIALS="$(cat service-account-key.json)"

sandboxctl request "a small linux vm and a bucket in sydney for 2 days" --project my-project
```

The flow:

1. The agent plans with a `read_only` server: resolves regions and machine
   types, checks quotas, estimates cost, and prints a plan with the exact
   SQL it intends to run
2. Nothing is created until you approve (interactive y/N, or `--approve`)
3. On approval, a fresh server starts in `safe` mode and the agent
   provisions, labelling everything with `sandboxctl-expires=<unix-ts>`
4. `sandboxctl reap --project my-project` finds expired sandboxes and
   tears them down (`delete_safe` mode, same approval gate; `--approve`
   makes it cron-able)

Every SQL statement the agent runs is printed. `sandboxctl tools` lists
the embedded server's MCP tools and needs no credentials.

Build it:

```sh
go generate ./cmd/sandboxctl
go build ./cmd/sandboxctl
```

## Testing

```sh
go generate ./cmd/sandboxctl   # once per clone: fetches the demo's
                               #   embedded binary (gitignored)
go test -short ./...   # unit tests
go test ./embed/       # + conformance tests (downloads the release bundle,
                       #   spawns it, exercises the github provider with no
                       #   credentials; ports the packaging repo smoke test)
```

CI runs both on Linux, macOS, and Windows.

## Roadmap

- In-process mode: when stackql core exposes a public `pkg/mcpserver` API,
  this module gains `StartInProcess` with the same options surface and no
  embedded binary. The proposal is drafted in
  [docs/upstream-mcpserver-proposal.md](docs/upstream-mcpserver-proposal.md).

## License

MIT. MCP registry name: `io.github.stackql/stackql-mcp`.
