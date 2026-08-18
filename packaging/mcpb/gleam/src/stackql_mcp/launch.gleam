//// Construction of the canonical StackQL MCP launch argv.
////
//// This is the embedding contract, shared verbatim across the family. The
//// argv MUST be cwd-independent (the `--approot` makes it so); this exact
//// failure was found and fixed in Claude Desktop, June 2026. Do not deviate.
////
////   mcp --mcp.server.type=stdio --approot <home>/.stackql
////       --mcp.config {"server": {"mode": "<mode>", "audit": {"disabled": true}}}

import gleam/json
import stackql_mcp/mode.{type Mode}

/// Build the argv that follows the stackql executable. The executable path
/// itself is the `command`; these are the arguments to it.
///
/// `approot` is the directory stackql treats as its home (typically
/// `<home>/.stackql`). `mode` is the server mode (default ReadOnly).
pub fn args(approot approot: String, mode mode: Mode) -> List(String) {
  [
    "mcp",
    "--mcp.server.type=stdio",
    "--approot",
    approot,
    "--mcp.config",
    config_json(mode),
  ]
}

/// The `--mcp.config` JSON blob. Audit is disabled (the embedding contract);
/// the mode is the only caller-visible knob here.
pub fn config_json(mode: Mode) -> String {
  json.object([
    #(
      "server",
      json.object([
        #("mode", json.string(mode.to_string(mode))),
        #("audit", json.object([#("disabled", json.bool(True))])),
      ]),
    ),
  ])
  |> json.to_string
}
