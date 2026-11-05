# StackQL local plugin for ChatGPT and Codex (packaging)

This package is the ChatGPT desktop and Codex distribution of the StackQL MCP
server as a local stdio process. The user-facing description is in
[plugins/stackql/README.md](plugins/stackql/README.md); this file covers the
package layout, how it is rendered, validated and shipped. The sibling Claude
plugin under `packaging/claude-plugin` is built from the same sources.

## Layout

`plugins/stackql/` is the plugin root and ships in two manifest layouts at
once, from one source:

- Portable Agent Plugins layout: `plugin.json` (`$schema` plus
  `extensions.com.openai` with the listing `interface`, `review` test cases,
  `publication` release notes, `onboardingSkill` and
  `requires_local_executor`) and `mcp.json` (`type: stdio`, `${PLUGIN_ROOT}`
  paths). This is what the plugin directory validator reads.
- Codex overlay: `.codex-plugin/plugin.json` and `.mcp.json`, read by the
  Codex marketplace route. The overlay is **derived** from `plugin.json` by
  `packaging/mcpb/scripts/render-platforms.sh`; edit `plugin.json` and
  re-render, never the overlay. `validate.py` fails when the two drift.

Both layouts launch the same vendored payload:

- `server/bin/stackql-mcp.js`: the `@stackql/mcp-server` npm launcher,
  byte-identical to `packaging/mcpb/npm/bin/stackql-mcp.js`. It needs only
  Node.js built-ins (no `npx`, no npm at runtime), downloads the pinned
  StackQL bundle on first run, verifies its SHA-256 and spawns
  `stackql mcp --mcp.server.type=stdio` with `~/.stackql` as the approot,
  `.env` as the credential file and `.stackqlrc` as the config file
  (`STACKQL_PLUGIN_CONFIG_DIR` overrides the directory).
- `server/platforms.json`: the pins, rendered from the published release
  checksums and committed (the directory reads the package from git).
- `skills/stackql-setup/SKILL.md`: the onboarding skill, copied from
  `packaging/mcpb/skills/stackql-setup/SKILL.md`.
- `LICENSE`, `assets/icon.png`, `README.md`.

The repo marketplace entry `.agents/plugins/marketplace.json` points at the
plugin root and is version-independent.

## Install from this repository (Codex marketplace)

Node.js 24.22.1 or later is required (22.23.4 or 26.12.1 on those release
lines: the [October 2026 Node.js security releases](https://nodejs.org/en/blog/vulnerability/october-2026-security-releases);
earlier builds have unpatched TLS hostname verification bypasses and still
trust the compromised eMudhra intermediate CA).

```shell
codex plugin marketplace add stackql/stackql --sparse .agents/plugins --sparse packaging/openai-plugin
codex plugin add stackql@stackql
```

Restart the ChatGPT desktop app or Codex after installation. The plugin is
also available automatically as a repo marketplace when working in this
repository. The public plugin directory path (ChatGPT desktop, Desktop only
badge) is in [docs/chatgpt-codex-plugin.md](/docs/chatgpt-codex-plugin.md).

## Provider authentication

The launcher uses these StackQL configuration files:

- `~/.stackql/.env` for provider credential values (created empty on first
  launch)
- `~/.stackql/.stackqlrc` for optional nonstandard auth configuration

Providers define their standard environment variable names, so no auth block
is needed when those names are used; for example AWS and AWS Cloud Control use
`AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`. GitHub public resources need
no credential. The `reload_credentials` MCP tool re-sources `.env` without
restarting the server and reports variable names and a per-provider status of
`ok`, `unresolved` or `not_checked`, never secret values. Restart after
changing `.stackqlrc`, because `reload_credentials` reloads values but not
auth contexts.

## Render, validate, package

From `packaging/mcpb`, after the `.mcpb` assets for `X.Y.Z` are on the GitHub
release (the renderer fetches their checksums):

```shell
make openai-plugin-manifest VERSION=X.Y.Z   # also run by: make npm-manifest
make openai-plugin-zip VERSION=X.Y.Z        # ../openai-plugin/dist/stackql-plugin-X.Y.Z.zip
```

From the repository root:

```shell
python packaging/openai-plugin/scripts/validate.py
python packaging/openai-plugin/scripts/smoke-test.py
python packaging/openai-plugin/scripts/build-zip.py --check packaging/openai-plugin/dist/stackql-plugin-X.Y.Z.zip
```

`validate.py` checks the directory's listing limits (30-character
`shortDescription`, 5 positive and 3 negative review cases, https listing
URLs, onboarding skill), the ZIP content policy (no hooks, app references,
reviewer credentials or package installs) and the single-source invariants
(launcher and skill byte-equal to their sources, every version equal to
`packaging/mcpb/npm/package.json`). It warns while
`review.demo_recording_url` is unset; set it before submitting for review.
The smoke test drives the launcher over stdio: MCP initialization, tool
discovery, provider download and a provider query; set `STACKQL_MCP_BIN` to a
built binary to skip the download.

The `stdio-plugins` workflow runs all of this on Linux, macOS, Windows x64 and
Windows ARM64, re-renders the plugin from the published checksums to prove the
committed pins, and uploads the ZIP as a run artifact. Directory submission is
manual: docs/stackql_release_process.md step 11.
