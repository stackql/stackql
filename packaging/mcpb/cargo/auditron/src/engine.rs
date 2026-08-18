//! Scan engine: drives the embedded StackQL MCP server through a control
//! pack and streams per-control results.

use std::collections::BTreeMap;

use anyhow::{anyhow, Context, Result};
use chrono::{DateTime, SecondsFormat, Utc};
use rmcp::model::CallToolRequestParams;
use serde::Serialize;
use stackql_mcp::{Mode, RunningServer, StackqlMcp};
use tokio::sync::mpsc::UnboundedSender;

use crate::pack::{Pack, PassWhen};

/// One row of a control's result set: column -> value (stackql returns all
/// values as strings; SQL NULL arrives as the string "null").
pub type Row = BTreeMap<String, String>;

#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize)]
#[serde(rename_all = "snake_case")]
pub enum Status {
    Pass,
    Fail,
    Error,
}

#[derive(Clone, Debug, Serialize)]
pub struct ControlResult {
    pub id: String,
    pub title: String,
    pub status: Status,
    /// The exact SQL sent to the server (variables rendered).
    pub sql: String,
    /// Findings (pass_when: no_rows) or evidence rows (pass_when: rows).
    pub rows: Vec<Row>,
    /// Set when status is Error.
    pub error: Option<String>,
    pub started_at: String,
    pub finished_at: String,
    pub duration_ms: u64,
}

/// Events streamed to the UI / headless reporter as the scan progresses.
#[derive(Debug)]
pub enum Event {
    /// Server is up; identity payload from server_info.
    ServerReady {
        server_info: serde_json::Value,
    },
    ControlStarted {
        index: usize,
    },
    ControlFinished {
        index: usize,
        result: ControlResult,
    },
    /// The scan is over; the run is complete.
    Finished,
    /// The scan could not run at all (acquisition/spawn/handshake failure).
    Fatal {
        message: String,
    },
}

/// Outcome summary of a finished scan.
#[derive(Clone, Debug, Serialize)]
pub struct RunSummary {
    pub started_at: String,
    pub finished_at: String,
    pub server_info: serde_json::Value,
    pub results: Vec<ControlResult>,
}

impl RunSummary {
    pub fn counts(&self) -> (usize, usize, usize) {
        let mut counts = (0, 0, 0);
        for result in &self.results {
            match result.status {
                Status::Pass => counts.0 += 1,
                Status::Fail => counts.1 += 1,
                Status::Error => counts.2 += 1,
            }
        }
        counts
    }
}

fn now_iso() -> (DateTime<Utc>, String) {
    let now = Utc::now();
    let iso = now.to_rfc3339_opts(SecondsFormat::Secs, true);
    (now, iso)
}

/// Run the pack end to end, emitting [`Event`]s as controls complete.
/// Returns the summary (also derivable from the events) for callers that
/// just await the whole run.
pub async fn run_scan(
    pack: &Pack,
    row_limit: u32,
    events: &UnboundedSender<Event>,
) -> Result<RunSummary> {
    let (_, started_at) = now_iso();

    let server = match start_server(pack).await {
        Ok(server) => server,
        Err(e) => {
            let _ = events.send(Event::Fatal {
                message: format!("{e:#}"),
            });
            return Err(e);
        }
    };

    let server_info = call_json(&server, "server_info", serde_json::json!({}))
        .await
        .unwrap_or_else(|e| serde_json::json!({"error": e.to_string()}));
    let _ = events.send(Event::ServerReady {
        server_info: server_info.clone(),
    });

    // Install the pack's provider before the first query.
    call_json(
        &server,
        "pull_provider",
        serde_json::json!({"provider": pack.provider}),
    )
    .await
    .with_context(|| format!("pulling provider {}", pack.provider))?;

    let mut results = Vec::with_capacity(pack.controls.len());
    for (index, control) in pack.controls.iter().enumerate() {
        let _ = events.send(Event::ControlStarted { index });
        let result = run_control(&server, control, row_limit).await;
        let _ = events.send(Event::ControlFinished {
            index,
            result: result.clone(),
        });
        results.push(result);
    }

    let _ = events.send(Event::Finished);
    server.shutdown().await.ok();

    let (_, finished_at) = now_iso();
    Ok(RunSummary {
        started_at,
        finished_at,
        server_info,
        results,
    })
}

async fn start_server(pack: &Pack) -> Result<RunningServer> {
    let builder = StackqlMcp::builder()
        // Scans are point-in-time reads; the server enforces it.
        .mode(Mode::ReadOnly)
        .auth(pack.auth.clone());
    #[cfg(feature = "vendored")]
    let builder = builder.bundle_bytes(stackql_mcp::include_bundle!());
    builder
        .start()
        .await
        .context("starting the embedded stackql mcp server")
}

async fn run_control(
    server: &RunningServer,
    control: &crate::pack::Control,
    row_limit: u32,
) -> ControlResult {
    let (start, started_at) = now_iso();
    let outcome = call_json(
        server,
        "run_select_query",
        serde_json::json!({"sql": control.sql, "row_limit": row_limit}),
    )
    .await;
    let (end, finished_at) = now_iso();
    let duration_ms = (end - start).num_milliseconds().max(0) as u64;

    let base = ControlResult {
        id: control.id.clone(),
        title: control.title.clone(),
        status: Status::Error,
        sql: control.sql.clone(),
        rows: Vec::new(),
        error: None,
        started_at,
        finished_at,
        duration_ms,
    };

    match outcome {
        Ok(value) => {
            let rows: Vec<Row> = match serde_json::from_value(value["rows"].clone()) {
                Ok(rows) => rows,
                Err(e) => {
                    return ControlResult {
                        error: Some(format!("unexpected result shape: {e}")),
                        ..base
                    }
                }
            };
            let pass = match control.pass_when {
                PassWhen::NoRows => rows.is_empty(),
                PassWhen::Rows => !rows.is_empty(),
            };
            ControlResult {
                status: if pass { Status::Pass } else { Status::Fail },
                rows,
                ..base
            }
        }
        Err(e) => ControlResult {
            error: Some(format!("{e:#}")),
            ..base
        },
    }
}

/// Call a stackql MCP tool and return its structured payload. The server
/// puts the typed DTO in structuredContent and a markdown rendering in the
/// text content; prefer the former, fall back to parsing text as JSON.
async fn call_json(
    server: &RunningServer,
    tool: &str,
    arguments: serde_json::Value,
) -> Result<serde_json::Value> {
    let mut params = CallToolRequestParams::new(tool.to_string());
    params.arguments = arguments.as_object().cloned();
    let result = server
        .call_tool(params)
        .await
        .with_context(|| format!("calling tool {tool}"))?;

    // Dig payloads out via serde to stay independent of rmcp's content
    // accessor surface.
    let value = serde_json::to_value(&result)?;
    let text = value["content"][0]["text"].as_str();
    if result.is_error == Some(true) {
        return Err(anyhow!(
            "tool {tool} failed: {}",
            text.unwrap_or("(no error text)")
        ));
    }
    let structured = &value["structuredContent"];
    if !structured.is_null() {
        return Ok(structured.clone());
    }
    let text = text.ok_or_else(|| anyhow!("tool {tool} returned no content: {value}"))?;
    serde_json::from_str(text)
        .with_context(|| format!("parsing {tool} response as JSON: {text:.200}"))
}
