//// Conformance launcher: acquire the StackQL MCP server binary (sidecar path:
//// STACKQL_MCP_BIN / STACKQL_MCP_BUNDLE, the shared cache, or a pin-verified
//// download), then run it with the canonical launch arguments, relaying stdio
//// through the BEAM byte for byte. Extra argv after `--` is forwarded to the
//// server verbatim.
////
//// This is the command packaging/mcpb/scripts/smoke-test.py --cmd drives:
////
////   python scripts/smoke-test.py --cmd "gleam run -m stackql_mcp/launcher --"

import gleam/io
import gleam/list
import stackql_mcp
import stackql_mcp/acquire
import stackql_mcp/launch
import stackql_mcp/mode
import stackql_mcp/platform

@external(erlang, "stackql_mcp_ffi", "os_family")
fn os_family() -> String

@external(erlang, "stackql_mcp_ffi", "arch")
fn arch() -> String

@external(erlang, "stackql_mcp_ffi", "home")
fn home() -> String

@external(erlang, "stackql_mcp_ffi", "plain_arguments")
fn plain_arguments() -> List(String)

@external(erlang, "stackql_mcp_ffi", "relay")
fn relay(exe: String, args: List(String)) -> Int

@external(erlang, "erlang", "halt")
fn halt(status: Int) -> Nil

pub fn main() -> Nil {
  let home = home()
  case
    acquire.ensure_binary(
      binary: Error(Nil),
      bundle_path: Error(Nil),
      home: home,
      os: os_family(),
      arch: arch(),
      getenv: stackql_mcp.getenv,
    )
  {
    Error(e) -> {
      io.println_error("stackql-mcp: " <> acquire.error_to_string(e))
      halt(1)
    }
    Ok(binary) -> {
      let args =
        list.append(
          launch.args(platform.default_approot(home), mode.ReadOnly),
          plain_arguments(),
        )
      halt(relay(binary, args))
    }
  }
}
