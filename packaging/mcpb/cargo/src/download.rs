//! sha256 hashing, and sidecar-mode bundle download with verification.

use std::fs;
use std::io::Read;
#[cfg(feature = "sidecar")]
use std::io::Write;
use std::path::Path;

use sha2::{Digest, Sha256};

#[cfg(feature = "sidecar")]
use crate::error::Error;
use crate::error::Result;

/// Download `url` to `dest`, verifying the stream against `expected_sha256`
/// (lowercase hex). Writes to a temp file and renames into place, so `dest`
/// only ever holds a fully verified bundle.
#[cfg(feature = "sidecar")]
pub fn download_verified(url: &str, expected_sha256: &str, dest: &Path) -> Result<()> {
    if let Some(parent) = dest.parent() {
        fs::create_dir_all(parent)?;
    }
    let tmp = dest.with_extension(format!("part-{}", std::process::id()));

    let result = (|| -> Result<()> {
        let mut response = ureq::get(url).call().map_err(|e| Error::Http {
            url: url.to_string(),
            message: e.to_string(),
        })?;
        let mut reader = response.body_mut().as_reader();
        let mut file = fs::File::create(&tmp)?;

        let mut hasher = Sha256::new();
        let mut buf = [0u8; 64 * 1024];
        loop {
            let n = reader.read(&mut buf).map_err(|e| Error::Http {
                url: url.to_string(),
                message: e.to_string(),
            })?;
            if n == 0 {
                break;
            }
            hasher.update(&buf[..n]);
            file.write_all(&buf[..n])?;
        }
        file.flush()?;
        drop(file);

        let actual = hex(&hasher.finalize());
        if actual != expected_sha256 {
            return Err(Error::ChecksumMismatch {
                bundle: url.to_string(),
                expected: expected_sha256.to_string(),
                actual,
            });
        }
        fs::rename(&tmp, dest)?;
        Ok(())
    })();

    if result.is_err() {
        let _ = fs::remove_file(&tmp);
    }
    result
}

/// sha256 of a file on disk, as lowercase hex.
pub fn sha256_file(path: &Path) -> Result<String> {
    let mut file = fs::File::open(path)?;
    let mut hasher = Sha256::new();
    let mut buf = [0u8; 64 * 1024];
    loop {
        let n = file.read(&mut buf)?;
        if n == 0 {
            break;
        }
        hasher.update(&buf[..n]);
    }
    Ok(hex(&hasher.finalize()))
}

fn hex(bytes: &[u8]) -> String {
    let mut out = String::with_capacity(bytes.len() * 2);
    for b in bytes {
        out.push_str(&format!("{b:02x}"));
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn sha256_file_matches_known_vector() {
        // sha256("abc")
        let path =
            std::env::temp_dir().join(format!("stackql-mcp-test-sha-{}.txt", std::process::id()));
        fs::write(&path, b"abc").unwrap();
        assert_eq!(
            sha256_file(&path).unwrap(),
            "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
        );
        fs::remove_file(&path).unwrap();
    }
}
