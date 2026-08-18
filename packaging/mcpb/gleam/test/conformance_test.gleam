//// Family conformance test.
////
//// Ports the packaging repo's scripts/smoke-test.py `--cmd` assertions to
//// gleeunit so this tile proves cross-implementation parity:
////
////   initialize
////   -> tools/list contains pull_provider / list_services / list_providers
////   -> pull github (null_auth)
////   -> list_services returns real services
////
//// This needs a real stackql binary. It is GATED: when STACKQL_MCP_BIN (or a
//// cached binary) is not available, the test logs and passes rather than
//// failing the build. CI sets STACKQL_MCP_BIN after downloading the bundle, so
//// the gate is closed there and the assertions run for real.

import envoy
import gleam/io
import gleam/list
import gleam/string
import gleeunit/should
import stackql_mcp

// Read the host facts the same way the library expects them.
@external(erlang, "stackql_mcp_ffi", "os_family")
fn os_family() -> String

@external(erlang, "stackql_mcp_ffi", "arch")
fn arch() -> String

@external(erlang, "stackql_mcp_ffi", "home")
fn home() -> String

fn binary_available() -> Bool {
  case envoy.get("STACKQL_MCP_BIN") {
    Ok(_) -> True
    Error(Nil) -> False
  }
}

pub fn conformance_test() {
  case binary_available() {
    False -> {
      io.println(
        "conformance_test: STACKQL_MCP_BIN unset, skipping live handshake "
        <> "(set it in CI after downloading the bundle to run this)",
      )
      Nil
    }
    True -> run_live()
  }
}

fn run_live() {
  let config =
    stackql_mcp.Config(..stackql_mcp.default_config(), auth: [
      stackql_mcp.auth_for("github", "null_auth"),
    ])

  let assert Ok(server) =
    stackql_mcp.start(
      config: config,
      home: home(),
      os: os_family(),
      arch: arch(),
      getenv: envoy.get,
    )

  // tools/list must advertise the family-required tools.
  let assert Ok(tools) = stackql_mcp.list_tools(server)
  list.contains(tools, "pull_provider") |> should.be_true
  list.contains(tools, "list_services") |> should.be_true
  list.contains(tools, "list_providers") |> should.be_true

  // pull github with null_auth (no credentials).
  let assert Ok(_) =
    stackql_mcp.call_tool(server, "pull_provider", "{\"provider\":\"github\"}")

  // list_services should return real github services.
  let assert Ok(services) =
    stackql_mcp.call_tool(
      server,
      "list_services",
      "{\"provider\":\"github\",\"row_limit\":5}",
    )
  string.is_empty(services) |> should.be_false

  stackql_mcp.stop(server)
}
