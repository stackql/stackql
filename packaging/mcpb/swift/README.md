# StackQLMCP

[![mcp-packaging](https://github.com/stackql/stackql/actions/workflows/mcp-packaging.yml/badge.svg)](https://github.com/stackql/stackql/actions/workflows/mcp-packaging.yml)

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

1. An explicit override (`Options.binaryOverride` or the `STACKQL_MCP_BIN`
   environment variable, shared with every StackQL wrapper; the old
   `STACKQL_MCP_BINARY` spelling is still read for one release, deprecated).
2. A binary bundled inside the host app's `.app`
   (`Contents/Resources/stackql` or `Contents/Helpers/stackql`). This is the
   shipping path. Resources in a notarised app are not quarantined and keep
   their own Developer ID signature.
3. The shared on-disk cache `~/.stackql/mcp-server-bin/<version>/<key>/`,
   shared with the npm/pypi wrappers and the other language siblings.
4. A local `.mcpb` (`Options.bundleOverride` or `STACKQL_MCP_BUNDLE`),
   extracted without a pin check into `<cache>/custom/<sha256 prefix>/`.
5. Download the release `.mcpb` bundle from `https://releases.stackql.io`
   (User-Agent `stackql-mcp-server-swift/<version>`), verify it against the
   `platforms.json` sha256 pin shipped in the package, extract the binary
   into the shared cache, and clear `com.apple.quarantine`.

Shipping apps should bundle the binary (step 2) and set
`Options.allowDownload = false` so there is no runtime network dependency.

Bundles are published per release at
[stackql/stackql](https://github.com/stackql/stackql/releases) by
[packaging/mcpb](https://github.com/stackql/stackql/tree/main/packaging/mcpb),
which also owns this package (`packaging/mcpb/swift`); the
`stackql/stackql-mcp-swift` repository is the publish mirror SwiftPM
resolves. The package version (git tag `v<version>`) equals the stackql
release it embeds, and its pins come from `platforms.json`, rendered by
`packaging/mcpb/scripts/render-platforms.sh` from the published
`.mcpb.sha256` release assets - the same manifest the npm, PyPI and other
SDK wrappers use.
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
// pin the minor: the package version tracks the stackql release
.package(url: "https://github.com/stackql/stackql-mcp-swift.git", .upToNextMinor(from: "0.10.601"))
```

and the product:

```swift
.product(name: "StackQLMCP", package: "stackql-mcp-swift")
```

Requires macOS 13+ and Swift 6.1 (Xcode 16.3+), inherited from the MCP
Swift SDK, which is the only dependency.

## Tests

```
make swift-manifest VERSION=X.Y.Z            # from packaging/mcpb: render the platforms.json resource + Version.swift (gitignored)
swift test                                   # offline unit suite
STACKQL_MCP_INTEGRATION=1 swift test \
  --filter ConformanceTests                  # spawns the real server
swift run stackql-mcp-launch                 # the conformance launcher (extra argv goes to the server)
python scripts/smoke-test.py --cmd "swift run --package-path swift stackql-mcp-launch"  # from packaging/mcpb
```

The conformance suite mirrors `packaging/mcpb/scripts/smoke-test.py`:
initialize -> tools/list -> `pull_provider` github (null_auth) ->
`list_services`, plus a check that `read_only` refuses mutation execution.
The `mcp-packaging` workflow in stackql/stackql runs both on macOS runners.

## Demo app

`CloudLens`, the menu bar cloud sentinel demo that embeds the notarised
binary and runs a small read_only check suite, lives in
[stackql-labs](https://github.com/stackql-labs) and depends on the published
package.

## License

MIT. mcp-name reference: `io.github.stackql/stackql-mcp`.
