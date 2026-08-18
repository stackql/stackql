//! Embedded StackQL MCP server for Rust agentic apps.
//!
//! StackQL exposes cloud providers (AWS, GitHub, Google, Azure, ...) as SQL
//! tables, served over the Model Context Protocol. This crate acquires the
//! `stackql` binary, launches it as an MCP server over stdio, and hands you a
//! connected [`rmcp`] client.
//!
//! Two acquisition modes behind one API:
//!
//! - sidecar (default feature): download the platform's .mcpb bundle at first
//!   run, verify its sha256 against pins baked into the crate, and cache it
//!   under `~/.stackql/mcp-server-bin/` (shared with the npm and PyPI
//!   wrappers)
//! - vendored (`vendored` feature): embed the .mcpb with `include_bytes!` and
//!   extract on first run - no network at runtime, single shippable binary
//!
//! ```no_run
//! use stackql_mcp::{Mode, StackqlMcp};
//!
//! # async fn run() -> Result<(), Box<dyn std::error::Error>> {
//! let server = StackqlMcp::builder()
//!     .mode(Mode::ReadOnly)
//!     .auth(serde_json::json!({"github": {"type": "null_auth"}}))
//!     .start()
//!     .await?;
//! let tools = server.list_all_tools().await?;
//! println!("{} tools available", tools.len());
//! server.shutdown().await?;
//! # Ok(())
//! # }
//! ```

mod acquire;
mod bundle;
mod cache;
mod download;
mod error;
mod launch;
mod pins;
mod platform;

use std::ops::Deref;
use std::path::PathBuf;
use std::process::Stdio;

use rmcp::service::RunningService;
use rmcp::{RoleClient, ServiceExt};

pub use cache::{ENV_BIN, ENV_BUNDLE};
pub use error::{Error, Result};
pub use pins::{Pin, PINS, STACKQL_VERSION};
pub use platform::Platform;

/// Safety contract for query / mutation / lifecycle tools, enforced
/// server-side. Maps to `server.mode` in the server's `--mcp.config`.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum Mode {
    /// SELECT and metadata tools only. The default: escalation is an
    /// explicit caller opt-in.
    #[default]
    ReadOnly,
    /// Reads plus non-destructive mutations (the server's own default).
    Safe,
    /// Safe plus deletes.
    DeleteSafe,
    /// All operations, including lifecycle provisioning.
    FullAccess,
}

impl Mode {
    /// The wire value for `server.mode`.
    pub fn as_str(self) -> &'static str {
        match self {
            Mode::ReadOnly => "read_only",
            Mode::Safe => "safe",
            Mode::DeleteSafe => "delete_safe",
            Mode::FullAccess => "full_access",
        }
    }
}

/// Download the pinned .mcpb bundle for the host platform into the shared
/// cache (verified against the baked sha256 pin) and return its path. Skips
/// the download when a verified copy is already present.
///
/// This is the producer side of vendored builds: fetch the bundle once on the
/// build machine, then embed it with [`include_bundle!`].
#[cfg(feature = "sidecar")]
pub fn fetch_bundle() -> Result<PathBuf> {
    let platform = Platform::detect()?;
    let pin = pins::pin_for(platform)?;
    let dest = cache::bin_cache_root()?
        .join(pins::STACKQL_VERSION)
        .join(pin.bundle_name);
    if dest.is_file() && download::sha256_file(&dest)? == pin.sha256 {
        return Ok(dest);
    }
    download::download_verified(&pins::bundle_url(pin), pin.sha256, &dest)?;
    Ok(dest)
}

/// Embed the .mcpb bundle named by the compile-time env var
/// `STACKQL_MCP_BUNDLE_FILE`, for use with `Builder::bundle_bytes` (vendored
/// feature):
///
/// ```ignore
/// let server = StackqlMcp::builder()
///     .bundle_bytes(stackql_mcp::include_bundle!())
///     .start()
///     .await?;
/// ```
///
/// Build with `STACKQL_MCP_BUNDLE_FILE=/abs/path/to/bundle.mcpb cargo build`.
/// Pair with [`fetch_bundle`] to produce the bundle.
#[macro_export]
macro_rules! include_bundle {
    () => {
        include_bytes!(env!(
            "STACKQL_MCP_BUNDLE_FILE",
            "set STACKQL_MCP_BUNDLE_FILE to the absolute path of the platform .mcpb bundle \
             (see stackql_mcp::fetch_bundle)"
        ))
    };
}

/// Entry point. See the crate docs for the full example.
pub struct StackqlMcp;

impl StackqlMcp {
    pub fn builder() -> Builder {
        Builder::default()
    }
}

/// Configures and starts the embedded server.
#[derive(Default)]
pub struct Builder {
    mode: Mode,
    auth: Option<serde_json::Value>,
    approot: Option<PathBuf>,
    acquisition: acquire::Acquisition,
}

