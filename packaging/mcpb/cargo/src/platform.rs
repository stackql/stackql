use crate::error::{Error, Result};

/// A platform the packaging repo publishes a .mcpb bundle for.
///
/// Keys mirror stackql/stackql-mcpb-packaging: linux-x64, linux-arm64,
/// windows-x64, darwin-universal.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Platform {
    LinuxX64,
    LinuxArm64,
    WindowsX64,
    DarwinUniversal,
}

impl Platform {
    /// Detect the host platform, or fail if no bundle is published for it.
    pub fn detect() -> Result<Self> {
        Self::from_os_arch(std::env::consts::OS, std::env::consts::ARCH)
    }

    pub(crate) fn from_os_arch(os: &'static str, arch: &'static str) -> Result<Self> {
        match (os, arch) {
            ("linux", "x86_64") => Ok(Platform::LinuxX64),
            ("linux", "aarch64") => Ok(Platform::LinuxArm64),
            ("windows", "x86_64") => Ok(Platform::WindowsX64),
            // The darwin bundle is a universal binary, so any macOS arch works.
            ("macos", _) => Ok(Platform::DarwinUniversal),
            _ => Err(Error::UnsupportedPlatform { os, arch }),
        }
    }

    /// The platform key used in bundle names and cache paths.
    pub fn key(self) -> &'static str {
        match self {
            Platform::LinuxX64 => "linux-x64",
            Platform::LinuxArm64 => "linux-arm64",
            Platform::WindowsX64 => "windows-x64",
            Platform::DarwinUniversal => "darwin-universal",
        }
    }

    /// Name of the server binary inside the bundle's server/ directory.
    pub fn binary_name(self) -> &'static str {
        match self {
            Platform::WindowsX64 => "stackql.exe",
            _ => "stackql",
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn known_platforms_map_to_keys() {
        assert_eq!(
            Platform::from_os_arch("linux", "x86_64").unwrap().key(),
            "linux-x64"
        );
        assert_eq!(
            Platform::from_os_arch("linux", "aarch64").unwrap().key(),
            "linux-arm64"
        );
        assert_eq!(
            Platform::from_os_arch("windows", "x86_64").unwrap().key(),
            "windows-x64"
        );
        assert_eq!(
            Platform::from_os_arch("macos", "aarch64").unwrap().key(),
            "darwin-universal"
        );
        assert_eq!(
            Platform::from_os_arch("macos", "x86_64").unwrap().key(),
            "darwin-universal"
        );
    }

    #[test]
    fn unsupported_platform_is_an_error() {
        assert!(Platform::from_os_arch("freebsd", "x86_64").is_err());
        assert!(Platform::from_os_arch("windows", "aarch64").is_err());
    }

    #[test]
    fn windows_binary_has_exe_suffix() {
        assert_eq!(Platform::WindowsX64.binary_name(), "stackql.exe");
        assert_eq!(Platform::LinuxX64.binary_name(), "stackql");
    }
}
