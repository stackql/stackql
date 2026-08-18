//// Unit tests for resolve_command: the canonical argv assembly, env-override
//// precedence, and the default approot. No binary required.

import gleam/list
import gleeunit/should
import stackql_mcp

fn no_env(_: String) -> Result(String, Nil) {
  Error(Nil)
}

pub fn resolve_uses_shared_cache_when_unset_test() {
  let cfg = stackql_mcp.default_config()
  let argv =
    stackql_mcp.resolve_command(
      config: cfg,
      home: "/home/u",
      os: "linux",
      arch: "x86_64",
      getenv: no_env,
    )

  case argv {
    Ok([binary, "mcp", "--mcp.server.type=stdio", "--approot", approot, ..]) -> {
      binary
      |> should.equal(
        "/home/u/.stackql/mcp-server-bin/"
        <> stackql_mcp.stackql_version
        <> "/linux-x64/stackql",
      )
      approot |> should.equal("/home/u/.stackql")
    }
    _ -> should.fail()
  }
}

pub fn resolve_honors_stackql_mcp_bin_env_test() {
  let getenv = fn(k) {
    case k {
      "STACKQL_MCP_BIN" -> Ok("/opt/custom/stackql")
      _ -> Error(Nil)
    }
  }
  let argv =
    stackql_mcp.resolve_command(
      config: stackql_mcp.default_config(),
      home: "/home/u",
      os: "linux",
      arch: "x86_64",
      getenv: getenv,
    )

  case argv {
    Ok([binary, ..]) -> binary |> should.equal("/opt/custom/stackql")
    _ -> should.fail()
  }
}

pub fn resolve_explicit_binary_wins_over_env_test() {
  let cfg =
    stackql_mcp.Config(
      ..stackql_mcp.default_config(),
      binary: Ok("/explicit/stackql"),
    )
  let getenv = fn(k) {
    case k {
      "STACKQL_MCP_BIN" -> Ok("/env/stackql")
      _ -> Error(Nil)
    }
  }
  let argv =
    stackql_mcp.resolve_command(
      config: cfg,
      home: "/home/u",
      os: "linux",
      arch: "x86_64",
      getenv: getenv,
    )

  case argv {
    Ok([binary, ..]) -> binary |> should.equal("/explicit/stackql")
    _ -> should.fail()
  }
}

pub fn resolve_explicit_approot_test() {
  let cfg =
    stackql_mcp.Config(
      ..stackql_mcp.default_config(),
      binary: Ok("/explicit/stackql"),
      approot: Ok("/custom/approot"),
    )
  let argv =
    stackql_mcp.resolve_command(
      config: cfg,
      home: "/home/u",
      os: "linux",
      arch: "x86_64",
      getenv: no_env,
    )

  case argv {
    Ok(args) -> list.contains(args, "/custom/approot") |> should.be_true
    _ -> should.fail()
  }
}

pub fn resolve_unsupported_platform_is_error_test() {
  case
    stackql_mcp.resolve_command(
      config: stackql_mcp.default_config(),
      home: "/home/u",
      os: "plan9",
      arch: "x86_64",
      getenv: no_env,
    )
  {
    Error(_) -> Nil
    Ok(_) -> should.fail()
  }
}
