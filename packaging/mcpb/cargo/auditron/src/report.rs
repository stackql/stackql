//! Evidence pack output: a zip with the run manifest, the exact pack source
//! and SQL, and per-control CSVs. The point is re-runnability - an auditor
//! can re-execute the same pack and compare.

use std::collections::BTreeSet;
use std::io::Write;
use std::path::Path;

use anyhow::{Context, Result};
use sha2::{Digest, Sha256};
use zip::write::SimpleFileOptions;

use crate::engine::{ControlResult, RunSummary, Status};
use crate::pack::Pack;

/// Identity of the machine/user that collected the evidence.
#[derive(serde::Serialize)]
struct Collector {
    user: String,
    hostname: String,
    auditron_version: String,
}

fn collector() -> Collector {
    let env_any = |keys: &[&str]| {
        keys.iter()
            .find_map(|k| std::env::var(*k).ok().filter(|v| !v.is_empty()))
            .unwrap_or_else(|| "unknown".to_string())
    };
    let mut hostname = env_any(&["HOSTNAME", "COMPUTERNAME"]);
    if hostname == "unknown" {
        if let Ok(contents) = std::fs::read_to_string("/etc/hostname") {
            let trimmed = contents.trim();
            if !trimmed.is_empty() {
                hostname = trimmed.to_string();
            }
        }
    }
    Collector {
        user: env_any(&["USER", "USERNAME"]),
        hostname,
        auditron_version: env!("CARGO_PKG_VERSION").to_string(),
    }
}

/// Write the evidence zip for a finished run.
pub fn write_evidence(
    out: &Path,
    pack: &Pack,
    pack_source: &str,
    summary: &RunSummary,
    row_limit: u32,
) -> Result<()> {
    let file = std::fs::File::create(out)
        .with_context(|| format!("creating evidence pack {}", out.display()))?;
    let mut zip = zip::ZipWriter::new(file);
    let opts = SimpleFileOptions::default();

    let (passed, failed, errored) = summary.counts();
    let manifest = serde_json::json!({
        "schema": "auditron-evidence/v1",
        "run": {
            "started_at": summary.started_at,
            "finished_at": summary.finished_at,
            "mode": "read_only",
            "row_limit": row_limit,
            "passed": passed,
            "failed": failed,
            "errored": errored,
        },
        "pack": {
            "id": pack.id,
            "name": pack.name,
            "description": pack.description,
            "provider": pack.provider,
            "variables": pack.variables,
            "sha256": hex(&Sha256::digest(pack_source.as_bytes())),
        },
        "collector": collector(),
        "server": summary.server_info,
        "results": summary.results.iter().map(|r| serde_json::json!({
            "id": r.id,
            "title": r.title,
            "status": r.status,
            "rows": r.rows.len(),
            "duration_ms": r.duration_ms,
            "started_at": r.started_at,
            "finished_at": r.finished_at,
            "error": r.error,
        })).collect::<Vec<_>>(),
    });

    zip.start_file("manifest.json", opts)?;
    zip.write_all(serde_json::to_string_pretty(&manifest)?.as_bytes())?;

    // The exact pack that ran, for re-execution.
    zip.start_file("pack.yaml", opts)?;
    zip.write_all(pack_source.as_bytes())?;

    zip.start_file("summary.csv", opts)?;
    zip.write_all(summary_csv(&summary.results)?.as_bytes())?;

    for result in &summary.results {
        zip.start_file(format!("controls/{}.sql", result.id), opts)?;
        zip.write_all(result.sql.as_bytes())?;
        zip.write_all(b"\n")?;

        zip.start_file(format!("controls/{}.csv", result.id), opts)?;
        zip.write_all(rows_csv(result)?.as_bytes())?;
    }

    zip.finish()?;
    Ok(())
}

fn summary_csv(results: &[ControlResult]) -> Result<String> {
    let mut w = csv::Writer::from_writer(Vec::new());
    w.write_record([
        "id",
        "title",
        "status",
        "rows",
        "duration_ms",
        "started_at",
        "finished_at",
        "error",
    ])?;
    for r in results {
        w.write_record([
            r.id.as_str(),
            r.title.as_str(),
            status_str(r.status),
            &r.rows.len().to_string(),
            &r.duration_ms.to_string(),
            r.started_at.as_str(),
            r.finished_at.as_str(),
            r.error.as_deref().unwrap_or(""),
        ])?;
    }
    Ok(String::from_utf8(w.into_inner()?)?)
}

/// Findings/evidence rows as CSV. Header is the union of keys across rows
/// (BTreeSet keeps column order deterministic).
fn rows_csv(result: &ControlResult) -> Result<String> {
    let columns: BTreeSet<&str> = result
        .rows
        .iter()
        .flat_map(|row| row.keys().map(String::as_str))
        .collect();
    let mut w = csv::Writer::from_writer(Vec::new());
    w.write_record(&columns)?;
    for row in &result.rows {
        w.write_record(
            columns
                .iter()
                .map(|c| row.get(*c).map(String::as_str).unwrap_or("")),
        )?;
    }
    Ok(String::from_utf8(w.into_inner()?)?)
}

pub fn status_str(status: Status) -> &'static str {
    match status {
        Status::Pass => "pass",
        Status::Fail => "fail",
        Status::Error => "error",
    }
}

fn hex(bytes: &[u8]) -> String {
    bytes.iter().map(|b| format!("{b:02x}")).collect()
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::engine::Row;

    fn result_with_rows() -> ControlResult {
        let mut row1 = Row::new();
        row1.insert("name".into(), "repo-a".into());
        row1.insert("visibility".into(), "public".into());
        let mut row2 = Row::new();
        row2.insert("name".into(), "repo-b".into());
        ControlResult {
            id: "GH-003".into(),
            title: "Repos have a description".into(),
            status: Status::Fail,
            sql: "SELECT 1".into(),
            rows: vec![row1, row2],
            error: None,
            started_at: "2026-06-13T00:00:00Z".into(),
            finished_at: "2026-06-13T00:00:01Z".into(),
            duration_ms: 1000,
        }
    }

    #[test]
    fn rows_csv_uses_union_header_and_blank_gaps() {
        let csv = rows_csv(&result_with_rows()).unwrap();
        let mut lines = csv.lines();
        assert_eq!(lines.next(), Some("name,visibility"));
        assert_eq!(lines.next(), Some("repo-a,public"));
        assert_eq!(lines.next(), Some("repo-b,"));
    }

    #[test]
    fn evidence_zip_contains_expected_entries() {
        let pack = Pack::load("github-core", &[]).unwrap();
        let source = Pack::source("github-core").unwrap();
        let summary = RunSummary {
            started_at: "2026-06-13T00:00:00Z".into(),
            finished_at: "2026-06-13T00:00:05Z".into(),
            server_info: serde_json::json!({"version": "0.10.500"}),
            results: vec![result_with_rows()],
        };
        let out = std::env::temp_dir().join(format!("auditron-test-{}.zip", std::process::id()));
        write_evidence(&out, &pack, &source, &summary, 1000).unwrap();

        let mut archive = zip::ZipArchive::new(std::fs::File::open(&out).unwrap()).unwrap();
        let names: Vec<String> = (0..archive.len())
            .map(|i| archive.by_index(i).unwrap().name().to_string())
            .collect();
        for expected in [
            "manifest.json",
            "pack.yaml",
            "summary.csv",
            "controls/GH-003.sql",
            "controls/GH-003.csv",
        ] {
            assert!(
                names.contains(&expected.to_string()),
                "missing {expected} in {names:?}"
            );
        }
        std::fs::remove_file(&out).unwrap();
    }
}
