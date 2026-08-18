//! Control pack schema and loading.
//!
//! Packs are YAML data (controls/*.yaml in the repo): id, description, SQL,
//! pass criteria per control, plus pack-level provider/auth/variables.

use std::collections::BTreeMap;
use std::path::Path;

use anyhow::{bail, Context, Result};
use serde::Deserialize;

/// The github fixture pack, embedded so the single binary carries its demo.
const GITHUB_CORE: &str = include_str!("../../controls/github-core.yaml");

#[derive(Clone, Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Pack {
    /// Schema marker; only auditron/v1 is accepted.
    pub schema: String,
    pub id: String,
    pub name: String,
    #[serde(default)]
    pub description: String,
    /// Provider to pull before running controls (e.g. "github").
    pub provider: String,
    /// Auth document handed to the embedded server as --auth.
    pub auth: serde_json::Value,
    /// Substituted into control SQL as {{name}}, overridable with --var.
    #[serde(default)]
    pub variables: BTreeMap<String, String>,
    pub controls: Vec<Control>,
}

#[derive(Clone, Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Control {
    pub id: String,
    pub title: String,
    #[serde(default)]
    pub description: String,
    #[serde(default)]
    pub remediation: Option<String>,
    pub sql: String,
    #[serde(default)]
    pub pass_when: PassWhen,
}

/// Pass criteria for a control's query result.
#[derive(Clone, Copy, Debug, Default, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum PassWhen {
    /// Pass when the query returns no rows; returned rows are findings.
    #[default]
    NoRows,
    /// Pass when the query returns rows (evidence-collection controls);
    /// returned rows are the evidence.
    Rows,
}

impl Pack {
    /// Load a pack by builtin name or filesystem path, then apply --var
    /// overrides and render the SQL.
    pub fn load(name_or_path: &str, overrides: &[(String, String)]) -> Result<Pack> {
        let yaml = if name_or_path == "github-core" {
            GITHUB_CORE.to_string()
        } else {
            let path = Path::new(name_or_path);
            std::fs::read_to_string(path)
                .with_context(|| format!("reading control pack {}", path.display()))?
        };
        Self::parse(&yaml, overrides)
    }

    /// Raw YAML for the pack (for evidence manifests).
    pub fn source(name_or_path: &str) -> Result<String> {
        if name_or_path == "github-core" {
            Ok(GITHUB_CORE.to_string())
        } else {
            std::fs::read_to_string(name_or_path)
                .with_context(|| format!("reading control pack {name_or_path}"))
        }
    }

    fn parse(yaml: &str, overrides: &[(String, String)]) -> Result<Pack> {
        let mut pack: Pack = serde_yaml::from_str(yaml).context("parsing control pack YAML")?;
        if pack.schema != "auditron/v1" {
            bail!(
                "unsupported pack schema {:?} (expected auditron/v1)",
                pack.schema
            );
        }
        if pack.controls.is_empty() {
            bail!("control pack {} has no controls", pack.id);
        }
        for (key, value) in overrides {
            if !pack.variables.contains_key(key) {
                bail!(
                    "--var {key} is not declared by pack {} (declared: {})",
                    pack.id,
                    pack.variables
                        .keys()
                        .cloned()
                        .collect::<Vec<_>>()
                        .join(", ")
                );
            }
            pack.variables.insert(key.clone(), value.clone());
        }
        for control in &mut pack.controls {
            control.sql = render(&control.sql, &pack.variables)?;
        }
        Ok(pack)
    }
}

/// Substitute {{name}} placeholders; unknown placeholders are an error so a
/// typo never reaches the server as literal SQL.
fn render(sql: &str, vars: &BTreeMap<String, String>) -> Result<String> {
    let mut out = sql.to_string();
    for (key, value) in vars {
        out = out.replace(&format!("{{{{{key}}}}}"), value);
    }
    if let Some(start) = out.find("{{") {
        let rest = &out[start..];
        let end = rest.find("}}").map(|i| i + 2).unwrap_or(rest.len());
        bail!("undeclared variable {} in control SQL", &rest[..end]);
    }
    Ok(out.trim().to_string())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn builtin_pack_parses_and_renders() {
        let pack = Pack::load("github-core", &[]).unwrap();
        assert_eq!(pack.id, "github-core");
        assert_eq!(pack.provider, "github");
        assert!(pack.controls.len() >= 5);
        for control in &pack.controls {
            assert!(
                !control.sql.contains("{{"),
                "{} still has placeholders",
                control.id
            );
        }
        assert_eq!(pack.controls.last().unwrap().pass_when, PassWhen::Rows);
    }

    #[test]
    fn var_overrides_apply() {
        let pack = Pack::load("github-core", &[("org".into(), "octocat".into())]).unwrap();
        assert!(pack.controls[1].sql.contains("org = 'octocat'"));
    }

    #[test]
    fn undeclared_var_override_is_rejected() {
        let err = Pack::load("github-core", &[("nope".into(), "x".into())]).unwrap_err();
        assert!(err.to_string().contains("not declared"), "{err}");
    }

    #[test]
    fn undeclared_placeholder_in_sql_is_rejected() {
        let yaml = r#"
schema: auditron/v1
id: t
name: T
provider: github
auth: {}
controls:
  - id: C1
    title: c
    sql: SELECT * FROM x WHERE y = '{{missing}}'
"#;
        let err = Pack::parse(yaml, &[]).unwrap_err();
        assert!(err.to_string().contains("{{missing}}"), "{err}");
    }
}
