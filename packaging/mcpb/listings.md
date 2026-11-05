# Where to list the StackQL MCP server

A working register of every place worth listing the StackQL MCP server, from the
official registry and the Claude/Anthropic surfaces through to aggregators and
plain link directories. Ordered roughly by leverage.

The single most important principle: the official MCP Registry is the hub. A
large share of the aggregators and client directories below ingest from it
automatically, so publishing there once propagates the listing to many of the
others. Do that first, then submit directly to the venues that do not
auto-ingest (the Anthropic directory, the client/IDE directories, Docker, and
the awesome lists).

StackQL ships as a local stdio binary packaged as `.mcpb` (via `stackql mcp`),
plus two plugin packages that vendor the npm launcher: `packaging/claude-plugin`
for Anthropic's directory and `packaging/openai-plugin` for OpenAI's. Those
formats fit most venues directly. Two exceptions are flagged inline: the
Docker catalogue wants an OCI image, and a few venues (Smithery's URL method,
the MCP connector kind of Anthropic's directory) want a hosted streamable-HTTP
endpoint. Homebrew and the `.pkg` are install channels, not registries, so they
are not listed here.

## Summary

| Venue | Type | How to list | Status | Date | Listing URL | Follow up | Last Updated |
|---|---|---|---|---|---|---|---|
| Official MCP Registry | Canonical metadata registry | `mcp-publisher` CLI + `server.json` | ✅ Published | 2026-06-12 | [modelcontextprotocol.io/search](https://registry.modelcontextprotocol.io/v0/servers?search=stackql)<br/>[modelcontextprotocol.io/direct](https://registry.modelcontextprotocol.io/v0/servers?search=io.github.stackql/stackql-mcp) | v0.10.500.2 now advertises all 7 package types (mcpb x4, oci, npm, pypi) | 2026-06-15 |
| GitHub MCP Registry | GitHub discovery surface | Auto-ingest from official registry | ✅ Published | 2026-06-24 | [github-mcp](https://github.com/mcp?search=stackql)<br/>[api.mcp.github.com](https://api.mcp.github.com/v0/servers) | Listed as `io.github.stackql/stackql-mcp` (entry created 2026-06-24) via auto-ingest from the official registry. Sync is an hourly sweep of the official registry plus ~20 min of read-API caching: 0.11.669 was published to the official registry 2026-09-08 02:54Z, the GitHub entry updated 03:47Z and became visible 04:0xZ, so allow up to 2 hours. `make check-published` reports the advertised version; nothing to do on our side | 2026-09-08 |
| VS Code MCP Gallery | IDE client gallery (@mcp in Extensions view) | Ingests from GitHub MCP Registry | 🟡 Blocked upstream | 2026-06-16 | [docs](https://code.visualstudio.com/docs/agent-customization/mcp-servers) | Downstream of GitHub MCP Registry - the `@mcp` gallery pulls from it, so this follows automatically once that row lands. No separate submission. Manual `.vscode/mcp.json` with npx works today regardless of gallery listing | 2026-06-16 |
| Anthropic directory (Claude plugin bundle) | Vendor directory (Claude) | Developer portal at claude.ai/directory/manage, **Plugin bundle** read from the `stackql/stackql-claude-plugin` mirror (`make claude-plugin-mirror`); see [docs/anthropic-submission.md](docs/anthropic-submission.md) | ☐ Resubmission needed | 2026-11-05 |  | The June 2026 Desktop Extension (`.mcpb`) listing path is deprecated: the directory no longer accepts MCPB submissions and takes a local server only inside a plugin bundle (stackql#895). Withdraw or migrate the old listing (email directory@anthropic.com if it shows no Withdraw button), create the mirror repo, then submit; stackql.io `/terms` and `/support` must be live first (stackql.io#305) | 2026-11-05 |
| OpenAI plugin directory (ChatGPT desktop + Codex) | Vendor directory (OpenAI) | Repo marketplace, self-serve (`.agents/plugins/marketplace.json`, see `docs/chatgpt-codex-plugin.md`); public directory at platform.openai.com/plugins as a **Skills only** upload of the ZIP from `make openai-plugin-zip` (local MCP allowlist granted by the OpenAI representative; docs/stackql_release_process.md step 11) | 🟡 Marketplace live, directory submission in progress | 2026-11-05 | `codex plugin marketplace add stackql/stackql --sparse .agents/plugins --sparse packaging/openai-plugin` | Expedited review invited by OpenAI, target 2026-11-13 (stackql#893). Needs stackql.io `/terms` and `/support` (stackql.io#305), the org allowlisted for local MCP, business verification and `review.demo_recording_url` in `plugin.json`. The listing carries a Desktop only badge; desktop Chat may hide local MCP tools (openai/codex#38162), Codex threads and CLI work | 2026-11-05 |
| Docker MCP Catalog | Curated infra catalogue | PR to docker/mcp-registry | 🟡 In Progress (upstream PR pending) | 2026-06-15 |  | server.yaml prepared in the stackql/mcp-registry fork (community-image, points at public stackql/stackql-mcp); pending PR to docker/mcp-registry upstream + their review | 2026-06-15 |
| npm | Package registry | `npm publish` | ✅ Published | 2026-06-15 | [@stackql/mcp-server](https://www.npmjs.com/package/@stackql/mcp-server) | npx launcher | 2026-06-15 |
| PyPI | Package registry | `twine upload` | ✅ Published | 2026-06-15 | [stackql-mcp-server](https://pypi.org/project/stackql-mcp-server/) | uvx/pip launcher | 2026-06-15 |
| Docker Hub | Container registry | `docker buildx --push` | ✅ Published | 2026-06-15 | [stackql/stackql-mcp](https://hub.docker.com/r/stackql/stackql-mcp) | multi-arch amd64+arm64 | 2026-06-15 |
| GitHub Actions Marketplace | CI marketplace (setup-stackql-mcp action) | Public repo + release + marketplace listing | ✅ Published | 2026-06-15 | [setup-stackql-mcp-server](https://github.com/marketplace/actions/setup-stackql-mcp-server) | Verified publisher; own repo stackql/setup-stackql-mcp | 2026-06-15 |
| mcpmarket.com | Aggregator | Submit on site | ✅ Published | 2026-06-13 | [mcpmarket](https://mcpmarket.com/server/stackql) |  | 2026-06-13 |
| mcp.so | Largest aggregator | Self-submit on site | 🟡 Published (config update propagating) | 2026-06-13 | [mcp.so](https://mcp.so/server/stackql/stackql) | Server Config edited to npx form 06-15; verify public page reflects + List Tools/Playground work | 2026-06-15 |
| mcpservers.org | Awesome-list site | Ingests the awesome list | ✅ Published | 2026-06-13 | [mcpservers.org](https://mcpservers.org/servers/stackql-mcp-server) |  | 2026-06-13 |
| PulseMCP | Discovery, registry backer | Ingests official registry | ✅ Published | 2026-06-15 | [pulsemcp](https://www.pulsemcp.com/servers/stackql) | Confirmed live (Official badge) | 2026-06-15 |
| Glama.ai MCP | Searchable marketplace | Auto-indexes GitHub; claim/submit | ✅ Published | 2026-06-13 | [glama.ai](https://glama.ai/mcp/servers/stackql/stackql) | Follow up on submission review | 2026-06-13 |
| Cursor Directory | IDE client directory | Submit to Cursor directory | ✅ Published | 2026-06-13 | https://cursor.directory/plugins/stackql-mcp-server | Submit for verification | 2026-06-13 |
| mcp.directory | Aggregator | Submit on site (auto-pull npm/PyPI) | 🟡 Submitted | 2026-06-15 |  | Submitted 06-15, 24h review + email on publish; confirm live ~06-16 | 2026-06-15 |
| awesome-mcp-servers | Curated GitHub list | GitHub PR/issue | 🟡 Pending Review | 2026-06-16 | https://github.com/punkpeye/awesome-mcp-servers/pull/8138 | Await feedback | 2026-06-13 |
| Goose Extensions (Block) | In-client extension directory | PR/submit to goose extensions site | 🟡 Submitted | 2026-06-16 | https://goose-docs.ai/extensions/detail?id=stackql | Submitted PR https://github.com/aaif-goose/goose/pull/9816 | 2026-06-16 |
| Cline MCP Marketplace | In-client marketplace | GitHub Issue to marketplace repo | 🟡 Submitted |  |  | https://github.com/cline/mcp-marketplace/issues/1803 | 2026-06-13 |
| mpak | Binary-native registry, trust scoring | `mcpb-pack` action / publish | ☐ Not Submitted |  |  | Publish signed package | 2026-06-13 |
| MCP Index | Aggregator | Submit on site | ☐ Not Submitted |  |  | Submit listing | 2026-06-13 |
| Zed Extensions | IDE extension registry (context servers) | PR to zed-industries/extensions | ☐ Not Submitted |  |  | Wrap as a Zed context-server extension (npx wrapper) | 2026-06-13 |
| Warp MCP Catalog | Terminal client MCP directory | Submit to Warp's MCP catalog | ☐ Not Submitted |  |  | Verify current submission path | 2026-06-13 |

---

* 1. Ensure your server is listed on Glama. If it isn't already, submit it at https://glama.ai/mcp/servers and verify that it passes all checks (note: you must add Dockerfile directly to Glama. For checks to pass, we only need the server to start and respond to introspection requests).  2. Update your PR by adding a Glama score badge after the server description, using this format:  `[![OWNER/REPO MCP server](https://glama.ai/mcp/servers/OWNER/REPO/badges/score.svg)](https://glama.ai/mcp/servers/OWNER/REPO)` Replace OWNER/REPO with your server's Glama path.  

## The anchor: Official MCP Registry

The canonical, centrally hosted metadata registry, backed by Anthropic, GitHub,
Microsoft, and PulseMCP. It stores metadata only and points at the release
artefact. For StackQL this means an `mcpb` package entry per platform whose
`identifier` is the GitHub release download URL and whose `fileSha256` is the
hash you generate during packaging.

- Namespace: reverse-DNS tied to a verified GitHub account or domain, for
  example `io.github.stackql/stackql-mcp` or `ai.stackql/stackql-mcp`.
- Publish with the `mcp-publisher` CLI against a `server.json`.
- The `.mcpb` URL must contain the string `mcp` (the filenames already do).
- Currently in a preview / API-freeze state, so expect occasional churn.

Publishing here is the prerequisite for most of the auto-ingesting venues below.

## Anthropic / Claude surfaces

Anthropic runs one vetted directory (claude.ai/directory, managed at
claude.ai/directory/manage) with two submission kinds. Which one applies
depends on how the server is delivered:

- Plugin bundle: a GitHub repository folder with `.claude-plugin/plugin.json`,
  a README, a licence, skills and a local MCP server declared by command. This
  is the only path for a local stdio server: desktop extension (`.mcpb`)
  listings are deprecated and the directory no longer accepts them. A plugin's
  local server loads in Claude Code and in Cowork sessions on the user's
  computer; claude.ai chat ignores it, the skill still loads there. StackQL's
  plugin lives at `packaging/claude-plugin` and is mirrored to
  `stackql/stackql-claude-plugin`, the repository the listing tracks. The
  checklist is in [docs/anthropic-submission.md](docs/anthropic-submission.md).
- MCP connector: one remote HTTPS MCP server submitted by URL (OAuth, Origin
  validation). It applies only if a hosted streamable-HTTP StackQL endpoint is
  stood up; its review criteria still describe what a reviewer checks on the
  server inside the plugin (tool annotations, read/write separation,
  actionable errors), and the server meets them.

Manual `.mcpb` install in Claude Desktop (double-click, drag and drop, or
Settings > Extensions > Advanced settings > Install Extension) is unaffected
and remains the Claude Desktop chat path, so the bundles keep shipping.

Review requirements for the plugin bundle: a README of at least 40 words that
states everything the plugin runs, fetches and sends (it doubles as the
listing description), a licence, https privacy policy, terms, support and
documentation URLs in the manifest, a dependency-free launcher started as
`node ${CLAUDE_PLUGIN_ROOT}/<file>`, and no package installs or binaries in
the plugin folder. Every version is validated and security-scanned; a person
reviews the first one.

This is the highest-credibility placement for a "built on Claude" infrastructure
tool, so it is worth meeting the review bar even though it is more work than the
aggregators.

## Major aggregator registries

These are the high-traffic general directories. mcp.so and the official registry
are the two with the broadest reach.

- mcp.so: the largest public directory (20,000+ servers indexed). Self-submit on
  the site. Highest raw discovery volume.
- Smithery.ai: registry plus optional hosting and tool-call analytics. Publish
  with `smithery mcp publish <url|bundle> -n <org/server>`. The URL method needs
  a deployed streamable-HTTP server; a local stdio binary fits their CLI/bundle
  install flow instead. If you use their hosting, inject provider credentials
  locally rather than through their infrastructure.
- Glama.ai/mcp: a searchable marketplace with server previews. It auto-indexes
  GitHub repositories and lets you claim and enrich the listing.
- PulseMCP: a discovery site and a backer of the official registry; it ingests
  registry data, so a clean official-registry entry largely covers this.

## Infrastructure and security focused

Smaller and more curated than the mega-directories, which makes a listing here
carry more signal for an enterprise-facing infra tool.

- Docker MCP Catalog: a curated catalogue oriented toward databases, cloud
  services (AWS, GCP, Cloudflare, and similar), and API integrations - squarely
  StackQL's space. It requires a container image rather than a binary bundle, so
  to list here you would add a `ghcr.io` or Docker Hub image as an extra build
  target. Optional, but on-theme and high-signal.
- mpak (mpak.dev): an open-source registry that is binary-native (it supports
  node, python, and binary server types) with built-in supply-chain security
  scanning and L1-L4 trust scoring. A strong fit for a signed binary bundle, and
  the trust score is a useful credibility signal for enterprise buyers. The
  `NimbleBrainInc/mcpb-pack` GitHub Action can publish to it.

## Client and IDE directories

The places where users of specific clients discover servers. Several ingest from
the official registry; the in-client marketplaces usually want a direct
submission.

- GitHub MCP Registry: GitHub's own discovery surface. As of 2026-06 it is a
  CURATED directory (partners + selected community); the advertised auto-ingest
  from the official registry is announced but NOT yet shipped (GitHub blog uses
  future tense). Being in the official registry is the documented prerequisite
  and is done; appearing on github.com/mcp is then a waiting game on GitHub
  building the bridge. It sorts/surfaces by GitHub stars + community activity, so
  the lever on our side is making the source repo (stackql/stackql) legibly an
  MCP server. Supports npm, Docker, and MCPB packages once ingested.
- Cursor: maintains an MCP directory and an "Add to Cursor" install flow. Submit
  to the Cursor directory (verify the current submission path on their site).
- VS Code (GitHub Copilot agent mode): exposes an MCP servers gallery that draws
  on the registry and a curated list. Supports local stdio and remote
  HTTP/SSE servers.
- Cline MCP Marketplace: an in-client marketplace with filtering by installs,
  date, stars, and category. Listing is via a GitHub pull request to Cline's
  marketplace repository with a metadata entry and README (confirm the current
  repo and schema before submitting).

## Link aggregators and awesome lists

Lower effort, format-agnostic, and good for breadth. Most of these just index a
GitHub repo or the official registry, so the `.mcpb`-versus-binary distinction
does not matter to them.

- punkpeye/awesome-mcp-servers (GitHub): the canonical "awesome" list. Add an
  entry via a pull request or issue. No install tooling, but heavily browsed.
- mcpservers.org: a site rendering of the awesome list; gets you in by virtue of
  the list entry above.
- mcp.directory: auto-pulls metadata from GitHub and publishes within about a
  day. If it has already auto-discovered your server from the official registry,
  you can claim the listing for a verified badge and edit access.
- MCP Index (mcpindex.net): a directory aimed at Claude, Cursor, and Cline
  users. Submit on the site.
- mcpmarket.com (MCP Market): a general directory; submit on the site.
- modelcontextprotocol/servers (GitHub): the official repo's community-servers
  list. Add StackQL via a pull request - distinct from the registry, and a
  recognised reference point.

## Adjacent or optional

- mcpbundles.com: relevant only if you want a hosted, cloud-proxied bundle
  presence. StackQL self-hosts signed binaries, so this is unlikely to be
  needed.
- Vertical or industry registries (for example the MACH Alliance registry): only
  worth it if targeting that specific vertical; not a general fit.

## Prepare the metadata once

Every venue asks for substantially the same fields, so assemble one pack and
reuse it:

- Name (reverse-DNS) and display name
- One-line description and a longer description
- Categories and tags: infrastructure, cloud, database, devops,
  infrastructure-as-code, observability
- Transport: stdio (plus remote/streamable-HTTP if you host one)
- Install method: the per-platform `.mcpb` release URLs and SHA-256 hashes; the
  OCI image reference if you build one
- Repository, homepage, and documentation URLs
- Licence (MIT)
- Logo (SVG) and favicon
- Screenshots (required by Anthropic for any interactive UI)
- Privacy policy, terms of service and support URLs (the Anthropic and OpenAI
  directories read them from the plugin manifests; a missing or incomplete
  privacy policy is an immediate rejection)
- Maintainer and security contact
- A ready-to-paste config snippet for command-based clients, for example:
  `{ "stackql": { "command": "stackql", "args": ["mcp"] } }`

## Suggested order

1. Publish to the Official MCP Registry (propagates to many of the rest).
2. Submit the Claude plugin bundle to the Anthropic directory (the MCP connector
   kind only if you host a remote endpoint), and upload the ChatGPT plugin ZIP
   to OpenAI's directory (docs/stackql_release_process.md steps 11 and 12).
3. Submit directly to mcp.so, Smithery, and Glama.
4. Add the awesome-list and community-list PRs, and claim the mcp.directory and
   Glama auto-listings.
5. Add the client directories you care about: GitHub, Cursor, VS Code, Cline.
6. If you build an OCI image, submit to the Docker MCP Catalog; publish to mpak
   for the trust-score signal.

## Note on the launch command

Several listings carry a command-style config (`command: stackql`,
`args: ["mcp"]`). If running StackQL as a stdio MCP server needs more than the
bare subcommand, update both the `.mcpb` manifest and these snippets to the
exact invocation, since clients launch the binary with precisely those
arguments.
