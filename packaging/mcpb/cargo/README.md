# stackql-mcp

[![crates.io](https://img.shields.io/crates/v/stackql-mcp.svg)](https://crates.io/crates/stackql-mcp)
[![docs.rs](https://docs.rs/stackql-mcp/badge.svg)](https://docs.rs/stackql-mcp)
[![mcp-packaging](https://github.com/stackql/stackql/actions/workflows/mcp-packaging.yml/badge.svg)](https://github.com/stackql/stackql/actions/workflows/mcp-packaging.yml)

Embedded [StackQL](https://stackql.io) MCP server for Rust agentic apps. StackQL exposes cloud providers (AWS, GitHub, Google, Azure, and more) as SQL tables; this crate acquires the `stackql` binary, launches it as an MCP server over stdio, and hands you a connected [rmcp](https://crates.io/crates/rmcp) client.

## Quickstart

```rust
use stackql_mcp::{Mode, StackqlMcp};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let server = StackqlMcp::builder()
        .mode(Mode::ReadOnly)
        .auth(serde_json::json!({"github": {"type": "null_auth"}}))
        .start()
        .await?;
    let tools = server.list_all_tools().await?;
    println!("{} tools available", tools.len());
    server.shutdown().await?;
    Ok(())
}
```

Run it: `cargo run --example minimal`. The github provider in `null_auth` mode needs no cloud credentials.

## Acquisition modes

Two ways to get the server binary, both behind the same API:

- sidecar (default feature): downloads the platform's `.mcpb` bundle at first run, verifies its sha256 against pins baked into the crate, and caches it. Subsequent starts are offline.
- vendored (`vendored` feature): embed the bundle in your binary and extract it on first run - no network at runtime, a single shippable binary:

```rust
let server = StackqlMcp::builder()
    .bundle_bytes(include_bytes!("../stackql-mcp-linux-x64.mcpb"))
    .start()
    .await?;
```

Bundles are published per release at [stackql/stackql](https://github.com/stackql/stackql/releases) by [packaging/mcpb](https://github.com/stackql/stackql/tree/main/packaging/mcpb), which also owns this crate. The crate version equals the stackql release it embeds (pin the minor: `stackql-mcp = "0.10"`), and its per-platform sha256 pins come from `platforms.json`, rendered by `packaging/mcpb/scripts/render-platforms.sh` from the published `.mcpb.sha256` release assets - the same manifest the npm, PyPI and other SDK wrappers use. Downloads go through `https://releases.stackql.io` with `User-Agent: stackql-mcp-server-cargo/<version>`. Platforms: linux-x64, linux-arm64, windows-x64, darwin-universal.

## Safety modes

The server enforces a safety contract per session; the crate defaults to the most restrictive.

| Mode | Allows |
|---|---|
| `Mode::ReadOnly` (default) | SELECT and metadata tools only |
| `Mode::Safe` | reads plus non-destructive mutations |
| `Mode::DeleteSafe` | safe plus deletes |
| `Mode::FullAccess` | everything, including lifecycle provisioning |

Escalation is an explicit caller opt-in via `.mode(...)`.

## Cache and overrides

The binary cache is shared with the StackQL npm and PyPI wrappers: `~/.stackql/mcp-server-bin/<version>/<platform>/`. Existing cache entries are used before any download.

Env overrides:

- `STACKQL_MCP_BIN`: path to a stackql binary to run directly (skips acquisition)
- `STACKQL_MCP_BUNDLE`: path to a local `.mcpb` to extract instead of downloading

Builder equivalents: `.binary(path)`, `.bundle_path(path)`, plus `.approot(path)` to relocate StackQL's application root (default `~/.stackql`).

If you bring your own MCP stack, `Builder::command()` returns a `std::process::Command` preloaded with the canonical launch arguments instead of starting anything.

## Demo apps

The demo apps that used to live alongside this crate (`auditron`, a terminal compliance copilot, and `stackql-agent`, a rig-based agent) now live in [stackql-labs](https://github.com/stackql-labs) and depend on the published crate. The vendored single-binary path they use is `fetch_bundle` + `include_bundle!`:

```sh
BUNDLE=$(cargo run --example fetch_bundle)
STACKQL_MCP_BUNDLE_FILE=$BUNDLE cargo build --features vendored --release
```

## Development

```sh
make cargo-manifest VERSION=X.Y.Z                  # from packaging/mcpb: render platforms.json (build.rs needs it)
cargo test                                          # unit tests
cargo test --features vendored                      # vendored path
cargo test --test conformance -- --include-ignored  # downloads the pinned bundle
cargo fmt --check && cargo clippy --all-targets --all-features -- -D warnings
python scripts/smoke-test.py --cmd "cargo run -q --manifest-path cargo/Cargo.toml --example launcher --"  # from packaging/mcpb
```

MSRV: 1.88 (set by rmcp 1.x).

## License

MIT
