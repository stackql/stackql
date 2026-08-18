//! auditron - terminal compliance copilot.
//!
//! Point-in-time control checks over cloud/SaaS estates via an embedded
//! StackQL MCP server, with auditor-ready evidence packs. Diagnostics go to
//! stderr; stdout carries results.

mod engine;
mod explain;
mod pack;
mod report;
mod tui;

use std::path::PathBuf;
use std::process::ExitCode;

use anyhow::{Context, Result};
use clap::{Parser, Subcommand};
use tokio::sync::mpsc;

use engine::{Event, Status};
use pack::Pack;

#[derive(Parser)]
#[command(
    name = "auditron",
    version,
    about = "Terminal compliance copilot on embedded StackQL"
)]
struct Cli {
    #[command(subcommand)]
    command: Command,
}

#[derive(Subcommand)]
enum Command {
    /// Run a control pack and watch results live.
    Scan {
        #[command(flatten)]
        run: RunArgs,
        /// Print results line by line instead of the TUI (for CI and pipes).
        #[arg(long)]
        no_tui: bool,
    },
    /// Run a control pack and write an auditor-ready evidence zip.
    Evidence {
        #[command(flatten)]
        run: RunArgs,
        /// Output path for the evidence zip.
        #[arg(long, short)]
        out: PathBuf,
    },
    /// List available control packs.
    Packs,
}

#[derive(clap::Args)]
struct RunArgs {
    /// Builtin pack name (github-core) or path to a pack YAML.
    #[arg(long, default_value = "github-core")]
    pack: String,
    /// Override a pack variable: --var org=your-org (repeatable).
    #[arg(long = "var", value_parser = parse_var)]
    vars: Vec<(String, String)>,
    /// Max rows fetched per control query.
    #[arg(long, default_value_t = 1000)]
    row_limit: u32,
}

fn parse_var(s: &str) -> Result<(String, String), String> {
    s.split_once('=')
        .map(|(k, v)| (k.trim().to_string(), v.to_string()))
        .ok_or_else(|| format!("expected key=value, got {s:?}"))
}

fn main() -> ExitCode {
    match run() {
        Ok(code) => code,
        Err(e) => {
            eprintln!("auditron: {e:#}");
            ExitCode::FAILURE
        }
    }
}

fn run() -> Result<ExitCode> {
    let cli = Cli::parse();
    match cli.command {
        Command::Packs => {
            println!("github-core (builtin) - GitHub Organization Security Posture");
            if let Ok(entries) = std::fs::read_dir("controls") {
                for entry in entries.flatten() {
                    let path = entry.path();
                    if path.extension().is_some_and(|e| e == "yaml" || e == "yml") {
                        println!("{}", path.display());
                    }
                }
            }
            Ok(ExitCode::SUCCESS)
        }
        Command::Scan { run, no_tui } => {
            let pack = Pack::load(&run.pack, &run.vars)?;
            if no_tui {
                headless(&pack, run.row_limit, None)
            } else {
                tui::run(pack, run.row_limit)
            }
        }
        Command::Evidence { run, out } => {
            let pack = Pack::load(&run.pack, &run.vars)?;
            let source = Pack::source(&run.pack)?;
            headless(&pack, run.row_limit, Some((out, source)))
        }
    }
}

/// Run the scan without a TUI, printing one line per control. With an
/// evidence target, also write the zip. Exit code: 0 all pass, 2 any
/// fail/error, 1 the scan itself could not run.
fn headless(pack: &Pack, row_limit: u32, evidence: Option<(PathBuf, String)>) -> Result<ExitCode> {
    let runtime = tokio::runtime::Runtime::new().context("starting tokio runtime")?;
    let (tx, mut rx) = mpsc::unbounded_channel();

    let summary = runtime.block_on(async {
        let scan = engine::run_scan(pack, row_limit, &tx);
        tokio::pin!(scan);
        loop {
            tokio::select! {
                event = rx.recv() => {
                    if let Some(event) = event {
                        print_event(pack, &event);
                    }
                }
                summary = &mut scan => break summary,
            }
        }
    })?;
    // Drain anything emitted between the last poll and completion.
    while let Ok(event) = rx.try_recv() {
        print_event(pack, &event);
    }

    let (passed, failed, errored) = summary.counts();
    println!(
        "{}: {passed} passed, {failed} failed, {errored} errored ({} controls)",
        pack.id,
        summary.results.len()
    );

    if let Some((out, source)) = evidence {
        report::write_evidence(&out, pack, &source, &summary, row_limit)?;
        println!("evidence pack written to {}", out.display());
    }

    Ok(if failed + errored == 0 {
        ExitCode::SUCCESS
    } else {
        ExitCode::from(2)
    })
}

fn print_event(pack: &Pack, event: &Event) {
    match event {
        Event::ServerReady { server_info } => {
            eprintln!(
                "auditron: stackql {} ready (read_only)",
                server_info["version"].as_str().unwrap_or("?")
            );
        }
        Event::ControlStarted { index } => {
            eprintln!("auditron: running {} ...", pack.controls[*index].id);
        }
        Event::ControlFinished { result, .. } => {
            let label = match result.status {
                Status::Pass => "PASS ",
                Status::Fail => "FAIL ",
                Status::Error => "ERROR",
            };
            println!(
                "[{label}] {} {} ({} rows, {} ms){}",
                result.id,
                result.title,
                result.rows.len(),
                result.duration_ms,
                result
                    .error
                    .as_deref()
                    .map(|e| format!(" - {e}"))
                    .unwrap_or_default()
            );
        }
        Event::Finished => {}
        Event::Fatal { message } => eprintln!("auditron: fatal: {message}"),
    }
}
