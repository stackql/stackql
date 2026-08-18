//! Per-platform sha256 pins for the packaged stackql release.
//!
//! Rendered from the .sha256 assets on the stackql/stackql release that the
//! packaging repo (stackql/stackql-mcpb-packaging) targets. Update this table
//! when bumping STACKQL_VERSION. Once the packaging repo publishes a
//! consolidated platforms.json release asset, prefer rendering from that.

use crate::error::{Error, Result};
use crate::platform::Platform;

/// The stackql release this crate version pins (release.yaml in the
/// packaging repo, leading v stripped).
pub const STACKQL_VERSION: &str = "0.10.601";

/// A pinned bundle: name and sha256 as published on the GitHub release.
#[derive(Clone, Copy, Debug)]
pub struct Pin {
    pub platform_key: &'static str,
    pub bundle_name: &'static str,
    pub sha256: &'static str,
}

pub const PINS: &[Pin] = &[
    Pin {
        platform_key: "linux-x64",
        bundle_name: "stackql-mcp-linux-x64.mcpb",
        sha256: "4a1cad1345fba1aae1f31269fd96aebed7a7825b38f6509466c1c995ce114e52",
    },
    Pin {
        platform_key: "linux-arm64",
        bundle_name: "stackql-mcp-linux-arm64.mcpb",
        sha256: "518b2be638fa2b31438e39ed7adafc1621d7397f6700016e9f2d516e500d3cd1",
    },
    Pin {
        platform_key: "windows-x64",
        bundle_name: "stackql-mcp-windows-x64.mcpb",
        sha256: "35ed2c66ff6aa5551b9fd7976cc6c49e370cb2b619eb2b5047c3bdbd33076eab",
    },
    Pin {
        platform_key: "darwin-universal",
        bundle_name: "stackql-mcp-darwin-universal.mcpb",
        sha256: "1cb60eba9dbc849ad9d505eb4992d0a348e4088ab51c8856b5206b7cc389995d",
    },
];

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

/// Download URL for a pinned bundle. Bundles are attached to the matching
/// stackql/stackql release.
pub fn bundle_url(pin: &Pin) -> String {
    format!(
        "https://github.com/stackql/stackql/releases/download/v{STACKQL_VERSION}/{}",
        pin.bundle_name
    )
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
    fn bundle_url_points_at_the_pinned_release() {
        let pin = pin_for(Platform::LinuxX64).unwrap();
        assert_eq!(
            bundle_url(pin),
            "https://github.com/stackql/stackql/releases/download/v0.10.601/stackql-mcp-linux-x64.mcpb"
        );
    }
}
