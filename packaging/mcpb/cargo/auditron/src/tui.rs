//! Live scan TUI: a control table streaming pass/fail/error as results
//! arrive, with a detail pane showing the SQL that produced each finding and
//! agent-drafted explanations on demand.

use std::process::ExitCode;
use std::sync::Arc;
use std::time::Duration;

use anyhow::{Context, Result};
use crossterm::event::{Event as TermEvent, KeyCode, KeyEventKind};
use ratatui::layout::{Constraint, Direction, Layout, Rect};
use ratatui::style::{Color, Modifier, Style};
use ratatui::text::{Line, Span};
use ratatui::widgets::{Block, Borders, Cell, Paragraph, Row, Table, TableState, Wrap};
use ratatui::Frame;
use tokio::sync::mpsc;

use crate::engine::{self, ControlResult, Event, Status};
use crate::explain::{ClaudeExplainer, Explainer};
use crate::pack::Pack;

const SPINNER: [&str; 4] = ["|", "/", "-", "\\"];

enum ControlState {
    Pending,
    Running,
    Done(ControlResult),
}

enum ExplainState {
    None,
    Loading,
    Ready(String),
    Failed(String),
}

enum UiMsg {
    Engine(Event),
    Explain {
        index: usize,
        result: std::result::Result<String, String>,
    },
}

struct App {
    pack: Pack,
    states: Vec<ControlState>,
    explanations: Vec<ExplainState>,
    table: TableState,
    server_version: Option<String>,
    fatal: Option<String>,
    finished: bool,
    quit: bool,
    tick: usize,
    detail_scroll: u16,
}

impl App {
    fn new(pack: Pack) -> App {
        let n = pack.controls.len();
        let mut table = TableState::default();
        table.select(Some(0));
        App {
            pack,
            states: (0..n).map(|_| ControlState::Pending).collect(),
            explanations: (0..n).map(|_| ExplainState::None).collect(),
            table,
            server_version: None,
            fatal: None,
            finished: false,
            quit: false,
            tick: 0,
            detail_scroll: 0,
        }
    }

    fn selected(&self) -> usize {
        self.table.selected().unwrap_or(0)
    }

    fn apply(&mut self, msg: UiMsg) {
        match msg {
            UiMsg::Engine(Event::ServerReady { server_info }) => {
                self.server_version = server_info["version"].as_str().map(String::from);
            }
            UiMsg::Engine(Event::ControlStarted { index }) => {
                self.states[index] = ControlState::Running;
            }
            UiMsg::Engine(Event::ControlFinished { index, result }) => {
                self.states[index] = ControlState::Done(result);
            }
            UiMsg::Engine(Event::Finished) => self.finished = true,
            UiMsg::Engine(Event::Fatal { message }) => {
                self.fatal = Some(message);
                self.finished = true;
            }
            UiMsg::Explain { index, result } => {
                self.explanations[index] = match result {
                    Ok(text) => ExplainState::Ready(text),
                    Err(e) => ExplainState::Failed(e),
                };
            }
        }
    }

    fn move_selection(&mut self, delta: isize) {
        let n = self.pack.controls.len() as isize;
        let next = (self.selected() as isize + delta).rem_euclid(n);
        self.table.select(Some(next as usize));
        self.detail_scroll = 0;
    }

    fn exit_code(&self) -> ExitCode {
        if self.fatal.is_some() {
            return ExitCode::FAILURE;
        }
        let bad = self
            .states
            .iter()
            .any(|s| matches!(s, ControlState::Done(r) if r.status != Status::Pass));
        if bad {
            ExitCode::from(2)
        } else {
            ExitCode::SUCCESS
        }
    }
}

