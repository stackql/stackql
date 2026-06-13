# driftwatch

A cloud drift sentinel for .NET shops, built on the embedded StackQL MCP server.

Platform teams want to know the instant real cloud state drifts from intended
state - new public exposure, untagged spend, config that no longer matches
policy - posted where they already work (Teams), on a schedule, with zero
console-clicking. driftwatch is a .NET Worker Service that does exactly that.

## How it works

1. On startup it embeds a StackQL MCP server in `read_only` mode (nothing it does
   can mutate cloud state) and pulls the configured providers.
2. On a schedule it runs the drift suite in [`checks.json`](checks.json): each
   check is a named SQL query whose returned rows are findings. An empty result
   is "clean".
3. Findings are posted to a Teams Adaptive Card via an incoming webhook, each
   finding showing the exact SQL that produced it - the card doubles as an audit
   trail. With no webhook configured, findings are written to the log, so the
   sample runs end to end with zero external setup.

The default checks run against the github `null_auth` provider, which needs no
credentials. Point [`baseline.yaml`](baseline.yaml) and `checks.json` at Azure (or
any of the 40+ providers) for real coverage.

## Run it

```bash
dotnet run --project samples/driftwatch
```

Configuration lives in [`appsettings.json`](appsettings.json) under the
`Driftwatch` section:

| Setting | Meaning |
| --- | --- |
| `Interval` | How often to run the suite (e.g. `06:00:00`). |
| `RunOnStartup` | Run once immediately before the first interval wait. |
| `TeamsWebhookUrl` | Teams incoming-webhook URL. Empty = log findings. |
| `ChecksPath` | Path to the checks file. |
| `Providers` | Providers to pull at startup. |

The Teams webhook can also be supplied out-of-band via the
`DRIFTWATCH_TEAMS_WEBHOOK` environment variable so it stays out of config.

## Single-file publish

The project is set up for a self-contained single-file publish. Vendor the
StackQL bundle so the published executable carries the binary:

```bash
dotnet publish samples/driftwatch -c Release -r win-x64 \
  -p:StackqlVendorBundle=/path/to/stackql-mcp-windows-x64.mcpb
```

## Adding an Agent Framework agent

This v1 runs the drift SQL directly, so it needs no LLM credentials. To turn the
suite into an agent that explains findings in natural language, add the
`StackQL.Mcp.AgentFramework` package and wire the StackQL tools into an
`AIAgent` with `server.AsAgentToolsAsync()` - see the repo README.
