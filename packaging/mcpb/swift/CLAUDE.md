# CLAUDE.md - packaging/mcpb/swift (SwiftPM `StackQLMCP`)

The Swift/macOS member of the StackQL embedded-MCP family: `StackQLMCP` locates the `stackql` binary (override, a copy bundled inside the host `.app` - the shipping path, since the notarised binary keeps its Developer ID signature there - the shared cache, a local bundle, or a pin-verified download; `Options.allowDownload = false` for shipping apps), spawns it over stdio and returns a connected MCP Swift SDK client. Swift 6.1 / macOS 13 (set by the SDK). The source of truth is here; `scripts/publish-mirror.sh` pushes the rendered tree to `stackql/stackql-mcp-swift` and tags `v<version>` (SwiftPM needs `Package.swift` at the repo root). `Sources/StackQLMCP/Version.swift` (`StackQLMCPVersion.current`) is the stamp. No Swift toolchain runs on Windows/Linux CI slices - the swift slice is macOS only.

## Tier: preview

This vector is in tree, on the shared contract and renderer, and CI-validated by the `sdk` matrix on every packaging PR and dispatch - but it is NOT published and is not part of the release train (the published tier is cargo, go, dotnet). `make swift-publish` exists for a deliberate out-of-band publish; promoting this vector means adding a publish job to `mcp-packaging.yml`, its secrets, and its row in `docs/stackql release process.md`.

## The contract (do not deviate)

Owned by [packaging/mcpb](../CLAUDE.md) in stackql/stackql - read "The nine wrapper vectors and the ordering rules" there first. In short:

- Package version == stackql release version; stamped by `scripts/render-platforms.sh` (`make swift-manifest VERSION=X.Y.Z` from `packaging/mcpb`), never hand-edited.
- Pins are data: `platforms.json` `{version, baseUrl, platforms{<key>:{bundle, sha256}}}` rendered from the published `.mcpb.sha256` release assets. `Sources/StackQLMCP/Resources/platforms.json` is a package resource read via `Bundle.module` by `Pins`. No hand-written pin table anywhere. The rendered files are gitignored - render before building.
- Runtime download from `baseUrl` (`https://releases.stackql.io/stackql/<version>`) with `User-Agent: stackql-mcp-server-swift/<version>`. No GitHub API calls, no `latest`, no version override.
- Shared cache `~/.stackql/mcp-server-bin/<version>/<platform-key>/`; keys `linux-x64 | linux-arm64 | windows-x64 | darwin-universal`.
- Overrides `STACKQL_MCP_BIN` and `STACKQL_MCP_BUNDLE` (local `.mcpb`, no pin check); nothing else.
- Canonical argv `mcp --mcp.server.type=stdio --approot <home>/.stackql --mcp.config {"server":{"mode":"<mode>","audit":{"disabled":true}}} [--auth=<json>]`; default mode `read_only`.
- Launcher for `scripts/smoke-test.py --cmd`: `swift run --package-path swift stackql-mcp-launch` (`Sources/stackql-mcp-launch/main.swift`); `Tests/StackQLMCPTests/ConformanceTests.swift` is the in-package port (`STACKQL_MCP_INTEGRATION=1`).

## Build, test, publish

```
cd packaging/mcpb
make swift-manifest VERSION=X.Y.Z   # render pins + version stamp (after the .mcpb assets are on the release)
make swift-build    VERSION=X.Y.Z   # lint + unit tests + package
make swift-smoke    VERSION=X.Y.Z   # smoke-test.py --cmd against the launcher
make swift-publish  VERSION=X.Y.Z   # mirror push + tag via `scripts/publish-mirror.sh` (SDK_MIRROR_TOKEN or local git auth); idempotent, immutable
```

CI: the `sdk` matrix and `swift-publish` job in `.github/workflows/mcp-packaging.yml`.

## Out of scope here

Demo apps (CloudLens, CloudLensCore) live in stackql-labs and depend on the published package. No new features beyond the contract; parity items go on the backlog as separate PRs.

## Writing style

Plain hyphens, ASCII arrows (`->`), QWERTY characters only, matter-of-fact tone, no stacked headings.
