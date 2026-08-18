use std::fmt;
use std::path::PathBuf;

/// Errors produced while acquiring, verifying, or running the embedded server.
#[derive(Debug)]
#[non_exhaustive]
pub enum Error {
    /// No .mcpb bundle is published for the host OS/arch.
    UnsupportedPlatform {
        os: &'static str,
        arch: &'static str,
    },
    /// The home directory could not be resolved (HOME / USERPROFILE unset).
    NoHomeDir,
    /// Downloading the bundle failed.
    Http { url: String, message: String },
    /// The downloaded bundle did not match the sha256 pin baked into the crate.
    ChecksumMismatch {
        bundle: String,
        expected: String,
        actual: String,
    },
    /// The .mcpb bundle is malformed (bad zip, missing or invalid manifest).
    Bundle(String),
    /// Filesystem error.
    Io(std::io::Error),
    /// The server process could not be spawned.
    Spawn(std::io::Error),
    /// MCP handshake or client error.
    Mcp(String),
    /// A path given via an env override or builder option does not exist.
    OverrideNotFound { what: &'static str, path: PathBuf },
}

impl fmt::Display for Error {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Error::UnsupportedPlatform { os, arch } => {
                write!(f, "no stackql mcp bundle is published for {os}/{arch}")
            }
            Error::NoHomeDir => {
                write!(
                    f,
                    "could not resolve the home directory (HOME / USERPROFILE unset)"
                )
            }
            Error::Http { url, message } => write!(f, "download of {url} failed: {message}"),
            Error::ChecksumMismatch {
                bundle,
                expected,
                actual,
            } => write!(
                f,
                "sha256 mismatch for {bundle}: expected {expected}, got {actual}"
            ),
            Error::Bundle(msg) => write!(f, "invalid .mcpb bundle: {msg}"),
            Error::Io(e) => write!(f, "io error: {e}"),
            Error::Spawn(e) => write!(f, "failed to spawn the stackql mcp server: {e}"),
            Error::Mcp(msg) => write!(f, "mcp client error: {msg}"),
            Error::OverrideNotFound { what, path } => {
                write!(
                    f,
                    "{what} points to {} which does not exist",
                    path.display()
                )
            }
        }
    }
}

impl std::error::Error for Error {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            Error::Io(e) | Error::Spawn(e) => Some(e),
            _ => None,
        }
    }
}

impl From<std::io::Error> for Error {
    fn from(e: std::io::Error) -> Self {
        Error::Io(e)
    }
}

pub type Result<T> = std::result::Result<T, Error>;
