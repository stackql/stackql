# StackQL local plugin for ChatGPT and Codex

This package installs StackQL as a local MCP stdio server for the ChatGPT
desktop app and Codex CLI. It does not expose an HTTP service or send StackQL
credentials to StackQL Studios.

The plugin launches the pinned `@stackql/mcp-server` npm wrapper. That wrapper
selects the host platform, downloads the matching StackQL MCPB release,
verifies its SHA-256, and starts `stackql mcp --mcp.server.type=stdio`.

## Install from this repository

Node.js 18 or later is required.

```shell
codex plugin marketplace add stackql/stackql --sparse .agents/plugins --sparse packaging/openai-plugin
codex plugin add stackql@stackql
```

Restart the ChatGPT desktop app or Codex after installation. The plugin is
also available automatically as a repo marketplace when working in this
repository.

ChatGPT web cannot run local stdio MCP servers, and plugins are not available
in the Codex IDE extension. Public directory submission requires a remote HTTPS
MCP server and is outside this package's scope.

## Provider authentication

The launcher uses these existing StackQL configuration files:

- `~/.stackql/.env` for provider credential values
- `~/.stackql/.stackqlrc` for optional nonstandard auth configuration

Providers define their standard environment variable names. No auth block is
needed when those names are used. For example, AWS and AWS Cloud Control use:

```dotenv
AWS_ACCESS_KEY_ID=...
AWS_SECRET_ACCESS_KEY=...
```

GitHub public resources need no auth configuration when no GitHub credential
environment variables are set. Use `.stackqlrc` only when an auth type or
credential source differs from the provider defaults.

StackQL creates `.env` on first launch. The `reload_credentials` MCP tool
re-sources it without restarting the server. It reports variable names and a
per-provider status of `ok`, `unresolved`, or `not_checked`, but never returns
secret values. Restart after changing optional auth configuration in
`.stackqlrc`, because `reload_credentials` reloads values but not auth contexts.

## Validate

From the repository root:

```shell
python packaging/openai-plugin/scripts/validate.py
python packaging/openai-plugin/scripts/smoke-test.py
```

The smoke test exercises the plugin entrypoint, MCP initialization, tool
discovery, provider download, and a provider query over stdio.

For a release update, change the version in `plugin.json` and the pinned npm
package in `bin/stackql-mcp.js` after that npm version is published.
