//! Canonical launch arguments for the embedded server.
//!
//! The arg shape is the embedding contract from stackql/stackql-mcpb-packaging
//! and must stay cwd-independent:
//!
//! ```text
//! mcp --mcp.server.type=stdio --approot <home>/.stackql
//!     --mcp.config {"server": {"mode": "<mode>", "audit": {"disabled": true}}}
//! ```

use std::ffi::OsString;
use std::path::Path;

use crate::Mode;

/// Build the canonical argument vector. `auth` is appended as `--auth=<json>`
/// when present.
pub fn launch_args(mode: Mode, approot: &Path, auth: Option<&serde_json::Value>) -> Vec<OsString> {
    let config = serde_json::json!({
        "server": {
            "mode": mode.as_str(),
            "audit": {"disabled": true},
        }
    });
    let mut args: Vec<OsString> = vec![
        "mcp".into(),
        "--mcp.server.type=stdio".into(),
        "--approot".into(),
        approot.into(),
        "--mcp.config".into(),
        config.to_string().into(),
    ];
    if let Some(auth) = auth {
        args.push(format!("--auth={auth}").into());
    }
    args
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::path::PathBuf;

    fn strings(args: &[OsString]) -> Vec<String> {
        args.iter()
            .map(|a| a.to_string_lossy().into_owned())
            .collect()
    }

    #[test]
    fn canonical_args_match_the_contract() {
        let approot = PathBuf::from("/home/u/.stackql");
        let args = strings(&launch_args(Mode::ReadOnly, &approot, None));
        assert_eq!(
            args,
            vec![
                "mcp",
                "--mcp.server.type=stdio",
                "--approot",
                "/home/u/.stackql",
                "--mcp.config",
                r#"{"server":{"audit":{"disabled":true},"mode":"read_only"}}"#,
            ]
        );
    }

    #[test]
    fn mcp_config_is_valid_json_with_mode_and_audit_disabled() {
        for mode in [
            Mode::ReadOnly,
            Mode::Safe,
            Mode::DeleteSafe,
            Mode::FullAccess,
        ] {
            let args = strings(&launch_args(mode, Path::new("/tmp/approot"), None));
            let config: serde_json::Value = serde_json::from_str(&args[5]).unwrap();
            assert_eq!(config["server"]["mode"], mode.as_str());
            assert_eq!(config["server"]["audit"]["disabled"], true);
        }
    }

    #[test]
    fn auth_is_appended_as_a_single_flag() {
        let auth = serde_json::json!({"github": {"type": "null_auth"}});
        let args = strings(&launch_args(
            Mode::ReadOnly,
            Path::new("/tmp/approot"),
            Some(&auth),
        ));
        let last = args.last().unwrap();
        assert!(last.starts_with("--auth="), "{last}");
        let parsed: serde_json::Value =
            serde_json::from_str(last.strip_prefix("--auth=").unwrap()).unwrap();
        assert_eq!(parsed["github"]["type"], "null_auth");
    }
}
