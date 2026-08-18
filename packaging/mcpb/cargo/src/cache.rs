//! Cache layout and environment overrides.
//!
//! The binary cache is shared with the npm and PyPI wrappers:
//! `~/.stackql/mcp-server-bin/<version>/<platform-key>/` - always check it
//! before downloading.

use std::path::PathBuf;

use crate::error::{Error, Result};

/// Path to a stackql binary to run directly, skipping acquisition entirely.
pub const ENV_BIN: &str = "STACKQL_MCP_BIN";
/// Path to a local .mcpb bundle to extract instead of downloading.
pub const ENV_BUNDLE: &str = "STACKQL_MCP_BUNDLE";

/// Resolve the user's home directory without external crates.
pub fn home_dir() -> Result<PathBuf> {
    if let Some(home) = std::env::var_os("HOME").filter(|v| !v.is_empty()) {
        return Ok(PathBuf::from(home));
    }
    if cfg!(windows) {
        if let Some(profile) = std::env::var_os("USERPROFILE").filter(|v| !v.is_empty()) {
            return Ok(PathBuf::from(profile));
        }
    }
    Err(Error::NoHomeDir)
}

/// Default approot: `<home>/.stackql`.
pub fn default_approot() -> Result<PathBuf> {
    Ok(home_dir()?.join(".stackql"))
}

/// Root of the shared binary cache: `<home>/.stackql/mcp-server-bin`.
pub fn bin_cache_root() -> Result<PathBuf> {
    Ok(default_approot()?.join("mcp-server-bin"))
}

/// Cache directory for one extracted bundle:
/// `<home>/.stackql/mcp-server-bin/<version>/<platform-key>/`.
pub fn bundle_cache_dir(version: &str, platform_key: &str) -> Result<PathBuf> {
    Ok(bin_cache_root()?.join(version).join(platform_key))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn cache_dir_matches_the_shared_layout() {
        let dir = bundle_cache_dir("0.10.500", "linux-x64").unwrap();
        let suffix: PathBuf = [".stackql", "mcp-server-bin", "0.10.500", "linux-x64"]
            .iter()
            .collect();
        assert!(
            dir.ends_with(&suffix),
            "{} should end with {}",
            dir.display(),
            suffix.display()
        );
    }
}
