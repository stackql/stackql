//// Unit tests for the canonical launch-arg construction. These run with no
//// stackql binary present - they assert the embedding contract is built
//// exactly right, including the cwd-independent --approot.

import gleam/list
import gleam/string
import gleeunit/should
import stackql_mcp/launch
import stackql_mcp/mode

pub fn args_shape_test() {
  let args = launch.args(approot: "/home/u/.stackql", mode: mode.ReadOnly)

  // The argv that follows the executable is fixed and ordered.
  case args {
    [
      "mcp",
      "--mcp.server.type=stdio",
      "--approot",
      "/home/u/.stackql",
      "--mcp.config",
      _config,
    ] -> Nil
    _ -> should.fail()
  }
}

pub fn approot_is_present_test() {
  // cwd-independence is mandatory: --approot must always be emitted.
  launch.args(approot: "/somewhere/.stackql", mode: mode.ReadOnly)
  |> list.contains("--approot")
  |> should.be_true
}

pub fn config_json_read_only_test() {
  // Default mode is read_only and audit is disabled.
  let cfg = launch.config_json(mode.ReadOnly)
  string.contains(cfg, "\"mode\":\"read_only\"") |> should.be_true
  string.contains(cfg, "\"audit\":{\"disabled\":true}") |> should.be_true
}

pub fn config_json_modes_test() {
  string.contains(launch.config_json(mode.Safe), "\"safe\"")
  |> should.be_true
  string.contains(launch.config_json(mode.DeleteSafe), "\"delete_safe\"")
  |> should.be_true
  string.contains(launch.config_json(mode.FullAccess), "\"full_access\"")
  |> should.be_true
}

pub fn mode_strings_test() {
  mode.to_string(mode.ReadOnly) |> should.equal("read_only")
  mode.to_string(mode.Safe) |> should.equal("safe")
  mode.to_string(mode.DeleteSafe) |> should.equal("delete_safe")
  mode.to_string(mode.FullAccess) |> should.equal("full_access")
}
