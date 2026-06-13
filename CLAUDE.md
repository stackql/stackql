# CLAUDE.md - stackql-mcp-dotnet (embedded StackQL MCP server for .NET)

## What this project is

The .NET member of the StackQL embedded-MCP family: a NuGet library that gives
.NET agentic apps an embedded StackQL MCP server (cloud queries and provisioning
over SQL across AWS, Azure, Google, GitHub, Databricks, and 40+ providers).
Target repo: `stackql/stackql-mcp-dotnet` (public). NuGet package id:
`StackQL.Mcp` (verify availability before first publish; fallback
`StackQL.Mcp.Server`). Publishing to NuGet is manual (API key / 2FA),
consistent with the npm/PyPI/crates stance across the family.

### Why this target, and why now (read this - it shapes the design)

Microsoft shipped **Agent Framework 1.0 GA on 2026-04-03** - the production merge
of Semantic Kernel + AutoGen into one .NET (and Python) agent SDK, with
FIRST-CLASS MCP support built in (agents discover and invoke MCP servers over
stdio natively). Semantic Kernel and AutoGen are now in maintenance mode, so the
.NET agent community is actively migrating onto Agent Framework this quarter.
Named production adopters include KPMG (audit automation) and BMW. This is an
enterprise, Azure-leaning, governance-aware audience that maps directly onto the
StackQL "approvable MCP server" trust story - and almost nobody in the
MCP-server crowd is serving it. Best-timed target in the family.

Design consequence: **the framework already consumes any stdio MCP server**, so
the minimum-viable .NET play is doc + demo (wire StackQL into Agent Framework
using the existing signed binary). This library is the POLISH layer on top - a
native builder, single-file vendoring, and a connector helper that turns the
embedded server into the framework's tool abstraction in one call. Build the
demo path first to validate the integration, then the library ergonomics.

## Acquisition modes (one API, two modes - mirror the family)

1. Sidecar (default): download the platform's `.mcpb` at first run, verify
   sha256 against pins baked into the package, cache, spawn over stdio
2. Vendored: embed the bundle as a build resource so `dotnet publish` with
   PublishSingleFile produces a self-contained executable carrying the binary
   (the single-artifact story - .NET belongs in the EMBEDDED tier, not the
   sidecar/interpreted tier, because of this)

## The embedding contract (do NOT deviate - shared across the family)

Source of truth: stackql/stackql-mcpb-packaging (the packaging repo).

- Binaries come from the release `.mcpb` bundles; per-version sha256 pins are
  published as the release's `.sha256` assets (a consolidated platforms.json
  release asset is planned - prefer it once present)
- Canonical launch args (cwd-independence is MANDATORY; MCP hosts may launch
  with cwd `/`, read-only on macOS - this exact bug was found and fixed in
  Claude Desktop June 2026):
  `mcp --mcp.server.type=stdio --approot <home>/.stackql
   --mcp.config {"server": {"mode": "<mode>", "audit": {"disabled": true}}}`
- Default mode: `read_only`. Escalation to safe/delete_safe/full_access is an
  explicit caller opt-in, never a default
- Shared binary cache: `~/.stackql/mcp-server-bin/<version>/<platform-key>/`
  (the SAME path the npm/pypi/go/rust/kotlin/swift wrappers use - check before
  downloading/extracting; cross-runtime cache reuse is a feature)
- Platform keys: linux-x64, linux-arm64, windows-x64, darwin-universal
- Env overrides honored: STACKQL_MCP_BIN, STACKQL_MCP_BUNDLE
- Conformance: the packaging repo's scripts/smoke-test.py `--cmd` mode must pass
  against any launcher this package produces; port the same checks as an xUnit
  integration test (initialize -> tools/list contains pull_provider/
  list_services/list_providers -> pull github (null_auth) -> list_services
  returns real services)

## Public API surface (target shape - match the family's builder idiom)

Mirror the Rust/Kotlin builder feel, idiomatic C# (async, IAsyncDisposable):

```csharp
using StackQL.Mcp;

await using var server = await StackqlMcp.CreateBuilder()
    .WithMode(StackqlMode.ReadOnly)            // default; explicit here for clarity
    .WithAuth("github", "null_auth")
    .StartAsync();

var tools = await server.ListToolsAsync();
Console.WriteLine($"{tools.Count} tools available");

var result = await server.CallToolAsync("list_services",
    new() { ["provider"] = "github", ["row_limit"] = 5 });
Console.WriteLine(result.Text);
```

Types:
- `StackqlMcp` (static `CreateBuilder()`), `StackqlMcpBuilder`
  (`.WithMode/.WithAuth/.WithBinary/.WithBundlePath/.WithApproot/.WithCommand/.StartAsync`)
- `StackqlMode` enum: `ReadOnly` (default), `Safe`, `DeleteSafe`, `FullAccess`
- `StackqlServer : IAsyncDisposable` with `ListToolsAsync`, `CallToolAsync`,
  and a `Client`/transport handle exposing the official C# MCP SDK client for
  advanced use
