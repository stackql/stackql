# StackQL local plugin for ChatGPT and Codex

The StackQL plugin runs the existing StackQL MCP server as a local process over
stdio. ChatGPT desktop or Codex starts the process, and provider credentials
remain on the user's machine.

Supported clients:

- ChatGPT desktop app
- Codex CLI

ChatGPT web cannot run a local stdio server. Plugins are not available in the
Codex IDE extension.

## Install

Install Node.js 18 or later, then run:

```shell
codex plugin marketplace add stackql/stackql --sparse .agents/plugins --sparse packaging/openai-plugin
codex plugin add stackql@stackql
```

Restart ChatGPT desktop or Codex. In Codex CLI, use `/plugins` to confirm the
plugin is installed and `/mcp` to confirm that the `stackql` server is active.

The plugin launches a pinned `@stackql/mcp-server` package. On first use, the
package downloads the matching StackQL binary, verifies the published SHA-256,
and caches it under `~/.stackql/mcp-server-bin`.

## Hello world

Start a new ChatGPT desktop or Codex session and enter:

```text
Use StackQL to install the GitHub provider if needed, then list the first five GitHub services.
```

The agent should use `pull_provider` followed by `list_services`. No GitHub auth
block or credential is needed for public data. A successful result includes
GitHub services such as `actions` or `apps`. All tool calls run through the
local StackQL process over stdio.

## Provider credentials

The plugin passes these local files to StackQL:

- `~/.stackql/.env` contains provider credential values.
- `~/.stackql/.stackqlrc` contains optional nonstandard auth configuration.

StackQL creates `.env` on first launch. Providers define standard environment
variable names, so most users only add values to that file. AWS and AWS Cloud
Control use:

```dotenv
AWS_ACCESS_KEY_ID=...
AWS_SECRET_ACCESS_KEY=...
```

No `--auth` configuration is needed for those names. Use `.stackqlrc` only when
an auth type or credential source differs from the provider defaults.

After changing `.env`, ask the agent to call `reload_credentials`. It re-sources
the file and reports variable names plus a per-provider status of `ok`,
`unresolved`, or `not_checked`; it never returns secret values. Restart the
plugin after changing `.stackqlrc`, because the tool reloads credential values
but not auth contexts.

See [MCP server documentation](mcp.md) for tools and server modes, and
[authentication documentation](auth.md) for provider-specific configuration.
