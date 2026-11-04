# StackQL

Query, provision and deprovision cloud and SaaS resources with SQL. The plugin
runs the StackQL MCP server as a local process on your computer and connects
Claude to it over stdio: discover providers, services, resources and method
contracts, run SELECT queries against live provider APIs, validate a query
before running it, and change resources with INSERT, UPDATE, DELETE and
lifecycle EXEC operations. The bundled `stackql-setup` skill walks through
credentials and a first query.

## Where it runs

The MCP server is a local process, so it loads in Claude Code and in Cowork
sessions that run on your computer. Chat on claude.ai and in the desktop app
does not start local servers. For Claude Desktop chat, install the MCP Bundle
(`.mcpb`) for your platform from the
[latest StackQL release](https://github.com/stackql/stackql/releases/latest)
instead. The skill loads on every surface.

## Requirements

Node.js 24.22.1 or later (22.23.4 or 26.12.1 on those release lines; these are
the October 2026 Node.js security releases), and network access to GitHub
releases and the StackQL provider registry on first use.

## What the plugin runs, fetches and sends

- Runs: `server/bin/stackql-mcp.js` with Node.js. It starts the StackQL
  binary as `stackql mcp --mcp.server.type=stdio` with the configuration
  directory below as the application root and the audit log disabled.
- Fetches, on first use only: the StackQL release bundle for your platform
  from `releases.stackql.io` (a proxy in front of the GitHub release assets),
  verified against the SHA-256 pinned in `server/platforms.json` and cached
  under `~/.stackql/mcp-server-bin`. Provider definitions are downloaded from
  the StackQL provider registry the first time a provider is pulled.
- Sends: your queries go from your machine straight to the cloud or SaaS
  provider APIs you query, authenticated with the credentials in your
  StackQL environment file. Nothing is sent to StackQL Studios. The binary and
  the server carry no telemetry.

## Credentials

Provider credentials live in `~/.stackql/.env` (`%USERPROFILE%\.stackql\.env`
on Windows), created empty on first start. Providers use their standard
environment variable names, for example `AWS_ACCESS_KEY_ID` and
`AWS_SECRET_ACCESS_KEY`. After editing the file, ask Claude to call
`reload_credentials`; it reports resolution status by variable name and never
returns a value. Set `STACKQL_PLUGIN_CONFIG_DIR` in the environment Claude
Code starts from to use a different directory.

## Safety

The server starts in `safe` mode: SELECT queries run freely, and INSERT,
UPDATE, DELETE and EXEC ask for approval before they run. Read-only tools are
annotated `readOnlyHint` and mutating tools `destructiveHint`, so Claude can
apply its own permission rules on top.

## Links

- Documentation: https://stackql.io/docs
- Source: https://github.com/stackql/stackql
- Privacy policy: https://stackql.io/privacy
- Support: https://stackql.io/support
