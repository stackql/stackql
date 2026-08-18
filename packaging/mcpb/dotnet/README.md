# StackQL.Mcp (embedded StackQL MCP server for .NET)

Embedded StackQL MCP server for .NET. A NuGet library that gives .NET agentic
apps an in-process StackQL MCP server: query and provision cloud and SaaS
resources with SQL across AWS, Azure, Google, GitHub, Databricks and 40+
providers, over stdio, read-only by default.

The .NET member of the StackQL embedded-MCP family. Microsoft Agent Framework
1.0 consumes any stdio MCP server natively, so the minimum integration is just
the canonical launch args; this library is the polish layer on top - a native
builder, single-file vendoring, and a one-call bridge into Agent Framework's
tool abstraction.

Source of truth is [packaging/mcpb/dotnet](https://github.com/stackql/stackql/tree/main/packaging/mcpb/dotnet)
in stackql/stackql. The package version equals the stackql release it embeds
(pin the minor, e.g. `0.10.*`), and its per-platform sha256 pins come from
`platforms.json`, rendered by `packaging/mcpb/scripts/render-platforms.sh`
from the published `.mcpb.sha256` release assets - the same manifest the npm,
PyPI and other SDK wrappers use.

## Requirements

.NET 8 or later. Both packages ship `net8.0` and `net9.0` build targets; the .NET
8 floor is set by the official C# MCP SDK (`ModelContextProtocol.Core`). The
`Launcher` sample and the test project target `net9.0` only - that is a
sample/dev choice and does not raise the floor for consumers of the library.

## Packages

| Package | What it is |
| --- | --- |
| `StackQL.Mcp` | Core library: builder, embedded server, sidecar/vendored acquisition. Zero third-party deps beyond the official C# MCP SDK. |
| `StackQL.Mcp.AgentFramework` | One-call bridge that turns the StackQL tools into Microsoft Agent Framework / `Microsoft.Extensions.AI` tools. Separate package so the core has no Agent Framework dependency. |

## Quick start

```csharp
using StackQL.Mcp;

await using var server = await StackqlMcp.CreateBuilder()
    .WithMode(StackqlMode.ReadOnly)            // default; explicit here for clarity
    .WithAuth("github", "null_auth")           // no credentials needed for github
    .StartAsync();

var tools = await server.ListToolsAsync();
Console.WriteLine($"{tools.Count} tools available");

var result = await server.CallToolAsync("list_services",
    new() { ["provider"] = "github", ["row_limit"] = 5 });
Console.WriteLine(result.Text);
```

### Modes

`StackqlMode` controls what the server is allowed to do. The default is
`ReadOnly`; escalation is always an explicit opt-in, never a default.

| Mode | Effect |
| --- | --- |
| `ReadOnly` | Queries only (default). |
| `Safe` | Creates and updates, no deletes. |
| `DeleteSafe` | Creates, updates, and deletes the server classes as safe. |
| `FullAccess` | All operations, including unrestricted deletes. |

## Microsoft Agent Framework

`McpClientTool` (what the MCP SDK returns) derives from `AIFunction`, which is
the exact tool abstraction an Agent Framework agent consumes. So the bridge is a
thin, stable adapter:

```csharp
using StackQL.Mcp;
using StackQL.Mcp.AgentFramework;

await using var server = await StackqlMcp.CreateBuilder()
    .WithMode(StackqlMode.ReadOnly)
    .WithAuth("github", "null_auth")
    .StartAsync();

var tools = await server.AsAgentToolsAsync();

// chatClient is any IChatClient (Azure OpenAI, Anthropic, etc.)
AIAgent agent = chatClient.CreateAIAgent(
    instructions: "You answer cloud-posture questions using StackQL.",
    tools: tools.ToArray());

var reply = await agent.RunAsync("Which GitHub repos lack a license?");
```

Prefer to let Agent Framework own the StackQL process via its native
MCP-over-stdio support? Get the canonical, cwd-independent argv from us and
register it with the framework's own MCP client:

```csharp
var argv = await StackqlServer.ResolveCommandAsync(StackqlMode.ReadOnly);
// argv[0] is the executable; argv[1..] are the arguments.
```

## Acquisition modes

One API, two modes, mirroring the rest of the family.

1. **Sidecar (default).** On first run the platform's `.mcpb` bundle is
   downloaded from `https://releases.stackql.io` (User-Agent
   `stackql-mcp-server-dotnet/<version>`), verified by sha256 against the
   `platforms.json` pins embedded in the package, extracted into the shared
   cache, and spawned over stdio.
2. **Vendored.** Embed the bundle as a build resource so
   `dotnet publish -p:PublishSingleFile=true` produces a self-contained
   executable carrying the binary - the single-artifact story:

   ```bash
   dotnet build src/StackQL.Mcp/StackQL.Mcp.csproj \
     -p:StackqlVendorBundle=/path/to/stackql-mcp-windows-x64.mcpb
   ```

The binary cache is shared with the npm/pypi/go/rust/kotlin/swift wrappers at
`~/.stackql/mcp-server-bin/<version>/<platform-key>/`, so a binary fetched by any
one runtime is reused by the others.

### Overrides

| Knob | Effect |
| --- | --- |
| `WithBinary(path)` / `STACKQL_MCP_BIN` | Use an explicit binary, skip acquisition. |
| `WithBundlePath(path)` / `STACKQL_MCP_BUNDLE` | Use an explicit `.mcpb`, skip download. |
| `WithApproot(dir)` | Override the approot (default `~/.stackql`). |
| `WithCommand(argv)` | Take full control of the launch command. |

## Samples

[`samples/Launcher`](samples/Launcher) is the conformance launcher: it resolves
the server (sidecar path) and runs it with the canonical arguments and inherited
stdio, forwarding extra argv. `packaging/mcpb/scripts/smoke-test.py` drives it:

```bash
python scripts/smoke-test.py --cmd "dotnet run --project dotnet/samples/Launcher --"
```

`driftwatch`, the scheduled drift-check Worker Service demo that used to live
here, now lives in [stackql-labs](https://github.com/stackql-labs) and depends on
the published packages.

## Build and test

```bash
make dotnet-manifest VERSION=X.Y.Z   # from packaging/mcpb: render platforms.json (embedded resource, gitignored)
dotnet build StackQL.Mcp.sln -c Release
dotnet test  StackQL.Mcp.sln -c Release
```

Unit tests (pin parse, cache path, launch-arg construction) run everywhere. The
conformance integration test (initialize -> tools/list -> pull github -> list
services) runs when a StackQL binary is available via `STACKQL_MCP_BUNDLE` /
`STACKQL_MCP_BIN`; the `mcp-packaging` workflow in stackql/stackql activates it
across Linux, Windows, and macOS.

## License

MIT. mcp-name reference: `io.github.stackql/stackql-mcp`.
