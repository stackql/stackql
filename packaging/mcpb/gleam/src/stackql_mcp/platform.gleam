//// Platform detection and the shared binary cache path.
////
//// The cache layout is shared across the whole StackQL embedded-MCP family:
////   ~/.stackql/mcp-server-bin/<version>/<platform-key>/
//// Every family wrapper uses this SAME path, so a binary fetched by one tile
//// is reused by the next. Check here before downloading.

import gleam/result
import gleam/string

/// The platform keys used by the packaging repo and shared cache.
/// These exact strings are the cross-family contract.
pub type PlatformKey {
  LinuxX64
  LinuxArm64
  WindowsX64
  DarwinUniversal
}

pub fn platform_key_to_string(key: PlatformKey) -> String {
  case key {
    LinuxX64 -> "linux-x64"
    LinuxArm64 -> "linux-arm64"
    WindowsX64 -> "windows-x64"
    DarwinUniversal -> "darwin-universal"
  }
}

/// Resolve the platform key from an OS family string and a machine
/// architecture string (as reported by `erlang:system_info` /
/// `os:type`). Kept pure and string-driven so it is unit-testable without
/// touching the host.
///
///   os: "linux" | "darwin" | "windows"  (lowercased)
///   arch: "x86_64" | "amd64" | "aarch64" | "arm64" | ...
pub fn resolve(
  os os: String,
  arch arch: String,
) -> Result(PlatformKey, String) {
  let os = string.lowercase(os)
  let arch = string.lowercase(arch)
  case os {
    "darwin" -> Ok(DarwinUniversal)
    "windows" -> Ok(WindowsX64)
    "linux" ->
      case arch {
        "x86_64" | "amd64" -> Ok(LinuxX64)
        "aarch64" | "arm64" -> Ok(LinuxArm64)
        _ -> Error("unsupported linux architecture: " <> arch)
      }
    _ -> Error("unsupported os: " <> os)
  }
}

/// The shared cache root: `<home>/.stackql/mcp-server-bin`.
pub fn cache_root(home home: String) -> String {
  strip_trailing_slash(home) <> "/.stackql/mcp-server-bin"
}

/// The executable file name: `stackql.exe` on Windows, `stackql` elsewhere.
pub fn binary_name(key: PlatformKey) -> String {
  case key {
    WindowsX64 -> "stackql.exe"
    _ -> "stackql"
  }
}

/// The shared cache directory for a given StackQL version and platform,
/// rooted at the user's home directory. Always forward-slash joined; the
/// BEAM accepts forward slashes on Windows.
///
///   <home>/.stackql/mcp-server-bin/<version>/<platform-key>/
pub fn cache_dir(
  home home: String,
  version version: String,
  key key: PlatformKey,
) -> String {
  let home = strip_trailing_slash(home)
  home
  <> "/.stackql/mcp-server-bin/"
  <> version
  <> "/"
  <> platform_key_to_string(key)
}

/// The full path to the cached stackql executable. The executable is named
/// `stackql` on every platform except Windows, where it is `stackql.exe`.
pub fn binary_path(
  home home: String,
  version version: String,
  key key: PlatformKey,
) -> String {
  cache_dir(home, version, key) <> "/" <> binary_name(key)
}

/// The default approot the launcher points stackql at: `<home>/.stackql`.
pub fn default_approot(home home: String) -> String {
  strip_trailing_slash(home) <> "/.stackql"
}

fn strip_trailing_slash(path: String) -> String {
  case string.ends_with(path, "/") {
    True ->
      string.drop_end(path, 1)
      |> strip_trailing_slash
    False -> path
  }
}

/// Best-effort home directory from the OS environment. Honors HOME first
/// (set on unix and in most CI on Windows), then USERPROFILE (Windows).
pub fn home_from_env(
  getenv getenv: fn(String) -> Result(String, Nil),
) -> Result(String, String) {
  getenv("HOME")
  |> result.lazy_or(fn() { getenv("USERPROFILE") })
  |> result.replace_error(
    "could not determine home directory (HOME/USERPROFILE unset)",
  )
}
