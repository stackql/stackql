//! Per-platform sha256 pins for the packaged stackql release.
//!
//! The pins are NOT hand-written here: `build.rs` renders them from
//! `platforms.json`, which `packaging/mcpb/scripts/render-platforms.sh` writes
//! from the published `.mcpb.sha256` release assets. The same manifest drives
//! every wrapper vector (npm, PyPI, and the other SDKs).

use crate::error::{Error, Result};
use crate::platform::Platform;

/// A pinned bundle: name and sha256 as published on the GitHub release.
#[derive(Clone, Copy, Debug)]
pub struct Pin {
    pub platform_key: &'static str,
    pub bundle_name: &'static str,
    pub sha256: &'static str,
}

// STACKQL_VERSION, BASE_URL, PINS
include!(concat!(env!("OUT_DIR"), "/pins_gen.rs"));

/// Look up the pin for a platform. Every `Platform` variant has a pin; a miss
/// here is a crate bug, so it surfaces as `UnsupportedPlatform`.
pub fn pin_for(platform: Platform) -> Result<&'static Pin> {
    PINS.iter()
        .find(|p| p.platform_key == platform.key())
        .ok_or(Error::UnsupportedPlatform {
            os: std::env::consts::OS,
            arch: std::env::consts::ARCH,
        })
}

/// Download URL for a pinned bundle: `<baseUrl>/<bundle>` from platforms.json.
pub fn bundle_url(pin: &Pin) -> String {
    format!("{BASE_URL}/{}", pin.bundle_name)
}

/// User-Agent for bundle downloads, so the download proxy can attribute
/// traffic to this vector and version.
pub fn user_agent() -> String {
    format!("stackql-mcp-server-cargo/{STACKQL_VERSION}")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn every_platform_has_a_pin() {
        for platform in [
            Platform::LinuxX64,
            Platform::LinuxArm64,
            Platform::WindowsX64,
            Platform::DarwinUniversal,
        ] {
            let pin = pin_for(platform).unwrap();
            assert_eq!(pin.platform_key, platform.key());
            assert_eq!(
                pin.bundle_name,
                format!("stackql-mcp-{}.mcpb", platform.key())
            );
        }
    }

    #[test]
    fn pins_are_well_formed_sha256_hex() {
        for pin in PINS {
            assert_eq!(pin.sha256.len(), 64, "{}", pin.bundle_name);
            assert!(
                pin.sha256.chars().all(|c| c.is_ascii_hexdigit()),
                "{}",
                pin.bundle_name
            );
            assert_eq!(pin.sha256, pin.sha256.to_lowercase());
        }
    }

    #[test]
    fn bundle_url_is_the_proxy_front_door_for_the_pinned_version() {
        let pin = pin_for(Platform::LinuxX64).unwrap();
        assert_eq!(
            bundle_url(pin),
            format!(
                "https://releases.stackql.io/stackql/{STACKQL_VERSION}/stackql-mcp-linux-x64.mcpb"
            )
        );
        assert_eq!(
            user_agent(),
            format!("stackql-mcp-server-cargo/{STACKQL_VERSION}")
        );
    }
}
