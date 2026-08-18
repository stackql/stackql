//! .mcpb bundle extraction.
//!
//! A .mcpb is a zip containing manifest.json and the server binary at the
//! manifest's `server.entry_point` (server/stackql or server/stackql.exe).

use std::fs;
use std::io::{Read, Seek};
use std::path::{Component, Path, PathBuf};

use serde::Deserialize;

use crate::error::{Error, Result};

#[derive(Deserialize)]
struct Manifest {
    server: ManifestServer,
}

#[derive(Deserialize)]
struct ManifestServer {
    entry_point: String,
}

/// If `dest` already holds a valid extracted bundle, return the binary path.
pub fn cached_binary(dest: &Path) -> Option<PathBuf> {
    let manifest_path = dest.join("manifest.json");
    let data = fs::read(manifest_path).ok()?;
    let manifest: Manifest = serde_json::from_slice(&data).ok()?;
    let entry = sanitize_entry_point(&manifest.server.entry_point).ok()?;
    let binary = dest.join(entry);
    binary.is_file().then_some(binary)
}

/// Extract a .mcpb from `reader` into `dest` and return the path to the
/// server binary. Extraction goes to a sibling temp dir first and is moved
/// into place with a rename, so a crash never leaves a half-populated cache
/// entry and concurrent extractors race benignly.
pub fn extract_bundle<R: Read + Seek>(reader: R, dest: &Path) -> Result<PathBuf> {
    if let Some(binary) = cached_binary(dest) {
        return Ok(binary);
    }

    let parent = dest
        .parent()
        .ok_or_else(|| Error::Bundle(format!("cache dir {} has no parent", dest.display())))?;
    fs::create_dir_all(parent)?;

    let tmp = parent.join(format!(
        ".extract-{}-{}",
        std::process::id(),
        dest.file_name()
            .and_then(|n| n.to_str())
            .unwrap_or("bundle")
    ));
    if tmp.exists() {
        fs::remove_dir_all(&tmp)?;
    }

    let result = extract_into(reader, &tmp).and_then(|entry| {
        // We only reach here when `dest` was not a valid cache at entry. If
        // something already occupies `dest`, it is either a valid cache a
        // concurrent extractor just produced (use it) or a stale/partial dir
        // from an interrupted run (replace it). Without this, `rename` fails
        // with ENOTEMPTY on a non-empty `dest` and the binary is bricked until
        // the cache is cleared by hand.
        if dest.exists() {
            if let Some(binary) = cached_binary(dest) {
                let _ = fs::remove_dir_all(&tmp);
                return Ok(binary);
            }
            fs::remove_dir_all(dest)?;
        }
        match fs::rename(&tmp, dest) {
            Ok(()) => {}
            Err(_) if cached_binary(dest).is_some() => {
                // A concurrent extractor won the race; its copy is valid.
                let _ = fs::remove_dir_all(&tmp);
            }
            Err(e) => return Err(Error::Io(e)),
        }
        Ok(dest.join(entry))
    });
    if result.is_err() {
        let _ = fs::remove_dir_all(&tmp);
    }
    result
}

/// Unzip into `dir`, validate the manifest, and return the relative
/// entry_point path.
fn extract_into<R: Read + Seek>(reader: R, dir: &Path) -> Result<PathBuf> {
    let mut archive =
        zip::ZipArchive::new(reader).map_err(|e| Error::Bundle(format!("bad zip: {e}")))?;
    archive
        .extract(dir)
        .map_err(|e| Error::Bundle(format!("extraction failed: {e}")))?;

    let manifest_data = fs::read(dir.join("manifest.json"))
        .map_err(|_| Error::Bundle("manifest.json missing from bundle".into()))?;
    let manifest: Manifest = serde_json::from_slice(&manifest_data)
        .map_err(|e| Error::Bundle(format!("manifest.json invalid: {e}")))?;
    let entry = sanitize_entry_point(&manifest.server.entry_point)?;

    let binary = dir.join(&entry);
    if !binary.is_file() {
        return Err(Error::Bundle(format!(
            "entry_point {} not found in bundle",
            manifest.server.entry_point
        )));
    }
    make_executable(&binary)?;
    Ok(entry)
}

