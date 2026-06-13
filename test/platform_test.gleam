//// Unit tests for platform detection and the shared cache path. The cache
//// layout is the cross-family contract; assert the exact strings.

import gleeunit/should
import stackql_mcp/platform

pub fn resolve_linux_x64_test() {
  platform.resolve(os: "Linux", arch: "x86_64")
  |> should.equal(Ok(platform.LinuxX64))
  platform.resolve(os: "linux", arch: "amd64")
  |> should.equal(Ok(platform.LinuxX64))
}

pub fn resolve_linux_arm64_test() {
  platform.resolve(os: "linux", arch: "aarch64")
  |> should.equal(Ok(platform.LinuxArm64))
  platform.resolve(os: "linux", arch: "arm64")
  |> should.equal(Ok(platform.LinuxArm64))
}

pub fn resolve_darwin_is_universal_test() {
  platform.resolve(os: "Darwin", arch: "arm64")
  |> should.equal(Ok(platform.DarwinUniversal))
  platform.resolve(os: "darwin", arch: "x86_64")
  |> should.equal(Ok(platform.DarwinUniversal))
}

pub fn resolve_windows_test() {
  platform.resolve(os: "Windows", arch: "amd64")
  |> should.equal(Ok(platform.WindowsX64))
}

pub fn resolve_unsupported_test() {
  case platform.resolve(os: "plan9", arch: "x86_64") {
    Error(_) -> Nil
    Ok(_) -> should.fail()
  }
}

pub fn platform_keys_are_the_contract_test() {
  platform.platform_key_to_string(platform.LinuxX64)
  |> should.equal("linux-x64")
  platform.platform_key_to_string(platform.LinuxArm64)
  |> should.equal("linux-arm64")
  platform.platform_key_to_string(platform.WindowsX64)
  |> should.equal("windows-x64")
  platform.platform_key_to_string(platform.DarwinUniversal)
  |> should.equal("darwin-universal")
}

pub fn cache_dir_is_shared_layout_test() {
  // The SAME path every other family wrapper uses.
  platform.cache_dir(
    home: "/home/u",
    version: "v0.7.0",
    key: platform.LinuxX64,
  )
  |> should.equal("/home/u/.stackql/mcp-server-bin/v0.7.0/linux-x64")
}

pub fn cache_dir_strips_trailing_slash_test() {
  platform.cache_dir(
    home: "/home/u/",
    version: "v0.7.0",
    key: platform.LinuxX64,
  )
  |> should.equal("/home/u/.stackql/mcp-server-bin/v0.7.0/linux-x64")
}

pub fn binary_path_windows_has_exe_test() {
  platform.binary_path(
    home: "C:/Users/u",
    version: "v0.7.0",
    key: platform.WindowsX64,
  )
  |> should.equal(
    "C:/Users/u/.stackql/mcp-server-bin/v0.7.0/windows-x64/stackql.exe",
  )
}

pub fn binary_path_unix_no_exe_test() {
  platform.binary_path(
    home: "/home/u",
    version: "v0.7.0",
    key: platform.LinuxX64,
  )
  |> should.equal("/home/u/.stackql/mcp-server-bin/v0.7.0/linux-x64/stackql")
}

pub fn default_approot_test() {
  platform.default_approot(home: "/home/u")
  |> should.equal("/home/u/.stackql")
}

pub fn home_from_env_prefers_home_test() {
  let getenv = fn(k) {
    case k {
      "HOME" -> Ok("/home/u")
      "USERPROFILE" -> Ok("C:/Users/u")
      _ -> Error(Nil)
    }
  }
  platform.home_from_env(getenv) |> should.equal(Ok("/home/u"))
}

pub fn home_from_env_falls_back_to_userprofile_test() {
  let getenv = fn(k) {
    case k {
      "USERPROFILE" -> Ok("C:/Users/u")
      _ -> Error(Nil)
    }
  }
  platform.home_from_env(getenv) |> should.equal(Ok("C:/Users/u"))
}