pub fn run(pack: Pack, row_limit: u32) -> Result<ExitCode> {
    let runtime = tokio::runtime::Runtime::new().context("starting tokio runtime")?;
    let (tx, mut rx) = mpsc::unbounded_channel::<UiMsg>();

    // Engine task: forward scan events into the UI channel.
    let (etx, mut erx) = mpsc::unbounded_channel::<Event>();
    let scan_pack = pack.clone();
    runtime.spawn(async move {
        let _ = engine::run_scan(&scan_pack, row_limit, &etx).await;
    });
    let fwd_tx = tx.clone();
    runtime.spawn(async move {
        while let Some(event) = erx.recv().await {
            if fwd_tx.send(UiMsg::Engine(event)).is_err() {
                break;
            }
        }
    });

    let explainer: Option<Arc<ClaudeExplainer>> = ClaudeExplainer::from_env().map(Arc::new);
    let mut app = App::new(pack);

    let mut terminal = ratatui::init();
    let loop_result = (|| -> Result<()> {
        loop {
            while let Ok(msg) = rx.try_recv() {
                app.apply(msg);
            }
            terminal.draw(|frame| draw(frame, &mut app, explainer.is_some()))?;
            app.tick = app.tick.wrapping_add(1);

            if crossterm::event::poll(Duration::from_millis(80))? {
                if let TermEvent::Key(key) = crossterm::event::read()? {
                    if key.kind != KeyEventKind::Press {
                        continue;
                    }
                    match key.code {
                        KeyCode::Char('q') | KeyCode::Esc => app.quit = true,
                        KeyCode::Up | KeyCode::Char('k') => app.move_selection(-1),
                        KeyCode::Down | KeyCode::Char('j') => app.move_selection(1),
                        KeyCode::PageUp => app.detail_scroll = app.detail_scroll.saturating_sub(5),
                        KeyCode::PageDown => {
                            app.detail_scroll = app.detail_scroll.saturating_add(5)
                        }
                        KeyCode::Char('e') => {
                            request_explanation(&runtime, &tx, &mut app, &explainer)
                        }
                        _ => {}
                    }
                }
            }
            if app.quit {
                break;
            }
        }
        Ok(())
    })();
    ratatui::restore();
    loop_result?;

    Ok(app.exit_code())
}

fn request_explanation(
    runtime: &tokio::runtime::Runtime,
    tx: &mpsc::UnboundedSender<UiMsg>,
    app: &mut App,
    explainer: &Option<Arc<ClaudeExplainer>>,
) {
    let index = app.selected();
    let Some(explainer) = explainer else {
        app.explanations[index] =
            ExplainState::Failed("set ANTHROPIC_API_KEY to enable agent explanations".into());
        return;
    };
    if matches!(
        app.explanations[index],
        ExplainState::Loading | ExplainState::Ready(_)
    ) {
        return;
    }
    let ControlState::Done(result) = &app.states[index] else {
        return;
    };
    app.explanations[index] = ExplainState::Loading;
    let explainer = Arc::clone(explainer);
    let control = app.pack.controls[index].clone();
    let result = result.clone();
    let tx = tx.clone();
    runtime.spawn_blocking(move || {
        let outcome = explainer
            .explain(&control, &result)
            .map_err(|e| format!("{e:#}"));
        let _ = tx.send(UiMsg::Explain {
            index,
            result: outcome,
        });
    });
}

fn draw(frame: &mut Frame, app: &mut App, has_explainer: bool) {
    let outer = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Length(1),
            Constraint::Length(app.pack.controls.len() as u16 + 3),
            Constraint::Min(5),
            Constraint::Length(1),
        ])
        .split(frame.area());

    draw_header(frame, app, outer[0]);
    draw_table(frame, app, outer[1]);
    draw_detail(frame, app, outer[2]);
    draw_footer(frame, app, has_explainer, outer[3]);
}

fn draw_header(frame: &mut Frame, app: &App, area: Rect) {
    let (mut pass, mut fail, mut error, mut done) = (0, 0, 0, 0);
    for state in &app.states {
        if let ControlState::Done(r) = state {
            done += 1;
            match r.status {
                Status::Pass => pass += 1,
                Status::Fail => fail += 1,
                Status::Error => error += 1,
            }
        }
    }
    let progress = if app.fatal.is_some() {
        Span::styled(
            "FATAL",
            Style::default().fg(Color::Red).add_modifier(Modifier::BOLD),
        )
    } else if app.finished {
        Span::styled("done", Style::default().fg(Color::Green))
    } else {
        Span::raw(format!(
            "{} {}/{}",
            SPINNER[app.tick % SPINNER.len()],
            done,
            app.states.len()
        ))
    };
    let line = Line::from(vec![
        Span::styled(" auditron ", Style::default().add_modifier(Modifier::BOLD)),
        Span::raw(format!(
            "| {} | stackql {} | read_only | ",
            app.pack.name,
            app.server_version.as_deref().unwrap_or("starting...")
        )),
        progress,
        Span::raw(format!("  pass {pass} fail {fail} error {error}")),
    ]);
    frame.render_widget(Paragraph::new(line), area);
}

fn status_cell(app: &App, state: &ControlState) -> Cell<'static> {
    match state {
        ControlState::Pending => Cell::from("-").style(Style::default().fg(Color::DarkGray)),
        ControlState::Running => Cell::from(SPINNER[app.tick % SPINNER.len()].to_string())
            .style(Style::default().fg(Color::Cyan)),
        ControlState::Done(r) => match r.status {
            Status::Pass => Cell::from("PASS").style(Style::default().fg(Color::Green)),
            Status::Fail => Cell::from("FAIL")
                .style(Style::default().fg(Color::Red).add_modifier(Modifier::BOLD)),
            Status::Error => Cell::from("ERROR").style(Style::default().fg(Color::Yellow)),
        },
    }
}

