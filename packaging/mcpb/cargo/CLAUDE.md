# CLAUDE.md - stackql-mcp-rs (embedded StackQL MCP server for Rust)

## What this project is

The Rust member of the StackQL embedded-MCP family: a crate that gives Rust
agentic apps an embedded StackQL MCP server (cloud queries and provisioning
over SQL). Target repo: `stackql/stackql-mcp-rs`, published to crates.io as
`stackql-mcp` (verify name availability first; fallback `stackql-mcp-server`).
Publishing to crates.io is manual (API token), consistent with the npm/PyPI
stance.

Two acquisition modes, both behind one API:

1. Sidecar (default feature): download the platform's .mcpb at first run,
   verify sha256 against pins baked into the crate, cache, spawn over stdio
2. Vendored (`vendored` feature): caller embeds the binary with
   `include_bytes!` (we provide the macro/helper + extract-on-first-run) -
   the single-shippable-binary story for compiled agent apps

Public API sketch: `StackqlMcp::builder().mode(Mode::ReadOnly).auth(json)
.start()? -> RunningServer` exposing a child handle plus an `rmcp`
(official Rust MCP SDK) transport/client. Keep the dependency surface tiny:
rmcp, a zip crate, sha2, and an HTTP client (prefer ureq for minimalism) -
justify anything beyond that.

## The embedding contract (do not deviate)

Source of truth: stackql/stackql-mcpb-packaging (the packaging repo).

- Per-version sha256 pins from the release .sha256 assets (a consolidated
  platforms.json release asset is planned - prefer it once present); pins
  are baked at crate build/render time like npm's platforms.json
- Canonical launch args (cwd-independence mandatory):
  `mcp --mcp.server.type=stdio --approot <home>/.stackql
   --mcp.config {"server": {"mode": "<mode>", "audit": {"disabled": true}}}`
- Default `read_only`; escalation is explicit caller opt-in
- Shared binary cache: `~/.stackql/mcp-server-bin/<version>/<platform-key>/`
  (same as npm/pypi wrappers - check before downloading)
- Platform keys: linux-x64, linux-arm64, windows-x64, darwin-universal
- Env overrides honored: STACKQL_MCP_BIN, STACKQL_MCP_BUNDLE
- Conformance: packaging repo's scripts/smoke-test.py --cmd must pass
  against the crate's example launcher; port the same checks to Rust tests

## Demo app: `auditron` - a terminal compliance copilot

Business use case: compliance engineers run point-in-time control checks and
walk away with an auditor-ready evidence pack - from a TUI, no cloud console
screenshots.

A ratatui TUI + clap CLI in one binary (vendored feature - the demo IS the
single-binary pitch):

1. `auditron scan --pack cis-aws-core` - runs a YAML-defined control pack
   (id, description, SQL, pass criteria) through the embedded server in
   read_only mode; live TUI table of pass/fail/error as results stream in
2. Select a finding -> the agent (Claude via an HTTP client, or pluggable)
   explains the finding and drafts remediation steps; the SQL that produced
   the finding is always displayed
3. `auditron evidence --out evidence-2026-06.zip` - emits per-control CSVs,
   the exact SQL, timestamps, collector identity, and the run manifest -
   the re-runnable evidence pack
4. github provider in null_auth mode is the demo/test fixture (org security
   posture checks: repos without branch protection, etc.) so the demo runs
   with zero cloud credentials; AWS pack as the credentialed follow-up

Control packs live in the repo as data (controls/*.yaml) - community
extensible, and shared IP with the compliance content play.

## Build and test

- Rust stable, 2021 edition; fmt + clippy clean as CI gates
- Tests: unit (pins/extract/args), integration (spawn + initialize +
  tools/list against the github fixture), `cargo test --features vendored`
  path exercised in CI; linux+macos+windows matrix
- Examples: examples/minimal.rs (10 lines to a connected client) - the
  thing people paste from the talk

## Milestones

1. Crate core (sidecar mode) + conformance tests green on 3 OSes
2. Vendored feature + auditron demo with the github control pack, asciinema
   recording
3. crates.io publish (manual), README/docs.rs polish, announce (This Week in
   Rust, r/rust, a Rust meetup talk: "an auditor in a single binary")

## Conventions

- Plain hyphens only (no em dashes); ASCII arrows `->`
- Matter-of-fact tone; no hyperbole
- Stderr for diagnostics, stdout belongs to protocols
- MIT license; mcp-name reference: io.github.stackql/stackql-mcp
