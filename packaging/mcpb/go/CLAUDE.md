# CLAUDE.md - packaging/mcpb/go (module `github.com/stackql/stackql-mcp-go`)

The Go member of the StackQL embedded-MCP family: package `embed` runs the `stackql` MCP server either from a `go:embed`-carried binary (`cmd/stackql-mcp-fetch` downloads and pin-verifies it at `go generate` time) or via the sidecar path (shared cache, then a pin-verified download) when `Options.Binary` is unset, and returns a connected `modelcontextprotocol/go-sdk` client. Go 1.25. The module path stays `github.com/stackql/stackql-mcp-go`: the source of truth is here, `scripts/publish-mirror.sh` pushes the rendered tree to that mirror repo and tags `v<version>` (Go modules cannot be consumed cleanly from a monorepo subdirectory).

## The contract (do not deviate)

Owned by [packaging/mcpb](../CLAUDE.md) in stackql/stackql - read "The six wrapper vectors and the ordering rules" there first. In short:

- Package version == stackql release version; stamped by `scripts/render-platforms.sh` (`make go-manifest VERSION=X.Y.Z` from `packaging/mcpb`), never hand-edited.
- Pins are data: `platforms.json` `{version, baseUrl, platforms{<key>:{bundle, sha256}}}` rendered from the published `.mcpb.sha256` release assets. `embed/platforms.json` is `//go:embed`ded (`embed/manifest.go`); `DefaultVersion`, `BundlePins`, `BundleURL`, `UserAgent` derive from it. No hand-written pin table anywhere. The rendered files are gitignored - render before building.
- Runtime download from `baseUrl` (`https://releases.stackql.io/stackql/<version>`) with `User-Agent: stackql-mcp-server-go/<version>`. No GitHub API calls, no `latest`, no version override.
- Shared cache `~/.stackql/mcp-server-bin/<version>/<platform-key>/`; keys `linux-x64 | linux-arm64 | windows-x64 | darwin-universal`.
- Overrides `STACKQL_MCP_BIN` and `STACKQL_MCP_BUNDLE` (local `.mcpb`, no pin check); nothing else.
- Canonical argv `mcp --mcp.server.type=stdio --approot <home>/.stackql --mcp.config {"server":{"mode":"<mode>","audit":{"disabled":true}}} [--auth=<json>]`; default mode `read_only`.
- Launcher for `scripts/smoke-test.py --cmd`: `go run -C go ./cmd/stackql-mcp-launch`; `embed/integration_test.go` is the in-module port (`go test ./embed/`).

## Build, test, publish

```
cd packaging/mcpb
make go-manifest VERSION=X.Y.Z   # render pins + version stamp (after the .mcpb assets are on the release)
make go-build    VERSION=X.Y.Z   # lint + unit tests + package
make go-smoke    VERSION=X.Y.Z   # smoke-test.py --cmd against the launcher
make go-publish  VERSION=X.Y.Z   # mirror push + tag via `scripts/publish-mirror.sh` (SDK_MIRROR_TOKEN or local git auth); idempotent, immutable
```

CI: the `sdk` matrix and `go-publish` job in `.github/workflows/mcp-packaging.yml`.

## Out of scope here

Demo apps (sandboxctl) live in stackql-labs and depend on the published package. No new features beyond the contract; parity items go on the backlog as separate PRs.

## Writing style

Plain hyphens, ASCII arrows (`->`), QWERTY characters only, matter-of-fact tone, no stacked headings.
