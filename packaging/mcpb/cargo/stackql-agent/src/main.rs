//! stackql-agent - a Rust-native agent over your cloud estate.
//!
//! An embedded StackQL MCP server (vendored straight into this binary, or
//! downloaded as a verified sidecar) wired into a `rig` agent. The agent's
//! tools ARE the StackQL MCP tools; the persona is just a system prompt. Swap
//! the persona and the same backend becomes a platform-engineering, SRE, or
//! audit agent.
//!
//! Needs `ANTHROPIC_API_KEY`. Diagnostics and tool-call traces go to stderr;
//! the agent's answer goes to stdout.

mod persona;

use std::io::Write as _;

use anyhow::{Context, Result};
use clap::Parser;
use futures::StreamExt;
use rig::agent::{Agent, MultiTurnStreamItem};
use rig::client::{CompletionClient, ProviderClient};
use rig::providers::anthropic;
use rig::providers::anthropic::completion::CompletionModel;
use rig::streaming::{StreamedAssistantContent, StreamingPrompt};
use rmcp::model::CallToolRequestParams;
use stackql_mcp::{Mode, StackqlMcp};
use tokio::io::AsyncBufReadExt as _;

#[derive(Parser)]
#[command(
    name = "stackql-agent",
    version,
    about = "Rust-native agent over your cloud estate, on an embedded StackQL MCP server"
)]
struct Cli {
    /// Which persona to wear: platform | sre | audit. Only the system prompt
    /// changes between them.
    #[arg(long, default_value = "platform")]
    persona: String,
    /// Provider to pull and query (github runs with zero credentials).
    #[arg(long, default_value = "github")]
    provider: String,
    /// Provider auth document. Defaults to github null_auth (no credentials).
    /// Pass a JSON document to use a credentialed provider, e.g.
    /// --auth '{"aws":{"type":"aws_signing_v4"}}'.
    #[arg(long)]
    auth: Option<String>,
    /// One-shot: answer this prompt and exit, instead of the REPL.
    #[arg(long, short)]
    prompt: Option<String>,
    /// Max agent turns (tool-call rounds) per question.
    #[arg(long, default_value_t = 20)]
    max_turns: usize,
    /// Pre-flight: start the server, pull the provider, list the MCP tools the
    /// agent will get, and exit. Useful to verify a machine before a demo. No
    /// model calls, so no ANTHROPIC_API_KEY required.
    #[arg(long)]
    check: bool,
}

#[tokio::main]
async fn main() -> Result<()> {
    let cli = Cli::parse();

    let persona = persona::resolve(&cli.persona).with_context(|| {
        format!(
            "unknown persona {:?} (choose one of: {})",
            cli.persona,
            persona::KEYS.join(", ")
        )
    })?;

    let auth: serde_json::Value = match &cli.auth {
        Some(raw) => serde_json::from_str(raw).context("parsing --auth as JSON")?,
        None => serde_json::json!({ &cli.provider: { "type": "null_auth" } }),
    };

    // Start the embedded StackQL MCP server. Vendored when built with the
    // feature, otherwise downloaded and sha256-verified on first run.
    eprintln!("stackql-agent: starting embedded StackQL MCP server...");
    let builder = StackqlMcp::builder().mode(Mode::ReadOnly).auth(auth);
    #[cfg(feature = "vendored")]
    let builder = builder.bundle_bytes(stackql_mcp::include_bundle!());
    let server = builder
        .start()
        .await
        .context("starting the embedded stackql mcp server")?;

    // Pull the provider once so the first user query is instant.
    pull_provider(&server, &cli.provider).await?;

    // Hand the embedded server's MCP tools to a rig agent. This is the whole
    // integration: list_all_tools() + peer() come straight off our connected
    // rmcp client, and rig 0.38's rmcp_tools() consumes them directly.
    let tools = server.list_all_tools().await.context("listing MCP tools")?;
    eprintln!(
        "stackql-agent: {} | provider '{}' | {} tools | read_only",
        persona.title,
        cli.provider,
        tools.len()
    );
    // Pre-flight: prove the embedded server + MCP wiring works on this machine
    // without making any model calls.
    if cli.check {
        println!("MCP tools available to the agent:");
        for tool in &tools {
            println!("  - {}", tool.name);
        }
        match anthropic::Client::from_env() {
            Ok(client) => {
                let _agent = client
                    .agent(anthropic::completion::CLAUDE_OPUS_4_8)
                    .preamble(&persona.preamble)
                    .rmcp_tools(tools, server.peer().to_owned())
                    .default_max_turns(cli.max_turns)
                    .build();
                eprintln!("stackql-agent: check OK - agent built (ANTHROPIC_API_KEY present)");
            }
            Err(_) => {
                eprintln!(
                    "stackql-agent: check OK - server + {} MCP tools ready (set ANTHROPIC_API_KEY to run the agent)",
                    tools.len()
                );
            }
        }
        server.shutdown().await.ok();
        return Ok(());
    }

    let sink = server.peer().to_owned();

    let agent = anthropic::Client::from_env()
        .context("ANTHROPIC_API_KEY not set")?
        .agent(anthropic::completion::CLAUDE_OPUS_4_8)
        .preamble(&persona.preamble)
        .rmcp_tools(tools, sink)
        .default_max_turns(cli.max_turns)
        .build();

    if let Some(prompt) = cli.prompt {
        answer(&agent, &prompt, cli.max_turns).await?;
    } else {
        repl(&agent, &persona, cli.max_turns).await?;
    }

    server.shutdown().await.ok();
    Ok(())
}

