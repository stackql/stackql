# stackql-mcp-server

mcp-name: io.github.stackql/stackql-mcp

uvx/pip-able launcher for the [StackQL](https://stackql.io) MCP server - a
SQL-native query and provisioning engine for cloud and SaaS infrastructure
(AWS, Azure, Google, GitHub, Databricks, and other providers), served over
the Model Context Protocol.

On first run, the launcher downloads the signed `stackql` binary for your
platform from the matching GitHub release (sha256-verified against pins baked
into this package), caches it under `~/.stackql/mcp-server-bin/`, and starts
it as an MCP stdio server. Subsequent runs start instantly from the cache.
Pure stdlib - no dependencies.

## Usage

With any MCP client that supports stdio servers:

```json
{
  "mcpServers": {
    "stackql": {
      "command": "uvx",
      "args": ["stackql-mcp-server"]
    }
  }
}
```

Provider credentials are passed through with stackql's standard `--auth` flag:

```json
{
  "mcpServers": {
    "stackql": {
      "command": "uvx",
      "args": [
        "stackql-mcp-server",
        "--auth={\"github\":{\"type\":\"null_auth\"}}"
      ]
    }
  }
}
```

Or install it: `pip install stackql-mcp-server`, then use `stackql-mcp` as the
command. Any extra arguments are passed to `stackql` after the standard MCP
server arguments. The launcher sets `--approot` to `~/.stackql` and disables
the audit file sink by default; pass your own `--approot` or `--mcp.config`
to override (later flags win).

## Environment overrides

- `STACKQL_MCP_BIN` - path to an existing `stackql` binary; skips the download.
- `STACKQL_MCP_BUNDLE` - path to a local `.mcpb` bundle to extract the binary
  from (testing; skips download and sha verification).

## Other installation vectors

- Claude Desktop one-click bundles (`.mcpb`):
  https://github.com/stackql/stackql/releases/latest
- npm: `npx -y @stackql/mcp-server`
- Docker: `docker run -i --rm stackql/stackql-mcp`
- Native installers and package managers: https://stackql.io/docs/installing-stackql

## Links

- Docs: https://stackql.io/docs
- MCP Registry: `io.github.stackql/stackql-mcp`
- Source: https://github.com/stackql/stackql
