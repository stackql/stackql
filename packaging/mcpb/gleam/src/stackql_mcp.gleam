//// stackql_mcp - embedded StackQL MCP server for Gleam / BEAM.
////
//// Run cloud queries and provisioning over SQL across AWS, Azure, Google,
//// GitHub, Databricks, and 40+ providers, from Gleam, as a supervised OTP
//// process.
////
//// Two entry points:
////
////   - `start` / `list_tools` / `call_tool` / `stop` for the simple path:
////     spawn an embedded server, talk to it, shut it down.
////   - `child_spec` for the BEAM-native path: place the StackQL server under
////     your OWN supervision tree as a supervised child. This is the
////     differentiator - the server is a supervised process, not a bare port.
////
//// The server subprocess itself is owned and framed by `mcp_client` (JSON-RPC
//// 2.0 over STDIO). This module builds the canonical, cwd-independent argv and
//// wires the session.
////
//// mcp-name: io.github.stackql/stackql-mcp

import gleam/erlang/process
import gleam/list
import gleam/otp/actor
import gleam/otp/supervision
import gleam/result
import mcp_client
import mcp_client/manager
import stackql_mcp/acquire
import stackql_mcp/auth
import stackql_mcp/launch
import stackql_mcp/mode
import stackql_mcp/pins
import stackql_mcp/platform

// Re-export the user-facing types so callers need only import this module.
pub type Mode =
  mode.Mode

pub const read_only = mode.ReadOnly

pub const safe = mode.Safe

pub const delete_safe = mode.DeleteSafe

pub const full_access = mode.FullAccess

pub type Auth =
  auth.Auth

/// Configuration for an embedded StackQL MCP server.
///
///   - `mode`: server mode. Defaults to ReadOnly; escalation is an explicit
///     opt-in, never a default.
///   - `auth`: provider auth descriptors (e.g. github null_auth).
///   - `binary`: explicit path to the stackql executable. When `Error(Nil)`,
///     STACKQL_MCP_BIN is honoured, then the shared cache
///     (`<home>/.stackql/mcp-server-bin/<version>/<platform-key>/`), which
///     `start` populates on first run by downloading the platform bundle
///     from `pins.base_url` and verifying it against the pinned sha256.
///   - `bundle_path`: explicit path to a `.mcpb` bundle to extract instead of
///     downloading (STACKQL_MCP_BUNDLE is honoured when `Error(Nil)`); no
///     pin check.
///   - `approot`: the directory stackql treats as its home. When `Error(Nil)`,
///     `<home>/.stackql` is used.
///   - `version`: the stackql version used for the cache path. Defaults to
///     `stackql_version`; there is no download for any other version.
pub type Config {
  Config(
    mode: Mode,
    auth: List(Auth),
    binary: Result(String, Nil),
    bundle_path: Result(String, Nil),
    approot: Result(String, Nil),
    version: String,
  )
}

/// The stackql release this library is version-locked to (the library's own
/// version is the same number). Rendered into `stackql_mcp/pins` from
/// packaging/mcpb platforms.json, the manifest shared by every StackQL
/// wrapper.
pub const stackql_version = pins.version

/// A sensible default configuration: ReadOnly mode, no auth, cache-resolved
/// binary, default approot. Use record-update syntax to override fields:
///
///   Config(..default_config(), auth: [auth.github_null_auth()])
pub fn default_config() -> Config {
  Config(
    mode: mode.ReadOnly,
    auth: [],
    binary: Error(Nil),
    bundle_path: Error(Nil),
    approot: Error(Nil),
    version: stackql_version,
  )
}

/// Convenience: build a single provider auth descriptor.
///   stackql_mcp.auth_for("github", "null_auth")
pub fn auth_for(provider: String, kind: String) -> Auth {
  auth.for(provider, kind)
}

/// A running embedded server: the mcp_client session plus the server name we
/// registered it under.
pub opaque type Server {
  Server(client: mcp_client.Client, name: String)
}

/// A discovered tool's bare name.
pub type Tool =
  String

pub type StartError {
  /// The stackql binary could not be located on disk.
  BinaryNotFound(String)
  /// The host platform is not supported by the packaging matrix.
  UnsupportedPlatform(String)
  /// The MCP client failed to construct or register the server.
  ClientError(String)
  /// Acquiring the binary (override lookup, download, verification or
  /// extraction) failed.
  AcquireFailed(acquire.AcquireError)
}

/// Read an environment variable from the host. The pure entry points take a
/// `getenv` function so tests can substitute their own; pass this one for
/// the real environment.
@external(erlang, "stackql_mcp_ffi", "getenv")
pub fn getenv(name: String) -> Result(String, Nil)

/// The server name the embedded StackQL server is registered under in the
/// mcp_client session. Tool calls are qualified as `<name>/<tool>`.
const server_name = "stackql"

/// Resolve the canonical argv for the embedded server: the stackql executable
/// path followed by the canonical launch arguments. Pure: it names the binary
/// that would be used (override or shared-cache path) without acquiring it -
/// `start` (and `stackql_mcp/acquire.ensure_binary`) do the acquisition.
/// Exposed for callers and harnesses that want to own the process themselves
/// rather than let mcp_client spawn it.
///
/// `home` is the user's home directory (used for the default approot and the
/// cache-resolved binary path). `getenv` reads environment overrides
/// (STACKQL_MCP_BIN).
pub fn resolve_command(
  config config: Config,
  home home: String,
  os os: String,
  arch arch: String,
  getenv getenv: fn(String) -> Result(String, Nil),
) -> Result(List(String), StartError) {
  use binary <- result.try(resolve_binary(config, home, os, arch, getenv))
  let approot = case config.approot {
    Ok(a) -> a
    Error(Nil) -> platform.default_approot(home)
  }
  Ok([binary, ..launch.args(approot, config.mode)])
}