- `StackqlServer.ResolveCommand(...)` static helper that returns the argv an
  external harness (or Agent Framework's own MCP client) should spawn - so
  users who prefer to let the framework own the process can still get the
  canonical args from us

Build on the official C# MCP SDK (the `ModelContextProtocol` package,
Microsoft + Anthropic). Keep dependencies minimal: the MCP SDK + a zip reader
(System.IO.Compression is in-box) + sha (System.Security.Cryptography in-box).
HTTP via HttpClient (in-box). Aim for ZERO third-party deps beyond the MCP SDK.

### Agent Framework connector helper (the headline feature)

Provide a one-call bridge so the migrating Agent Framework crowd gets StackQL
tools into an agent with no MCP plumbing:

```csharp
// returns the StackQL tools as Agent Framework / Microsoft.Extensions.AI tools
var tools = await server.AsAgentToolsAsync();
// or a convenience that builds an AIAgent wired to StackQL:
//   StackqlAgent.Create(chatClient, mode: ReadOnly, auth: ...)
```

Research the exact Agent Framework tool/IChatClient tool-registration surface at
build time (it is new and moving) and match it. If a clean adapter is not yet
stable, ship `ResolveCommand` + a documented snippet wiring StackQL via the
framework's native MCP-stdio support, and add the adapter when the API settles.
Do not block the library on an unstable adapter.

## Demo app: `driftwatch` - cloud drift sentinel for .NET shops

Business use case: platform teams want to know the instant real cloud state
drifts from intended state - new public exposure, untagged spend, config that
no longer matches policy - posted where they already work (Teams), on a
schedule, with zero console-clicking.

Shape (pick ONE host for v1 - a Worker Service is the cleanest demo; an Azure
Function variant is the follow-up that wins the Azure crowd):

1. A .NET Worker Service (Generic Host, `dotnet publish` single-file) embedding
   the StackQL server (vendored) + an Agent Framework agent (Claude or Azure
   OpenAI via the framework's connector)
2. On a schedule, the agent runs a `read_only` drift suite expressed as SQL
   checks: newly public storage, security groups open to 0.0.0.0/0, resources
   missing required tags, drift from a declared baseline (start with a simple
   baseline.yaml of expected resources/config)
3. Findings -> a Teams Adaptive Card (incoming webhook) summarising what
   changed since the last run, EACH finding showing the SQL that produced it
   (inspectability is the brand; it is also the audit trail KPMG-style buyers
   want)
4. Azure-first provider coverage for the demo (where the .NET audience lives),
   with the github null_auth provider as the no-credentials CI/test fixture
   (org posture: repos without branch protection, etc.)

driftwatch doubles as: a Microsoft Agent Framework reference sample, a .NET
user-group talk ("MCP-connected cloud agents in Agent Framework"), and a
concrete artifact for the #7 enterprise-trust narrative (read_only + audit +
signed binary in a governance-conscious framework).

## Build, test, CI

- .NET 8 LTS baseline (also target net9.0 if trivial); C# latest; nullable
  enabled; `dotnet format` clean as a CI gate
- Solution layout: `src/StackQL.Mcp/` (library), `src/StackQL.Mcp.AgentFramework/`
  (the connector helper - separate package so the core lib has no Agent
  Framework dep), `samples/driftwatch/`, `tests/`
- Tests: xUnit - unit (pin parse / cache path / launch-arg construction),
  integration (spawn + initialize + tools/list against the github null_auth
  fixture - the family conformance check), and a `Vendored` build test that
  publishes single-file and runs it. CI matrix: ubuntu + windows + macos on
  GitHub Actions; `dotnet pack` produces the NuGet on tag
- Reuse the packaging repo's scripts/smoke-test.py via the `--cmd` mode in CI
  to assert cross-implementation parity
- Source Link + deterministic build + embedded PDBs (enterprise audiences
  check these); sign the NuGet if/when a cert is available (ties to #7)

## Milestones

1. Library core (sidecar mode) + conformance xUnit green on 3 OSes; NuGet id
   reserved; minimal README quick-start
2. Vendored single-file path working; Agent Framework wiring proven in a
   throwaway sample (validates the integration before polishing the adapter)
3. driftwatch demo (Worker Service) against Azure + the github fixture, with a
   recorded run and the Teams card; the AsAgentToolsAsync adapter if the
   framework API is stable enough
4. Publish StackQL.Mcp (+ .AgentFramework) to NuGet (manual), docs polish,
   announce: dev.to + the Microsoft Agent Framework GitHub discussions + a
   Melbourne/Sydney .NET user-group talk; cross-link from stackql.io docs

## Conventions (house style - match the family)

- Plain hyphens only (no em dashes); ASCII arrows `->`; no unicode bullets/arrows
- Matter-of-fact tone in all docs and comments; no hyperbole, no sycophancy
- Stderr/ILogger for diagnostics; stdout belongs to the MCP protocol
- MIT license; mcp-name reference: io.github.stackql/stackql-mcp
- Verify-don't-assume for moving APIs: the official C# MCP SDK and Microsoft
  Agent Framework are both young (Agent Framework 1.0 is from April 2026) -
  check their current package names, versions, and tool-registration surfaces
  against live docs at build time rather than from memory
