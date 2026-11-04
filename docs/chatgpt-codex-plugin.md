# StackQL local plugin for ChatGPT and Codex

The StackQL plugin runs the existing StackQL MCP server as a local process over
stdio. ChatGPT desktop or Codex starts the process, and provider credentials
remain on the user's machine.

Supported clients:

- ChatGPT desktop app, from the plugin directory (Desktop only badge) or the
  repository marketplace
- Codex CLI, from the repository marketplace

ChatGPT web cannot run a local stdio server. Plugins are not available in the
Codex IDE extension.

## Known limitation: desktop Chat sessions

The ChatGPT desktop app can start the plugin, complete the MCP handshake, and
still not expose the StackQL tools to a Chat session. This is an open upstream
bug, [openai/codex#38162](https://github.com/openai/codex/issues/38162),
reproduced on desktop builds through 26.924. Codex threads in the desktop app
and Codex CLI are unaffected. If a Chat session reports that no StackQL tools
are available, start a Codex thread instead.

## Install from the ChatGPT plugin directory

In the ChatGPT desktop app, search the plugin directory for StackQL and
install it. The listing carries a Desktop only badge: the plugin installs only
from the desktop app, and its tools are available in threads that run on your
computer. Node.js 24.22.1 or later must be installed (see the next section).
After installing, run the plugin's setup; the bundled `stackql-setup` skill
walks through credentials and a first query.

## Install from the repository marketplace (Codex)

Install Node.js 24.22.1 or later (22.23.4 or 26.12.1 on those release lines; these are the [October 2026 security releases](https://nodejs.org/en/blog/vulnerability/october-2026-security-releases)), then run:

```shell
codex plugin marketplace add stackql/stackql --sparse .agents/plugins --sparse packaging/openai-plugin
codex plugin add stackql@stackql
```

Restart ChatGPT desktop or Codex. In Codex CLI, use `/plugins` to confirm the
plugin is installed and `/mcp` to confirm that the `stackql` server is active.

The plugin ships the StackQL launcher (the same file as the `@stackql/mcp-server`
npm package; it needs only Node.js). On first use it downloads the pinned
StackQL binary for your platform, verifies the published SHA-256, and caches
it under `~/.stackql/mcp-server-bin`.

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
