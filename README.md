# stackql-mcp-gleam

Embedded StackQL MCP server for Gleam / BEAM.

Run cloud queries and provisioning over SQL across AWS, Azure, Google, GitHub,
Databricks, and 40+ providers, from Gleam, as a supervised OTP process.

This is the Gleam member of the StackQL embedded-MCP family. It is a sidecar /
library tile: at first run it downloads the platform's StackQL bundle, verifies
it against baked-in sha256 pins, caches it in the shared family cache, and
spawns it over stdio as an MCP server (JSON-RPC 2.0). The server can run as a
plain process or, the BEAM-native way, as a supervised child of your own OTP
supervision tree.

Hex package: `stackql_mcp`. Target runtime: Erlang/BEAM only (not the JS
target) - the entire point is the BEAM concurrency and supervision story.

## Status

Early-mover / flag-planting tile for an embryonic Gleam AI ecosystem. The value
is positioning and the supervised-fleet pattern, not download volume. The
library core (sidecar resolution, canonical launch argv, supervised child spec)
and the family conformance test are in place; the `pipewatch` demo and the
extracted minimal Anthropic client are the next milestones.

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

Precedence: explicit `Config.binary` -> `STACKQL_MCP_BIN` env -> the shared
family cache at `~/.stackql/mcp-server-bin/<version>/<platform-key>/`. Set
`STACKQL_MCP_BUNDLE` to point at a `.mcpb` bundle for offline / air-gapped use.

## Development

```sh
gleam format src test
gleam check
gleam test
```

The unit tests (launch args, cache path, platform keys, resolve_command
precedence, auth rendering) run with no StackQL binary present. The conformance
test self-gates: with `STACKQL_MCP_BIN` unset it logs and passes; CI sets it
after downloading the bundle to run the live `initialize -> tools/list -> pull
github -> list_services` handshake.

## License

MIT. mcp-name reference: `io.github.stackql/stackql-mcp`.