impl Builder {
    /// Safety mode for the server. Defaults to [`Mode::ReadOnly`].
    pub fn mode(mut self, mode: Mode) -> Self {
        self.mode = mode;
        self
    }

    /// Provider auth document, passed to the server as `--auth=<json>`.
    /// Example: `json!({"github": {"type": "null_auth"}})`.
    pub fn auth(mut self, auth: serde_json::Value) -> Self {
        self.auth = Some(auth);
        self
    }

    /// Override the server's application root. Defaults to `<home>/.stackql`.
    pub fn approot(mut self, approot: impl Into<PathBuf>) -> Self {
        self.approot = Some(approot.into());
        self
    }

    /// Run an existing stackql binary instead of acquiring one. The
    /// `STACKQL_MCP_BIN` env var takes precedence over this.
    pub fn binary(mut self, path: impl Into<PathBuf>) -> Self {
        self.acquisition.binary = Some(path.into());
        self
    }

    /// Extract a local .mcpb bundle instead of downloading. The
    /// `STACKQL_MCP_BUNDLE` env var takes precedence over this.
    pub fn bundle_path(mut self, path: impl Into<PathBuf>) -> Self {
        self.acquisition.bundle_path = Some(path.into());
        self
    }

    /// Embed the .mcpb bundle in your binary and extract it on first run:
    /// `builder.bundle_bytes(include_bytes!("../stackql-mcp-linux-x64.mcpb"))`.
    #[cfg(feature = "vendored")]
    pub fn bundle_bytes(mut self, bytes: &'static [u8]) -> Self {
        self.acquisition.bundle_bytes = Some(bytes);
        self
    }

    /// Resolve the binary (acquiring it if needed) and return a
    /// [`std::process::Command`] preloaded with the canonical launch args.
    /// Blocking. The escape hatch for callers bringing their own MCP stack
    /// or process supervision; stdio configuration is left to the caller.
    pub fn command(&self) -> Result<std::process::Command> {
        let binary = acquire::resolve_binary(&self.acquisition)?;
        let approot = self.resolved_approot()?;
        let mut cmd = std::process::Command::new(binary);
        cmd.args(launch::launch_args(self.mode, &approot, self.auth.as_ref()));
        Ok(cmd)
    }

    /// Acquire the binary if needed, spawn the server, and complete the MCP
    /// handshake. Must be called from within a tokio runtime.
    pub async fn start(self) -> Result<RunningServer> {
        let approot = self.resolved_approot()?;
        let acquisition = self.acquisition;
        let binary = tokio::task::spawn_blocking(move || acquire::resolve_binary(&acquisition))
            .await
            .map_err(|e| Error::Mcp(format!("acquisition task failed: {e}")))??;

        let mut child = tokio::process::Command::new(&binary)
            .args(launch::launch_args(self.mode, &approot, self.auth.as_ref()))
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            // Diagnostics belong on stderr; let them flow through.
            .stderr(Stdio::inherit())
            .kill_on_drop(true)
            .spawn()
            .map_err(Error::Spawn)?;

        let stdout = child
            .stdout
            .take()
            .ok_or_else(|| Error::Mcp("child stdout not captured".into()))?;
        let stdin = child
            .stdin
            .take()
            .ok_or_else(|| Error::Mcp("child stdin not captured".into()))?;

        let client = ()
            .serve((stdout, stdin))
            .await
            .map_err(|e| Error::Mcp(format!("initialize failed: {e}")))?;

        Ok(RunningServer {
            child,
            client,
            binary,
        })
    }

    fn resolved_approot(&self) -> Result<PathBuf> {
        match &self.approot {
            Some(p) => Ok(p.clone()),
            None => cache::default_approot(),
        }
    }
}

/// A running embedded server: the child process handle plus a connected
/// rmcp client. Derefs to the client, so rmcp peer methods
/// (`list_all_tools`, `call_tool`, ...) are available directly.
pub struct RunningServer {
    child: tokio::process::Child,
    client: RunningService<RoleClient, ()>,
    binary: PathBuf,
}

impl RunningServer {
    /// The connected rmcp client.
    pub fn client(&self) -> &RunningService<RoleClient, ()> {
        &self.client
    }

    /// OS process id of the server, if it is still running.
    pub fn pid(&self) -> Option<u32> {
        self.child.id()
    }

    /// Path of the stackql binary that was launched.
    pub fn binary_path(&self) -> &std::path::Path {
        &self.binary
    }

    /// Close the MCP session and stop the server process.
    pub async fn shutdown(self) -> Result<()> {
        let RunningServer {
            mut child, client, ..
        } = self;
        // Cancelling drops the transport; the server sees EOF on stdin and
        // exits. The kill is a backstop for a wedged process.
        let _ = client.cancel().await;
        let _ = child.kill().await;
        Ok(())
    }
}

impl Deref for RunningServer {
    type Target = RunningService<RoleClient, ()>;

    fn deref(&self) -> &Self::Target {
        &self.client
    }
}
