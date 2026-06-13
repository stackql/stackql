# stackql-mcp-dotnet

Embedded StackQL MCP server for .NET. A NuGet library that gives .NET agentic
apps an in-process StackQL MCP server: query and provision cloud and SaaS
resources with SQL across AWS, Azure, Google, GitHub, Databricks and 40+
providers, over stdio, read-only by default.

The .NET member of the StackQL embedded-MCP family. Microsoft Agent Framework
1.0 consumes any stdio MCP server natively, so the minimum integration is just
the canonical launch args; this library is the polish layer on top - a native
builder, single-file vendoring, and a one-call bridge into Agent Framework's
tool abstraction.

## Requirements

.NET 8 or later. Both packages ship `net8.0` and `net9.0` build targets; the .NET
8 floor is set by the official C# MCP SDK (`ModelContextProtocol.Core`). The
`driftwatch` sample and the test project target `net9.0` only - that is a
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
   downloaded, verified by sha256 against pins baked into the package, extracted
   into the shared cache, and spawned over stdio.
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

> The pinned sha256 values in `src/StackQL.Mcp/pins.json` ship as placeholders
> until populated from a release's `.sha256` assets. Until then, set
> `STACKQL_MCP_BIN` or `STACKQL_MCP_BUNDLE` for local development; an
> unconfigured sidecar download fails the integrity check on purpose rather than
> running an unverified binary.

## Sample: driftwatch

[`samples/driftwatch`](samples/driftwatch) is a .NET Worker Service that embeds
the read-only StackQL server and, on a schedule, runs a suite of SQL drift
checks (public exposure, missing tags/licenses, baseline drift) and posts the
findings to a Teams Adaptive Card - each finding showing the exact SQL that
produced it. It runs against the github `null_auth` provider with zero
credentials, so it works out of the box.

## Build and test

```bash
dotnet build StackQL.Mcp.sln -c Release
dotnet test  StackQL.Mcp.sln -c Release
```

Unit tests (pin parse, cache path, launch-arg construction) run everywhere. The
conformance integration test (initialize -> tools/list -> pull github -> list
services) runs when a StackQL binary is available via `STACKQL_MCP_BUNDLE` /
`STACKQL_MCP_BIN`; CI downloads the platform bundle to activate it across Linux,
Windows, and macOS.

## License

MIT. mcp-name reference: `io.github.stackql/stackql-mcp`.
