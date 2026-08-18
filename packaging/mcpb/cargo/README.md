# stackql-mcp

[![crates.io](https://img.shields.io/crates/v/stackql-mcp.svg)](https://crates.io/crates/stackql-mcp)
[![docs.rs](https://docs.rs/stackql-mcp/badge.svg)](https://docs.rs/stackql-mcp)
[![ci](https://github.com/stackql/stackql-mcp-rs/actions/workflows/ci.yml/badge.svg)](https://github.com/stackql/stackql-mcp-rs/actions/workflows/ci.yml)

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

Bundles are published per release at [stackql/stackql](https://github.com/stackql/stackql/releases) by [stackql/stackql-mcpb-packaging](https://github.com/stackql/stackql-mcpb-packaging). Platforms: linux-x64, linux-arm64, windows-x64, darwin-universal.

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

## Demo app: auditron

The repo ships `auditron`, a terminal compliance copilot built on this crate: point-in-time control checks with auditor-ready evidence packs. Control packs are YAML data under [controls/](controls/); the github pack runs unauthenticated, so it works with zero cloud credentials.

```sh
cargo run -p auditron -- scan                      # live TUI, github-core pack
cargo run -p auditron -- scan --no-tui             # line output for CI/pipes
cargo run -p auditron -- scan --var org=your-org   # point it at your org
cargo run -p auditron -- evidence --out evidence-2026-06.zip
```

The TUI streams pass/fail/error per control and always shows the SQL that produced a finding. Select a finding and press `e` to have Claude explain it and draft remediation steps (needs `ANTHROPIC_API_KEY`). The evidence zip contains the run manifest, the exact pack and SQL, and per-control CSVs - re-runnable by an auditor.

auditron is also the single-binary pitch. Build it with the server embedded:

```sh
BUNDLE=$(cargo run -p stackql-mcp --example fetch_bundle)
STACKQL_MCP_BUNDLE_FILE=$BUNDLE cargo build -p auditron --features vendored --release
```

The resulting binary (~80 MB) carries the StackQL server inside and runs on a clean machine with no downloads.

## Demo app: stackql-agent

[stackql-agent](stackql-agent) is the agentic companion: it embeds this crate and wires the StackQL MCP tools into a [rig](https://docs.rig.rs) agent. One binary becomes a platform-engineering, SRE, or audit agent by swapping a system prompt - the backend and the read-only contract are identical across all three.

```sh
export ANTHROPIC_API_KEY=sk-ant-...
cargo run -p stackql-agent -- --persona platform   # also: sre, audit
cargo run -p stackql-agent -- --check              # pre-flight, no model calls
```

It runs against public GitHub data with zero cloud credentials; point `--auth` at a credentialed provider for IGA, CSPM, FinOps, and AWS/Google/Azure. The integration is about ten lines: `server.list_all_tools()` plus `server.peer()` feed straight into rig's `rmcp_tools()`.

## Development

```sh
cargo test --workspace                              # unit tests
cargo test -p stackql-mcp --features vendored       # vendored path
cargo test -p stackql-mcp --test conformance -- --include-ignored  # downloads the pinned bundle
cargo fmt --check && cargo clippy --workspace --all-targets -- -D warnings
```

MSRV: 1.88 (set by rmcp 1.x).

## License

MIT
