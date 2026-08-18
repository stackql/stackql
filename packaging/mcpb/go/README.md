# stackql-mcp-go

An embedded [StackQL](https://github.com/stackql/stackql) MCP server for Go
programs: cloud queries and provisioning over SQL. No npx, no separate
install.

Two acquisition paths behind one API. Vendored: your program carries the
signed stackql release binary via `go:embed` and this module extracts it to
the shared on-disk cache. Sidecar: leave `Options.Binary` unset and the
module resolves the binary at first run from the shared cache, downloading
and pin-verifying the release bundle when absent. Either way it spawns the
binary as an MCP stdio server and hands you a connected client from the
official
[modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk).

The module is version-locked to the stackql release it embeds: the module
tag is `v<stackql version>` (pin the minor, e.g. `v0.10`), and its
per-platform sha256 pins come from `platforms.json`, rendered by
`packaging/mcpb/scripts/render-platforms.sh` from the published
`.mcpb.sha256` release assets - the same manifest the npm, PyPI and other
SDK wrappers use. Source of truth is
[packaging/mcpb/go](https://github.com/stackql/stackql/tree/main/packaging/mcpb/go)
in stackql/stackql; this repository is the publish mirror that `go get`
resolves.

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
  [packaging/mcpb](https://github.com/stackql/stackql/tree/main/packaging/mcpb),
  downloaded through `https://releases.stackql.io` with
  `User-Agent: stackql-mcp-server-go/<version>`
- `stackql-mcp-fetch` (and the sidecar path) verify the downloaded bundle
  against the `platforms.json` sha256 pin before extracting; the generator
  also records the extracted binary's sha256 in the generated code
- At runtime, extraction re-verifies that sha256 before the binary is
  executed; a tampered or corrupted cache entry is replaced

Supported platforms: `linux-x64`, `linux-arm64`, `windows-x64`,
`darwin-universal` (both Mac architectures).

The extraction cache is `~/.stackql/mcp-server-bin/<version>/<platform>/`,
shared with the StackQL npm and pypi MCP wrappers, so multiple embedders on
one machine extract once. Launch arguments are cwd-independent: the server
only touches its approot (`~/.stackql` by default), never the working
directory.

Env overrides (shared with every StackQL wrapper):

- `STACKQL_MCP_BIN`: path to a stackql binary to run directly (skips acquisition)
- `STACKQL_MCP_BUNDLE`: path to a local `.mcpb` to extract instead of downloading (no pin check)

`cmd/stackql-mcp-launch` is a launcher over the sidecar path with inherited
stdio; extra argv is forwarded to the server. It is what
`packaging/mcpb/scripts/smoke-test.py --cmd "go run ./cmd/stackql-mcp-launch"`
drives.

## Demo: sandboxctl

`sandboxctl`, the on-demand infrastructure concierge demo that used to live
in this repository, now lives in [stackql-labs](https://github.com/stackql-labs)
and depends on the published module.

## Testing

```sh
make go-manifest VERSION=X.Y.Z   # from packaging/mcpb: render embed/platforms.json
                                 #   (go:embed needs it; gitignored)
go test -short ./...   # unit tests
go test ./embed/       # + conformance tests (downloads the release bundle,
                       #   spawns it, exercises the github provider with no
                       #   credentials; ports scripts/smoke-test.py)
python scripts/smoke-test.py --cmd "go run -C go ./cmd/stackql-mcp-launch"  # from packaging/mcpb
```

CI (`mcp-packaging.yml` in stackql/stackql) runs these on Linux, macOS, and Windows.

## Roadmap

- In-process mode: when stackql core exposes a public `pkg/mcpserver` API,
  this module gains `StartInProcess` with the same options surface and no
  embedded binary. The proposal is drafted in
  [docs/upstream-mcpserver-proposal.md](docs/upstream-mcpserver-proposal.md).

## License

MIT. MCP registry name: `io.github.stackql/stackql-mcp`.
