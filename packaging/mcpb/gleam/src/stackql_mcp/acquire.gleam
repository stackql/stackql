//// Binary acquisition: make sure a runnable stackql executable exists on
//// disk, downloading and verifying the release bundle when it does not.
////
//// Resolution order (first match wins), identical to the other StackQL
//// wrappers:
////
////   1. explicit `binary` (config)
////   2. STACKQL_MCP_BIN
////   3. explicit `bundle_path` (config), then STACKQL_MCP_BUNDLE: extract a
////      local .mcpb into <cache>/custom/<sha256[..16]>/ - no pin check, the
////      override is explicit operator intent
////   4. the shared cache <cache>/<version>/<platform-key>/stackql[.exe]
////   5. download <baseUrl>/<bundle> from pins.gleam (rendered from
////      platforms.json), verify sha256, extract the entry point into 4.
////
//// Everything effectful lives behind the Erlang FFI in stackql_mcp_ffi.erl.

import gleam/io
import gleam/result
import gleam/string
import stackql_mcp/pins
import stackql_mcp/platform

pub type AcquireError {
  /// No bundle is published for the host OS/arch.
  UnsupportedPlatform(String)
  /// An explicit binary / bundle path does not exist.
  OverrideNotFound(String)
  /// The download failed (network, HTTP status, TLS).
  DownloadFailed(String)
  /// The downloaded bundle did not match the pinned sha256.
  ChecksumMismatch(expected: String, actual: String)
  /// The .mcpb is malformed or could not be extracted.
  BundleError(String)
}

pub fn error_to_string(e: AcquireError) -> String {
  case e {
    UnsupportedPlatform(s) -> "unsupported platform: " <> s
    OverrideNotFound(s) -> "override points to a missing file: " <> s
    DownloadFailed(s) -> "download failed: " <> s
    ChecksumMismatch(expected, actual) ->
      "sha256 mismatch: expected " <> expected <> ", got " <> actual
    BundleError(s) -> "invalid .mcpb bundle: " <> s
  }
}

/// The User-Agent sent to the download proxy, per vector and version.
pub fn user_agent() -> String {
  "stackql-mcp-server-gleam/" <> pins.version
}

@external(erlang, "stackql_mcp_ffi", "file_exists")
fn file_exists(path: String) -> Bool

@external(erlang, "stackql_mcp_ffi", "read_file")
fn read_file(path: String) -> Result(BitArray, String)

@external(erlang, "stackql_mcp_ffi", "download")
fn download(url: String, user_agent: String) -> Result(BitArray, String)

@external(erlang, "stackql_mcp_ffi", "sha256_hex")
fn sha256_hex(data: BitArray) -> String

@external(erlang, "stackql_mcp_ffi", "extract_bundle")
fn extract_bundle(bundle: BitArray, dest: String) -> Result(String, String)

/// Ensure the server binary exists and return its path. `getenv` reads the
/// STACKQL_MCP_BIN / STACKQL_MCP_BUNDLE overrides.
pub fn ensure_binary(
  binary binary: Result(String, Nil),
  bundle_path bundle_path: Result(String, Nil),
  home home: String,
  os os: String,
  arch arch: String,
  getenv getenv: fn(String) -> Result(String, Nil),
) -> Result(String, AcquireError) {
  case binary, getenv("STACKQL_MCP_BIN") {
    Ok(path), _ -> existing(path)
    Error(Nil), Ok(path) -> existing(path)
    Error(Nil), Error(Nil) ->
      case bundle_path, getenv("STACKQL_MCP_BUNDLE") {
        Ok(path), _ -> local_bundle(path, home, os, arch)
        Error(Nil), Ok(path) -> local_bundle(path, home, os, arch)
        Error(Nil), Error(Nil) -> sidecar(home, os, arch)
      }
  }
}

fn existing(path: String) -> Result(String, AcquireError) {
  case file_exists(path) {
    True -> Ok(path)
    False -> Error(OverrideNotFound(path))
  }
}

fn local_bundle(
  path: String,
  home: String,
  os: String,
  arch: String,
) -> Result(String, AcquireError) {
  use key <- result.try(
    platform.resolve(os, arch) |> result.map_error(UnsupportedPlatform),
  )
  use bytes <- result.try(read_file(path) |> result.map_error(OverrideNotFound))
  let slot =
    platform.cache_root(home)
    <> "/custom/"
    <> string.slice(sha256_hex(bytes), 0, 16)
  let target = slot <> "/" <> platform.binary_name(key)
  case file_exists(target) {
    True -> Ok(target)
    False -> extract_bundle(bytes, slot) |> result.map_error(BundleError)
  }
}

fn sidecar(
  home: String,
  os: String,
  arch: String,
) -> Result(String, AcquireError) {
  use key <- result.try(
    platform.resolve(os, arch) |> result.map_error(UnsupportedPlatform),
  )
  let target = platform.binary_path(home, pins.version, key)
  case file_exists(target) {
    True -> Ok(target)
    False -> {
      use #(bundle, expected) <- result.try(
        pins.pin(platform.platform_key_to_string(key))
        |> result.replace_error(
          UnsupportedPlatform(platform.platform_key_to_string(key)),
        ),
      )
      let url = pins.base_url <> "/" <> bundle
      io.println_error(
        "stackql-mcp: downloading " <> bundle <> " (first run only) ...",
      )
      use bytes <- result.try(
        download(url, user_agent()) |> result.map_error(DownloadFailed),
      )
      let actual = sha256_hex(bytes)
      use _ <- result.try(case actual == expected {
        True -> Ok(Nil)
        False -> Error(ChecksumMismatch(expected: expected, actual: actual))
      })
      use path <- result.try(
        extract_bundle(bytes, platform.cache_dir(home, pins.version, key))
        |> result.map_error(BundleError),
      )
      io.println_error("stackql-mcp: installed " <> path)
      Ok(path)
    }
  }
}
