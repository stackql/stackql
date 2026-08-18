# StackQLMCP

[![ci](https://github.com/stackql/stackql-mcp-swift/actions/workflows/ci.yml/badge.svg)](https://github.com/stackql/stackql-mcp-swift/actions/workflows/ci.yml)

Embedded [StackQL](https://stackql.io) MCP server for Swift/macOS apps.
StackQL exposes cloud providers (AWS, GitHub, Google, Azure, and more) as
SQL tables; this package locates the `stackql` binary, launches it as an MCP
server over stdio, and hands you a connected client built on the official
[MCP Swift SDK](https://github.com/modelcontextprotocol/swift-sdk).

The reason this package exists in Swift specifically: the published
`stackql` darwin binary is Developer ID signed and Apple notarised, so a
signed Mac app can bundle it inside its own `.app` and still pass
notarisation. See [docs/bundling-and-notarisation.md](docs/bundling-and-notarisation.md).

## Quickstart

```swift
import StackQLMCP

var options = Options()
options.mode = .readOnly
options.auth = ["github": ["type": "null_auth"]]  // demos with zero creds

let server = try await StackQLServer.start(options)
let tools = try await server.listToolNames()
print("\(tools.count) tools available")

let services = try await server.call(
    "list_services", ["provider": "github", "row_limit": 5])
print(services.text)

await server.stop()
```

The github provider in `null_auth` mode needs no cloud credentials, which is
also how the conformance tests run.

## How the binary is located

`StackQLServer.start` resolves the server binary in priority order, the
first three offline:

1. An explicit override (`Options.binaryOverride` or the
   `STACKQL_MCP_BINARY` environment variable) - used by CI and tests.
2. A binary bundled inside the host app's `.app`
   (`Contents/Resources/stackql` or `Contents/Helpers/stackql`). This is the
   shipping path. Resources in a notarised app are not quarantined and keep
   their own Developer ID signature.
3. The shared on-disk cache `~/.stackql/mcp-server-bin/<version>/<key>/`,
   shared with the npm/pypi wrappers and the other language siblings.
4. Download the release `.mcpb` bundle, verify it against the published
   sha256 pin, extract the binary into the shared cache, and clear
   `com.apple.quarantine`.

Shipping apps should bundle the binary (step 2) and set
`Options.allowDownload = false` so there is no runtime network dependency.

Bundles are published per release at
[stackql/stackql](https://github.com/stackql/stackql/releases) by
[stackql/stackql-mcpb-packaging](https://github.com/stackql/stackql-mcpb-packaging).
Platforms: `linux-x64`, `linux-arm64`, `windows-x64`, `darwin-universal`.

## Safety modes

The server enforces a safety contract per session; the package defaults to
the most restrictive. The server lists every tool regardless of mode and
gates execution at call time, so `readOnly` still lists `run_mutation_query`
but refuses to run it.

| Mode | Allows |
|---|---|
| `.readOnly` (default) | SELECT and metadata tools only |
| `.safe` | reads plus non-destructive mutations |
| `.deleteSafe` | safe plus deletes |
| `.fullAccess` | everything |

Escalation is an explicit caller opt-in via `Options.mode`.

## The launch contract

Every launch uses the canonical, cwd-independent arguments (macOS hosts
often launch helpers with cwd `/`, which is read-only):

```
mcp --mcp.server.type=stdio --approot <home>/.stackql
    --mcp.config {"server":{"mode":"<mode>","audit":{"disabled":true}}}
    [--auth=<json>]
```

`StackQLServer.resolveCommand(_:)` returns this exact `(path, args)` pair
without spawning, so external conformance harnesses (the packaging repo's
`smoke-test.py --cmd` mode) can exercise the launcher.

## Installation

Add the package dependency:

```swift
.package(url: "https://github.com/stackql/stackql-mcp-swift.git", from: "0.1.0")
```

and the product:

```swift
.product(name: "StackQLMCP", package: "stackql-mcp-swift")
```

Requires macOS 13+ and Swift 6.1 (Xcode 16.3+), inherited from the MCP
Swift SDK, which is the only dependency.

## Tests

```
swift test                                   # offline unit suite
STACKQL_MCP_INTEGRATION=1 swift test \
  --filter ConformanceTests                  # spawns the real server
```

The conformance suite mirrors the packaging repo's `smoke-test.py`:
initialize -> tools/list -> `pull_provider` github (null_auth) ->
`list_services`, plus a check that `read_only` refuses mutation execution.
CI runs both on macOS runners.

## Demo app

`CloudLens`, a menu bar cloud sentinel that embeds the notarised binary and
runs a small read_only check suite, is planned as a separate target in this
repo. See [CLAUDE.md](CLAUDE.md) for the design.

## License

MIT. mcp-name reference: `io.github.stackql/stackql-mcp`.
