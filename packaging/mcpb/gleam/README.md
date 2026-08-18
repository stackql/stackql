# stackql_mcp (embedded StackQL MCP server for Gleam / BEAM)

Embedded StackQL MCP server for Gleam / BEAM.

Run cloud queries and provisioning over SQL across AWS, Azure, Google, GitHub,
Databricks, and 40+ providers, from Gleam, as a supervised OTP process.

This is the Gleam member of the StackQL embedded-MCP family. It is a sidecar /
library tile: at first run it downloads the platform's StackQL bundle from
`https://releases.stackql.io` (User-Agent `stackql-mcp-server-gleam/<version>`),
verifies it against the sha256 pins in `stackql_mcp/pins`, caches it in the
shared family cache, and spawns it over stdio as an MCP server (JSON-RPC 2.0). The server can run as a
plain process or, the BEAM-native way, as a supervised child of your own OTP
supervision tree.

Hex package: `stackql_mcp`. Target runtime: Erlang/BEAM only (not the JS
target) - the entire point is the BEAM concurrency and supervision story.

Source of truth is [packaging/mcpb/gleam](https://github.com/stackql/stackql/tree/main/packaging/mcpb/gleam)
in stackql/stackql. The package version equals the stackql release it embeds
(pin the minor, e.g. `>= 0.10.0 and < 0.11.0`); `stackql_mcp/pins` is generated
by `packaging/mcpb/scripts/render-platforms.sh` from `platforms.json`, the
manifest shared by the npm, PyPI and other SDK wrappers, rendered from the
published `.mcpb.sha256` release assets.

## Status

Early-mover / flag-planting tile for an embryonic Gleam AI ecosystem. The value
is positioning and the supervised-fleet pattern, not download volume. The
library core (sidecar acquisition, canonical launch argv, supervised child spec)
and the family conformance test are in place; the `pipewatch` demo lives in
[stackql-labs](https://github.com/stackql-labs) and depends on the published
package.

## Usage (simple path)

```gleam
import gleam/io
import gleam/int
import gleam/list
import envoy
import stackql_mcp

pub fn main() {
  let config =
    stackql_mcp.Config(
      ..stackql_mcp.default_config(),
      auth: [stackql_mcp.auth_for("github", "null_auth")],
    )

  let assert Ok(server) =
    stackql_mcp.start(
      config: config,
      home: "/home/u",
      os: "linux",
      arch: "x86_64",
      getenv: envoy.get,
    )

  let assert Ok(tools) = stackql_mcp.list_tools(server)
  io.println(int.to_string(list.length(tools)) <> " tools available")

  let assert Ok(result) =
    stackql_mcp.call_tool(
      server,
      "list_services",
      "{\"provider\":\"github\",\"row_limit\":5}",
    )
  io.println(result)

  stackql_mcp.stop(server)
}
```

## Usage (supervised path - the BEAM-native selling point)

`child_spec` returns an OTP child specification so the StackQL server becomes a
supervised, independently restartable child of your own supervision tree.
Address the running actor with `supervised_list_tools` / `supervised_call_tool`.

```gleam
let spec =
  stackql_mcp.child_spec(
    config: stackql_mcp.default_config(),
    home: home,
    os: os,
    arch: arch,
    getenv: envoy.get,
  )
// place `spec` under your supervisor; the started data is a Subject(Message).
```

## Modes

Default is `read_only`. Escalation to `safe` / `delete_safe` / `full_access` is
an explicit opt-in, never a default:

```gleam
stackql_mcp.Config(..stackql_mcp.default_config(), mode: stackql_mcp.safe)
```

## Binary resolution

Precedence: explicit `Config.binary` -> `STACKQL_MCP_BIN` env -> explicit
`Config.bundle_path` -> `STACKQL_MCP_BUNDLE` env (a local `.mcpb`, extracted
without a pin check) -> the shared family cache at
`~/.stackql/mcp-server-bin/<version>/<platform-key>/` -> download from the
pinned `base_url`, sha256-verified, extracted into that cache.

`stackql_mcp/launcher` is a launcher over that path with stdio relayed through
the BEAM; extra argv after `--` is forwarded to the server. It is what
`packaging/mcpb/scripts/smoke-test.py` drives (from this directory):

```sh
python ../scripts/smoke-test.py --cmd "gleam run -m stackql_mcp/launcher --"
```

## Development

```sh
make gleam-manifest VERSION=X.Y.Z   # from packaging/mcpb: render src/stackql_mcp/pins.gleam (gitignored)
gleam format src test
gleam check
gleam test
```

Requires OTP 27+ (the acquisition FFI uses the `json` module). The unit tests
(launch args, cache path, platform keys, resolve_command precedence, auth
rendering) run with no StackQL binary present. The conformance test self-gates:
with neither `STACKQL_MCP_BIN` nor `STACKQL_MCP_BUNDLE` set it logs and passes;
the `mcp-packaging` workflow in stackql/stackql sets one of them to run the live
`initialize -> tools/list -> pull github -> list_services` handshake.

## License

MIT. mcp-name reference: `io.github.stackql/stackql-mcp`.