/// Reject absolute or parent-traversing entry_point values.
fn sanitize_entry_point(entry: &str) -> Result<PathBuf> {
    let path = PathBuf::from(entry);
    let safe = !entry.is_empty() && path.components().all(|c| matches!(c, Component::Normal(_)));
    if safe {
        Ok(path)
    } else {
        Err(Error::Bundle(format!("unsafe entry_point: {entry}")))
    }
}

#[cfg(unix)]
fn make_executable(path: &Path) -> Result<()> {
    use std::os::unix::fs::PermissionsExt;
    fs::set_permissions(path, fs::Permissions::from_mode(0o755))?;
    Ok(())
}

#[cfg(not(unix))]
fn make_executable(_path: &Path) -> Result<()> {
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::{Cursor, Write};
    use zip::write::SimpleFileOptions;

    fn fake_bundle(entry_point: &str) -> Vec<u8> {
        let mut zip = zip::ZipWriter::new(Cursor::new(Vec::new()));
        let opts = SimpleFileOptions::default().compression_method(zip::CompressionMethod::Stored);
        zip.start_file("manifest.json", opts).unwrap();
        zip.write_all(format!(r#"{{"server": {{"entry_point": "{entry_point}"}}}}"#).as_bytes())
            .unwrap();
        zip.start_file("server/stackql", opts).unwrap();
        zip.write_all(b"#!/bin/sh\necho fake stackql\n").unwrap();
        zip.finish().unwrap().into_inner()
    }

    fn temp_dest(name: &str) -> PathBuf {
        let dir =
            std::env::temp_dir().join(format!("stackql-mcp-test-{}-{name}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        dir.join("bundle")
    }

    #[test]
    fn extracts_and_returns_the_entry_point() {
        let dest = temp_dest("extract");
        let binary = extract_bundle(Cursor::new(fake_bundle("server/stackql")), &dest).unwrap();
        assert_eq!(binary, dest.join("server").join("stackql"));
        assert!(binary.is_file());
        // Second call is a cache hit.
        let again = extract_bundle(Cursor::new(fake_bundle("server/stackql")), &dest).unwrap();
        assert_eq!(again, binary);
        fs::remove_dir_all(dest.parent().unwrap()).unwrap();
    }

    #[test]
    fn replaces_a_stale_invalid_cache_dir() {
        // A non-empty `dest` that is not a valid cached bundle (e.g. left by an
        // interrupted extraction) must be replaced, not cause ENOTEMPTY.
        let dest = temp_dest("stale");
        fs::create_dir_all(dest.join("server")).unwrap();
        fs::write(dest.join("partial.tmp"), b"junk").unwrap();
        assert!(
            cached_binary(&dest).is_none(),
            "precondition: dest is invalid"
        );

        let binary = extract_bundle(Cursor::new(fake_bundle("server/stackql")), &dest).unwrap();
        assert!(binary.is_file());
        assert!(
            !dest.join("partial.tmp").exists(),
            "stale contents should be gone"
        );
        fs::remove_dir_all(dest.parent().unwrap()).unwrap();
    }

    #[test]
    fn rejects_traversal_in_entry_point() {
        let dest = temp_dest("traversal");
        let err = extract_bundle(Cursor::new(fake_bundle("../../evil")), &dest).unwrap_err();
        assert!(matches!(err, Error::Bundle(_)), "{err}");
        assert!(
            !dest.exists(),
            "failed extraction must not populate the cache"
        );
        let _ = fs::remove_dir_all(dest.parent().unwrap());
    }

    #[test]
    fn missing_entry_point_is_an_error() {
        let dest = temp_dest("missing");
        let err = extract_bundle(Cursor::new(fake_bundle("server/nope")), &dest).unwrap_err();
        assert!(matches!(err, Error::Bundle(_)), "{err}");
        let _ = fs::remove_dir_all(dest.parent().unwrap());
    }

    #[cfg(unix)]
    #[test]
    fn extracted_binary_is_executable() {
        use std::os::unix::fs::PermissionsExt;
        let dest = temp_dest("exec");
        let binary = extract_bundle(Cursor::new(fake_bundle("server/stackql")), &dest).unwrap();
        let mode = fs::metadata(&binary).unwrap().permissions().mode();
        assert_eq!(mode & 0o111, 0o111, "mode was {mode:o}");
        fs::remove_dir_all(dest.parent().unwrap()).unwrap();
    }
}