fn draw_table(frame: &mut Frame, app: &mut App, area: Rect) {
    let rows: Vec<Row> = app
        .pack
        .controls
        .iter()
        .zip(&app.states)
        .map(|(control, state)| {
            let (rows_text, ms_text) = match state {
                ControlState::Done(r) => {
                    (r.rows.len().to_string(), format!("{} ms", r.duration_ms))
                }
                _ => (String::new(), String::new()),
            };
            Row::new(vec![
                Cell::from(control.id.clone()),
                status_cell(app, state),
                Cell::from(control.title.clone()),
                Cell::from(rows_text),
                Cell::from(ms_text),
            ])
        })
        .collect();

    let table = Table::new(
        rows,
        [
            Constraint::Length(8),
            Constraint::Length(6),
            Constraint::Min(30),
            Constraint::Length(6),
            Constraint::Length(9),
        ],
    )
    .header(
        Row::new(vec!["id", "status", "control", "rows", "time"])
            .style(Style::default().add_modifier(Modifier::BOLD)),
    )
    .block(Block::default().borders(Borders::ALL).title("controls"))
    .row_highlight_style(Style::default().bg(Color::DarkGray))
    .highlight_symbol("> ");

    frame.render_stateful_widget(table, area, &mut app.table);
}

fn draw_detail(frame: &mut Frame, app: &App, area: Rect) {
    let index = app.selected();
    let control = &app.pack.controls[index];
    let mut lines: Vec<Line> = Vec::new();

    if let Some(fatal) = &app.fatal {
        lines.push(Line::styled(
            format!("scan failed: {fatal}"),
            Style::default().fg(Color::Red),
        ));
    }
    lines.push(Line::styled(
        format!("{} - {}", control.id, control.title),
        Style::default().add_modifier(Modifier::BOLD),
    ));
    if !control.description.is_empty() {
        lines.push(Line::raw(control.description.trim().to_string()));
    }
    lines.push(Line::raw(""));
    // The SQL that produced the finding is always displayed.
    lines.push(Line::styled("sql:", Style::default().fg(Color::Cyan)));
    for sql_line in control.sql.lines() {
        lines.push(Line::raw(format!("  {sql_line}")));
    }

    if let ControlState::Done(result) = &app.states[index] {
        if let Some(error) = &result.error {
            lines.push(Line::raw(""));
            lines.push(Line::styled(
                format!("error: {error}"),
                Style::default().fg(Color::Yellow),
            ));
        } else if !result.rows.is_empty() {
            lines.push(Line::raw(""));
            lines.push(Line::styled(
                format!("rows ({}):", result.rows.len()),
                Style::default().fg(Color::Cyan),
            ));
            for row in result.rows.iter().take(10) {
                lines.push(Line::raw(format!(
                    "  {}",
                    serde_json::to_string(row).unwrap_or_default()
                )));
            }
            if result.rows.len() > 10 {
                lines.push(Line::raw(format!("  ... {} more", result.rows.len() - 10)));
            }
        }
    }

    match &app.explanations[index] {
        ExplainState::None => {}
        ExplainState::Loading => {
            lines.push(Line::raw(""));
            lines.push(Line::styled(
                format!("{} asking the agent...", SPINNER[app.tick % SPINNER.len()]),
                Style::default().fg(Color::Magenta),
            ));
        }
        ExplainState::Ready(text) => {
            lines.push(Line::raw(""));
            lines.push(Line::styled("agent:", Style::default().fg(Color::Magenta)));
            for explain_line in text.lines() {
                lines.push(Line::raw(format!("  {explain_line}")));
            }
        }
        ExplainState::Failed(error) => {
            lines.push(Line::raw(""));
            lines.push(Line::styled(
                format!("explain failed: {error}"),
                Style::default().fg(Color::Yellow),
            ));
        }
    }

    let detail = Paragraph::new(lines)
        .block(Block::default().borders(Borders::ALL).title("detail"))
        .wrap(Wrap { trim: false })
        .scroll((app.detail_scroll, 0));
    frame.render_widget(detail, area);
}

fn draw_footer(frame: &mut Frame, app: &App, has_explainer: bool, area: Rect) {
    let explain_hint = if has_explainer {
        "e explain"
    } else if matches!(app.explanations[app.selected()], ExplainState::Failed(_)) {
        "e explain (needs ANTHROPIC_API_KEY)"
    } else {
        "e explain (set ANTHROPIC_API_KEY)"
    };
    let footer = format!(" q quit | up/down select | pgup/pgdn scroll | {explain_hint}");
    frame.render_widget(
        Paragraph::new(footer).style(Style::default().fg(Color::DarkGray)),
        area,
    );
}
