# CLAUDE.md - packaging/mcpb/dotnet (NuGet `StackQL.Mcp`, `StackQL.Mcp.AgentFramework`)

The .NET member of the StackQL embedded-MCP family: `StackQL.Mcp` (net8.0/net9.0) acquires the `stackql` binary (explicit path, `STACKQL_MCP_BIN`, cache, local bundle, vendored `StackqlVendorBundle` resource for single-file publish, or a pin-verified download), spawns it over stdio and returns a connected `ModelContextProtocol.Core` client; `StackQL.Mcp.AgentFramework` bridges the tools into Microsoft Agent Framework / `Microsoft.Extensions.AI`. Both publish at the same version. `Directory.Build.props` `<Version>` is the stamp; `TreatWarningsAsErrors` is on and `dotnet format --verify-no-changes` gates the build.

## The contract (do not deviate)

Owned by [packaging/mcpb](../CLAUDE.md) in stackql/stackql - read "The nine wrapper vectors and the ordering rules" there first. In short:

- Package version == stackql release version; stamped by `scripts/render-platforms.sh` (`make dotnet-manifest VERSION=X.Y.Z` from `packaging/mcpb`), never hand-edited.
- Pins are data: `platforms.json` `{version, baseUrl, platforms{<key>:{bundle, sha256}}}` rendered from the published `.mcpb.sha256` release assets. `src/StackQL.Mcp/platforms.json` is an `EmbeddedResource` (`StackQL.Mcp.platforms.json`) parsed once by `Pins.Load()`. No hand-written pin table anywhere. The rendered files are gitignored - render before building.
- Runtime download from `baseUrl` (`https://releases.stackql.io/stackql/<version>`) with `User-Agent: stackql-mcp-server-dotnet/<version>`. No GitHub API calls, no `latest`, no version override.
- Shared cache `~/.stackql/mcp-server-bin/<version>/<platform-key>/`; keys `linux-x64 | linux-arm64 | windows-x64 | darwin-universal`.
- Overrides `STACKQL_MCP_BIN` and `STACKQL_MCP_BUNDLE` (local `.mcpb`, no pin check); nothing else.
- Canonical argv `mcp --mcp.server.type=stdio --approot <home>/.stackql --mcp.config {"server":{"mode":"<mode>","audit":{"disabled":true}}} [--auth=<json>]`; default mode `read_only`.
- Launcher for `scripts/smoke-test.py --cmd`: `dotnet run --project dotnet/samples/Launcher --` (`samples/Launcher`); `tests/StackQL.Mcp.Tests/ConformanceTests.cs` is the in-repo port (runs when `STACKQL_MCP_BIN` / `STACKQL_MCP_BUNDLE` is set).

## Build, test, publish

```
cd packaging/mcpb
make dotnet-manifest VERSION=X.Y.Z   # render pins + version stamp (after the .mcpb assets are on the release)
make dotnet-build    VERSION=X.Y.Z   # lint + unit tests + package
make dotnet-smoke    VERSION=X.Y.Z   # smoke-test.py --cmd against the launcher
make dotnet-publish  VERSION=X.Y.Z   # `dotnet nuget push --skip-duplicate` of both packages (NUGET_API_KEY)
```

CI: the `sdk` matrix and `dotnet-publish` job in `.github/workflows/mcp-packaging.yml`.

## Out of scope here

Demo apps (driftwatch) live in stackql-labs and depend on the published package. No new features beyond the contract; parity items go on the backlog as separate PRs.

## Writing style

Plain hyphens, ASCII arrows (`->`), QWERTY characters only, matter-of-fact tone, no stacked headings.
