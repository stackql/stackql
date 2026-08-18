# stackql-agent

A Rust-native agent over your cloud and SaaS estate. It embeds the StackQL MCP server (via [stackql-mcp](https://crates.io/crates/stackql-mcp)) and wires it into a [rig](https://docs.rig.rs) agent. The agent's tools are the StackQL MCP tools; the persona is just a system prompt.

One binary becomes a platform-engineering, SRE, or audit agent by swapping `--persona`. The backend, the tools, and the read-only safety contract are identical across all three. This is the demo companion to [auditron](../auditron) (the deterministic, evidence-pack sibling) and a worked example of embedding and vendoring an MCP server into a Rust agent.

## Quickstart

Needs `ANTHROPIC_API_KEY`. The default GitHub provider runs in `null_auth` mode, so it queries public data with zero cloud credentials.

```sh
export ANTHROPIC_API_KEY=sk-ant-...

# Interactive REPL, platform-engineering persona
cargo run -p stackql-agent -- --persona platform

# One-shot
cargo run -p stackql-agent -- --persona sre \
  -p "Show the most recent failed workflow runs for stackql/stackql"

# Pre-flight: start the server, list the MCP tools, exit. No model calls.
cargo run -p stackql-agent -- --check
```

## Personas

The only difference between them is the system prompt in [src/persona.rs](src/persona.rs).

| Persona | Focus |
|---|---|
| `platform` | Platform engineering: repo and CI configuration, branch protection, golden-path conformance, drift |
| `sre` | Site reliability: workflow runs, failures and blast radius, what changed, triage |
| `audit` | Compliance: IGA and entitlements, security posture (CSPM), and cost (FinOps), with verifiable findings |

## Options

```text
--persona <platform|sre|audit>   Which system prompt to wear (default: platform)
--provider <name>                Provider to pull and query (default: github)
--auth <json>                    Provider auth document (default: github null_auth)
--prompt, -p <text>              One-shot: answer and exit
--max-turns <n>                  Max agent tool-call rounds per question (default: 20)
--check                          Pre-flight: start server, list tools, exit (no model calls)
```

To point at a credentialed provider, pass its auth document, for example:

```sh
cargo run -p stackql-agent -- --persona audit --provider aws \
  --auth '{"aws":{"type":"aws_signing_v4"}}' \
  -p "Which security groups allow 0.0.0.0/0 on port 22?"
```

## Single-binary build

Embed the StackQL server in the binary so it runs with no runtime downloads:

```sh
BUNDLE=$(cargo run -q -p stackql-mcp --example fetch_bundle)
STACKQL_MCP_BUNDLE_FILE=$BUNDLE cargo build -p stackql-agent --features vendored --release
```

The result is a self-contained binary that carries the StackQL engine inside it.

## How the wiring works

```rust
let server = StackqlMcp::builder().mode(Mode::ReadOnly).auth(auth).start().await?;
let tools  = server.list_all_tools().await?;   // the StackQL MCP tools
let sink   = server.peer().to_owned();          // the rmcp client peer

let agent = anthropic::Client::from_env()?
    .agent(anthropic::completion::CLAUDE_OPUS_4_8)
    .preamble(&persona.preamble)
    .rmcp_tools(tools, sink)                     // rig consumes rmcp directly
    .default_max_turns(20)
    .build();
```

`stackql-mcp` hands back a connected rmcp client; rig 0.38's `rmcp_tools()` consumes it as-is (both are on rmcp 1.7). See [src/main.rs](src/main.rs) for the streaming REPL that prints each tool call as the agent makes it.
