# CLAUDE.md - packaging/mcpb/gleam (hex `stackql_mcp`)

The Gleam / BEAM member of the StackQL embedded-MCP family: `stackql_mcp` acquires the `stackql` binary (`stackql_mcp/acquire`: overrides, shared cache, httpc download + sha256 verify + entry-point extraction via the Erlang FFI in `src/stackql_mcp_ffi.erl`), spawns it through `mcp_client`, and exposes both a simple `start`/`call_tool`/`stop` API and an OTP `child_spec` for the caller's supervision tree. Erlang target only, OTP 27+ (the FFI uses the `json` module). `resolve_command` stays pure (names the binary, no acquisition) so the unit suite is offline; `start` acquires.

## Tier: preview

This vector is in tree, on the shared contract and renderer, and CI-validated by the `sdk` matrix on every packaging PR and dispatch - but it is NOT published and is not part of the release train (the published tier is cargo, go, dotnet). `make gleam-publish` exists for a deliberate out-of-band publish; promoting this vector means adding a publish job to `mcp-packaging.yml`, its secrets, and its row in `docs/stackql release process.md`.

## The contract (do not deviate)

Owned by [packaging/mcpb](../CLAUDE.md) in stackql/stackql - read "The nine wrapper vectors and the ordering rules" there first. In short:

- Package version == stackql release version; stamped by `scripts/render-platforms.sh` (`make gleam-manifest VERSION=X.Y.Z` from `packaging/mcpb`), never hand-edited.
- Pins are data: `platforms.json` `{version, baseUrl, platforms{<key>:{bundle, sha256}}}` rendered from the published `.mcpb.sha256` release assets. the renderer stamps `src/stackql_mcp/pins.gleam` (`version`, `base_url`, `pin/1`) - no build-time file reads on the BEAM - alongside `platforms.json`. No hand-written pin table anywhere. The rendered files are gitignored - render before building.
- Runtime download from `baseUrl` (`https://releases.stackql.io/stackql/<version>`) with `User-Agent: stackql-mcp-server-gleam/<version>`. No GitHub API calls, no `latest`, no version override.
- Shared cache `~/.stackql/mcp-server-bin/<version>/<platform-key>/`; keys `linux-x64 | linux-arm64 | windows-x64 | darwin-universal`.
- Overrides `STACKQL_MCP_BIN` and `STACKQL_MCP_BUNDLE` (local `.mcpb`, no pin check); nothing else.
- Canonical argv `mcp --mcp.server.type=stdio --approot <home>/.stackql --mcp.config {"server":{"mode":"<mode>","audit":{"disabled":true}}} [--auth=<json>]`; default mode `read_only`.
- Launcher for `scripts/smoke-test.py --cmd`: `gleam run -m stackql_mcp/launcher --` (relays stdio through the BEAM; run `smoke-test.py` from the `gleam/` dir); `test/conformance_test.gleam` is the in-package port (gated on `STACKQL_MCP_BIN` / `STACKQL_MCP_BUNDLE`).

## Build, test, publish

```
cd packaging/mcpb
make gleam-manifest VERSION=X.Y.Z   # render pins + version stamp (after the .mcpb assets are on the release)
make gleam-build    VERSION=X.Y.Z   # lint + unit tests + package
make gleam-smoke    VERSION=X.Y.Z   # smoke-test.py --cmd against the launcher
make gleam-publish  VERSION=X.Y.Z   # `gleam publish --yes` (HEX_API_KEY); skips if the version exists on hex.pm
```

CI: the `sdk` matrix and `gleam-publish` job in `.github/workflows/mcp-packaging.yml`.

## Out of scope here

Demo apps (pipewatch) live in stackql-labs and depend on the published package. No new features beyond the contract; parity items go on the backlog as separate PRs.

## Writing style

Plain hyphens, ASCII arrows (`->`), QWERTY characters only, matter-of-fact tone, no stacked headings.