/// Pull a provider into the server's local cache via the MCP tool. Read-only
/// safe (local cache only) and idempotent.
async fn pull_provider(server: &stackql_mcp::RunningServer, provider: &str) -> Result<()> {
    let mut params = CallToolRequestParams::new("pull_provider");
    params.arguments = serde_json::json!({ "provider": provider })
        .as_object()
        .cloned();
    server
        .call_tool(params)
        .await
        .with_context(|| format!("pulling provider {provider}"))?;
    Ok(())
}

async fn repl(
    agent: &Agent<CompletionModel>,
    persona: &persona::Persona,
    max_turns: usize,
) -> Result<()> {
    println!("\nstackql-agent - {}", persona.title);
    println!("Ask about your estate in plain English. Try:");
    for example in persona.examples {
        println!("  - {example}");
    }
    println!("Type 'quit' or Ctrl-D to exit.\n");

    let mut reader = tokio::io::BufReader::new(tokio::io::stdin()).lines();
    loop {
        print!("you> ");
        std::io::stdout().flush().ok();
        let Some(line) = reader.next_line().await? else {
            break;
        };
        let line = line.trim();
        if line.is_empty() {
            continue;
        }
        if matches!(line, "quit" | "exit" | ":q") {
            break;
        }
        if let Err(e) = answer(agent, line, max_turns).await {
            eprintln!("\nstackql-agent: {e:#}");
        }
    }
    Ok(())
}

/// Stream one answer. Text deltas go to stdout as they arrive; each tool the
/// agent invokes is announced on stderr so the audience sees it querying the
/// live estate.
async fn answer(agent: &Agent<CompletionModel>, prompt: &str, max_turns: usize) -> Result<()> {
    let mut stream = agent.stream_prompt(prompt).multi_turn(max_turns).await;
    while let Some(item) = stream.next().await {
        match item? {
            MultiTurnStreamItem::StreamAssistantItem(StreamedAssistantContent::Text(text)) => {
                print!("{}", text.text);
                std::io::stdout().flush().ok();
            }
            MultiTurnStreamItem::StreamAssistantItem(StreamedAssistantContent::ToolCall {
                tool_call,
                ..
            }) => {
                eprintln!(
                    "\n  \u{2192} {}({})",
                    tool_call.function.name,
                    truncate(&tool_call.function.arguments.to_string(), 200)
                );
            }
            _ => {}
        }
    }
    println!();
    Ok(())
}

fn truncate(s: &str, max: usize) -> String {
    if s.chars().count() <= max {
        return s.to_string();
    }
    let mut out: String = s.chars().take(max).collect();
    out.push_str(" ...");
    out
}
