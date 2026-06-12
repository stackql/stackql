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

StackQL ships as a local stdio binary packaged as `.mcpb` (via `stackql mcp`).
That format fits most venues directly. Two exceptions are flagged inline: the
Docker catalogue wants an OCI image, and a few venues (Smithery's URL method,
the remote side of Anthropic's directory) want a hosted streamable-HTTP
endpoint. Homebrew and the `.pkg` are install channels, not registries, so they
are not listed here.

## Summary

| Venue | Type | How to list | Status | Date | Listing URL | Follow up | Last Updated |
|---|---|---|---|---|---|---|---|
| Official MCP Registry | Canonical metadata registry | `mcp-publisher` CLI + `server.json` | ✅ Published | 2026-06-12 | [modelcontextprotocol.io/search](https://registry.modelcontextprotocol.io/v0/servers?search=stackql)<br/>[modelcontextprotocol.io/direct](https://registry.modelcontextprotocol.io/v0/servers?search=io.github.stackql/stackql-mcp) |  | 2026-06-13 |
| mcpmarket.com | Aggregator | Submit on site | ✅ Published | 2026-06-13 | [mcpmarket](https://mcpmarket.com/server/stackql) |  | 2026-06-13 |
| mcpservers.org | Awesome-list site | Ingests the awesome list | ✅ Published | 2026-06-13 | [mcpservers.org](https://mcpservers.org/servers/stackql-mcp-server) |  | 2026-06-13 |
| GitHub MCP Registry | GitHub discovery surface | Ingests official registry | 🟡 Auto Indexed (Pending) | 2026-06-12 | [github-mcp](https://github.com/mcp?search=stackql) | May need OCI package | 2026-06-13 |
| Anthropic Connectors / Desktop Extensions Directory | Vendor directory (Claude) | Review form, Anthropic-vetted | 🟡 Pending Review | 2026-06-13 |  | Follow up on submission review ([Google Form](https://docs.google.com/forms/u/0/d/e/1FAIpQLScHtjkiCNjpqnWtFLIQStChXlvVcvX8NPXkMfjtYPDPymgang/viewform?usp=form_confirm)) | 2026-06-13 |
| mcp.so | Largest aggregator | Self-submit on site | 🟡 Submitted | 2026-06-13 | [mcp.so](https://mcp.so/server/stackql/stackql) | Update post `npx` enablement | 2026-06-13 |
| Cursor Directory | IDE client directory | Submit to Cursor directory | 🟡 Pending Verification | 2026-06-13 | https://cursor.directory/plugins/stackql-mcp-server | Submit for verification | 2026-06-13 |
| Glama.ai MCP | Searchable marketplace | Auto-indexes GitHub; claim/submit | 🟡 Pending Review | 2026-06-13 |  | Follow up on submission review | 2026-06-13 |
| PulseMCP | Discovery, registry backer | Ingests official registry | 🟡 Pending Verification | 2026-06-13 |  | Confirm registry ingestion | 2026-06-13 |
| Smithery.ai | Registry + hosting + analytics | `smithery mcp publish` | ☐ Not Submitted |  |  | Publish via Smithery CLI | 2026-06-13 |
| Docker MCP Catalog | Curated infra catalogue | Submit container image | 🚧 Blocked |  |  | Needs OCI image | 2026-06-13 |
| mpak | Binary-native registry, trust scoring | `mcpb-pack` action / publish | ☐ Not Submitted |  |  | Publish signed package | 2026-06-13 |
| VS Code MCP Gallery | IDE client gallery | Ingests registry / curated list | ☐ Not Submitted |  |  | Monitor for registry ingestion | 2026-06-13 |
| Cline MCP Marketplace | In-client marketplace | GitHub PR to marketplace repo | ☐ Not Submitted |  |  | Submit marketplace PR | 2026-06-13 |
| mcp.directory | Aggregator | Auto-pulls; claim listing | ☐ Not Submitted |  |  | Claim listing once discovered | 2026-06-13 |
| awesome-mcp-servers | Curated GitHub list | GitHub PR/issue | 🟡 Pending Review | 2026-06-13 | https://github.com/punkpeye/awesome-mcp-servers/pull/7417 | Submit to Glama and update PR #7417 | 2026-06-13 |
| MCP Index | Aggregator | Submit on site | ☐ Not Submitted |  |  | Submit listing | 2026-06-13 |
| Goose Extensions (Block) | In-client extension directory | PR/submit to goose extensions site | ☐ Not Submitted |  |  | Verify current submission path; CLI-command extension fits the binary/npx vectors | 2026-06-13 |
| Zed Extensions | IDE extension registry (context servers) | PR to zed-industries/extensions | ☐ Not Submitted |  |  | Wrap as a Zed context-server extension (npx wrapper) | 2026-06-13 |
| Warp MCP Catalog | Terminal client MCP directory | Submit to Warp's MCP catalog | ☐ Not Submitted |  |  | Verify current submission path | 2026-06-13 |
| Roo Code MCP Marketplace | In-client marketplace | Submit to Roo marketplace repo | ☐ Not Submitted |  |  | Verify current submission path | 2026-06-13 |
| GitHub Actions Marketplace | CI marketplace (setup-stackql-mcp action) | Public repo + release + marketplace listing | 🚧 Blocked |  |  | Action built in this repo under action/; needs extraction to a public repo to list | 2026-06-13 |

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

Anthropic runs a vetted directory that surfaces servers across Claude, Claude
Desktop, Claude Mobile, Claude Code, and the API MCP connector. There are two
on-ramps depending on how the server is delivered.

- Desktop Extensions directory: the one-click local-install directory reachable
  from Claude Desktop under Settings then "Browse extensions". This is the
  natural home for the local `.mcpb` bundles StackQL produces. Listings are
  Anthropic-reviewed.
- Connectors Directory: the review form at
  `claude.com/docs/connectors/building/submission`. This path is geared toward
  remote servers (OAuth, HTTPS, Origin-header validation) and applies if you
  stand up a hosted streamable-HTTP StackQL endpoint. The two are converging
  into a single software directory covering connectors, skills, and plugins.

Review requirements to prepare for either path: a public documentation link by
publish date, a complete privacy policy (a missing or incomplete one is an
immediate rejection), a server logo (SVG or URL) and favicon, and, for any
interactive UI, promotional screenshots. The submission form is always open.

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

- GitHub MCP Registry: GitHub's own discovery surface. It ingests from the
  official registry and supports npm, Docker, and MCPB packages.
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
- Privacy policy URL (mandatory for the Anthropic directory)
- Maintainer and security contact
- A ready-to-paste config snippet for command-based clients, for example:
  `{ "stackql": { "command": "stackql", "args": ["mcp"] } }`

## Suggested order

1. Publish to the Official MCP Registry (propagates to many of the rest).
2. Submit to the Anthropic directory (Desktop Extensions for the local bundle;
   Connectors Directory if you host a remote endpoint).
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