fn resolve_binary(
  config: Config,
  home: String,
  os: String,
  arch: String,
  getenv: fn(String) -> Result(String, Nil),
) -> Result(String, StartError) {
  // Precedence: explicit config.binary -> STACKQL_MCP_BIN env -> shared cache.
  case config.binary, getenv("STACKQL_MCP_BIN") {
    Ok(path), _ -> Ok(path)
    Error(Nil), Ok(path) -> Ok(path)
    Error(Nil), Error(Nil) ->
      platform.resolve(os, arch)
      |> result.map(fn(key) { platform.binary_path(home, config.version, key) })
      |> result.map_error(UnsupportedPlatform)
  }
}

/// Start an embedded StackQL MCP server and complete the initialize
/// handshake. The subprocess is spawned and owned by mcp_client.
///
/// This is the simple, non-supervised path. For the supervised path, use
/// `child_spec`.
pub fn start(
  config config: Config,
  home home: String,
  os os: String,
  arch arch: String,
  getenv getenv: fn(String) -> Result(String, Nil),
) -> Result(Server, StartError) {
  use binary <- result.try(
    acquire.ensure_binary(
      binary: config.binary,
      bundle_path: config.bundle_path,
      home: home,
      os: os,
      arch: arch,
      getenv: getenv,
    )
    |> result.map_error(AcquireFailed),
  )
  let approot = case config.approot {
    Ok(a) -> a
    Error(Nil) -> platform.default_approot(home)
  }
  register(binary, launch.args(approot, config.mode))
}

fn register(command: String, args: List(String)) -> Result(Server, StartError) {
  use client <- result.try(
    mcp_client.new()
    |> result.map_error(fn(_) { ClientError("could not create mcp_client") }),
  )
  // The value constructor lives in mcp_client/manager; mcp_client only
  // re-exports ServerConfig as a type alias, not as a value.
  let server_config =
    manager.ServerConfig(
      name: server_name,
      command: command,
      args: args,
      env: [],
      retry: manager.NoRetry,
    )
  use _ <- result.try(
    mcp_client.register(client, server_config)
    |> result.map_error(ClientError),
  )
  Ok(Server(client: client, name: server_name))
}

/// List the bare tool names the server advertises. The family conformance set
/// includes pull_provider, list_services, list_providers.
pub fn list_tools(server: Server) -> Result(List(Tool), StartError) {
  mcp_client.tools(server.client)
  |> list.map(fn(t) { t.original_name })
  |> Ok
}

/// Call a tool by its bare name with a JSON arguments string. The name is
/// qualified with the server name for mcp_client. Returns the raw JSON
/// response string.
pub fn call_tool(
  server: Server,
  tool: String,
  arguments: String,
) -> Result(String, String) {
  mcp_client.call(server.client, server.name <> "/" <> tool, arguments)
}

/// Stop the embedded server and release the subprocess.
pub fn stop(server: Server) -> Nil {
  mcp_client.stop(server.client)
}

/// Messages the supervised StackQL actor accepts. The actor owns the embedded
/// server and serializes access to it.
pub type Message {
  /// Call a tool by bare name with a JSON arguments string; the reply is the
  /// raw JSON response (or an error string).
  CallTool(
    reply: process.Subject(Result(String, String)),
    tool: String,
    arguments: String,
  )
  /// List the bare tool names the server advertises.
  ListTools(reply: process.Subject(Result(List(Tool), StartError)))
}

/// Address a supervised StackQL actor: list its tools. `timeout_ms` bounds the
/// synchronous wait.
pub fn supervised_list_tools(
  subject: process.Subject(Message),
  timeout_ms: Int,
) -> Result(List(Tool), StartError) {
  actor.call(subject, timeout_ms, ListTools)
}

/// Address a supervised StackQL actor: call a tool.
pub fn supervised_call_tool(
  subject: process.Subject(Message),
  timeout_ms: Int,
  tool: String,
  arguments: String,
) -> Result(String, String) {
  actor.call(subject, timeout_ms, fn(reply) {
    CallTool(reply: reply, tool: tool, arguments: arguments)
  })
}

/// An OTP child specification that starts an embedded StackQL server under the
/// CALLER's supervision tree. This is the BEAM-native selling point: the
/// StackQL server becomes a supervised, independently restartable child.
///
/// The started data is the actor's `Subject(Message)`; address it with
/// `supervised_list_tools` / `supervised_call_tool`.
pub fn child_spec(
  config config: Config,
  home home: String,
  os os: String,
  arch arch: String,
  getenv getenv: fn(String) -> Result(String, Nil),
) -> supervision.ChildSpecification(process.Subject(Message)) {
  supervision.worker(fn() {
    case start(config, home, os, arch, getenv) {
      Ok(server) ->
        actor.new(server)
        |> actor.on_message(handle_message)
        |> actor.start
      Error(e) -> Error(actor.InitFailed(start_error_to_string(e)))
    }
  })
}

fn handle_message(
  server: Server,
  message: Message,
) -> actor.Next(Server, Message) {
  case message {
    ListTools(reply) -> {
      process.send(reply, list_tools(server))
      actor.continue(server)
    }
    CallTool(reply, tool, arguments) -> {
      process.send(reply, call_tool(server, tool, arguments))
      actor.continue(server)
    }
  }
}

fn start_error_to_string(e: StartError) -> String {
  case e {
    BinaryNotFound(s) -> "stackql binary not found: " <> s
    UnsupportedPlatform(s) -> "unsupported platform: " <> s
    ClientError(s) -> "mcp client error: " <> s
    AcquireFailed(e) -> "acquisition failed: " <> acquire.error_to_string(e)
  }
}
