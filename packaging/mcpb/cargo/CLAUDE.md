# CLAUDE.md - packaging/mcpb/cargo (crates.io `stackql-mcp`)

The Rust member of the StackQL embedded-MCP family: `stackql-mcp` acquires the `stackql` binary (sidecar download or `vendored` embed via `include_bundle!` + `fetch_bundle`), launches it as an MCP stdio server, and returns a connected `rmcp` client. `Builder::command()` is the bring-your-own-MCP-stack escape hatch. MSRV 1.88 (rmcp). Deps: rmcp, serde/serde_json, sha2, tokio, ureq (sidecar), zip; build-dep serde_json.

## The contract (do not deviate)

Owned by [packaging/mcpb](../CLAUDE.md) in stackql/stackql - read "The nine wrapper vectors and the ordering rules" there first. In short:

- Package version == stackql release version; stamped by `scripts/render-platforms.sh` (`make cargo-manifest VERSION=X.Y.Z` from `packaging/mcpb`), never hand-edited.
- Pins are data: `platforms.json` `{version, baseUrl, platforms{<key>:{bundle, sha256}}}` rendered from the published `.mcpb.sha256` release assets. `build.rs` renders `platforms.json` into `STACKQL_VERSION` / `BASE_URL` / `PINS` consts (`src/pins.rs` only declares `Pin` and the lookup helpers). No hand-written pin table anywhere. The rendered files are gitignored - render before building.
- Runtime download from `baseUrl` (`https://releases.stackql.io/stackql/<version>`) with `User-Agent: stackql-mcp-server-cargo/<version>`. No GitHub API calls, no `latest`, no version override.
- Shared cache `~/.stackql/mcp-server-bin/<version>/<platform-key>/`; keys `linux-x64 | linux-arm64 | windows-x64 | darwin-universal`.
- Overrides `STACKQL_MCP_BIN` and `STACKQL_MCP_BUNDLE` (local `.mcpb`, no pin check); nothing else.
- Canonical argv `mcp --mcp.server.type=stdio --approot <home>/.stackql --mcp.config {"server":{"mode":"<mode>","audit":{"disabled":true}}} [--auth=<json>]`; default mode `read_only`.
- Launcher for `scripts/smoke-test.py --cmd`: `cargo run -q --manifest-path cargo/Cargo.toml --example launcher --` (`examples/launcher.rs`); `tests/conformance.rs` is the in-crate port (`cargo test --test conformance -- --include-ignored`).

## Build, test, publish

```
cd packaging/mcpb
make cargo-manifest VERSION=X.Y.Z   # render pins + version stamp (after the .mcpb assets are on the release)
make cargo-build    VERSION=X.Y.Z   # lint + unit tests + package
make cargo-smoke    VERSION=X.Y.Z   # smoke-test.py --cmd against the launcher
make cargo-publish  VERSION=X.Y.Z   # `cargo publish --allow-dirty` (CARGO_REGISTRY_TOKEN); skips if the version exists on crates.io
```

CI: the `sdk` matrix and `cargo-publish` job in `.github/workflows/mcp-packaging.yml`.

## Out of scope here

Demo apps (auditron, stackql-agent, controls/) live in stackql-labs and depend on the published package. No new features beyond the contract; parity items go on the backlog as separate PRs.

## Writing style

Plain hyphens, ASCII arrows (`->`), QWERTY characters only, matter-of-fact tone, no stacked headings.
