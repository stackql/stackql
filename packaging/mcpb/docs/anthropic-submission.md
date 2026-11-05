# Anthropic directory submission (Claude plugin bundle)

This is the checklist for listing StackQL in Anthropic's directory
(claude.ai/directory), the catalogue people browse from claude.ai, Cowork and
Claude Code. It replaces the Desktop Extensions checklist that used to live
here: desktop extension (`.mcpb`) listings are deprecated and the directory no
longer accepts MCPB submissions. A local stdio server can only be listed inside
a **plugin bundle**, which is what `packaging/claude-plugin` is.

Sources (read 2026-11-05): claude.com/docs/directory/publish,
claude.com/docs/plugins/submit, claude.com/docs/plugins/pre-submission-checklist,
claude.com/docs/plugins/platform-support, claude.com/docs/connectors/building/submission.

## What we are submitting, in Anthropic's terms

- **Plugin bundle**, not MCP connector. The portal's MCP connector kind takes
  one remote HTTPS server by URL; "a local server can't be submitted on its
  own; include it in a plugin instead".
- The plugin is a GitHub repository folder holding `.claude-plugin/plugin.json`,
  `README.md` (shown as the listing description), `LICENSE`, the onboarding
  skill under `skills/`, and the local MCP server declared inline as
  `node ${CLAUDE_PLUGIN_ROOT}/server/bin/stackql-mcp.js`.
- Where the server runs: Claude Code, and Cowork sessions that run on the
  user's computer. claude.ai chat and the desktop app's chat ignore local
  servers; the skill still loads there. Claude Desktop chat keeps using the
  manually installed `.mcpb` (still supported: double-click, drag and drop, or
  Settings > Extensions > Advanced settings > Install Extension).
- Which repository: the directory reads the plugin from GitHub and tracks a
  branch or tag, and it holds every version of a plugin that lives in a
  subfolder of its repository for a reviewer (the launcher is a non-shell file
  the validator cannot follow from a subfolder). So the source of truth stays
  in `packaging/claude-plugin` here, and `make claude-plugin-mirror
  VERSION=X.Y.Z` pushes the rendered plugin to the root of
  `stackql/stackql-claude-plugin`, the repository the listing points at.

## Hard requirements (validation blocks the submission)

All enforced by `python packaging/mcpb/scripts/validate-claude-plugin.py` and
`claude plugin validate ./packaging/claude-plugin --strict`, both run by the
`stdio-plugins` workflow:

- `.claude-plugin/plugin.json` with `name` (`stackql`), `displayName`,
  `version` (equals the npm wrapper's), `description`, `author`, `license`,
  and the listing fields `icon`, `documentationUrl`, `supportUrl`,
  `privacyPolicyUrl`, `termsOfServiceUrl` (all `https://`).
- `README.md` of at least 40 words outside code blocks, stating everything
  the plugin runs, fetches and sends (the security scan compares it with the
  source). `LICENSE` in the plugin folder.
- Local server started as `node ${CLAUDE_PLUGIN_ROOT}/<file>` with plain
  arguments: no shell, no inline program, no `npm run`, no package launcher
  (`npx`, `uvx`), no `.mcpb` bundle reference.
- No `package.json` beside a lockfile at the plugin root, no `node_modules`,
  no `.npmrc`: the launcher is dependency-free for this reason.
- No top-level `bin/` directory (it stops claude.ai and Cowork from installing
  the plugin); the launcher lives under `server/bin/`.
- Every non-image file under 256 KiB, at most 512 files, only text files,
  images and fonts, no symlinks, no `.DS_Store`.
- Skill front matter with `name` and a single-string `description`.
- Repository under 50 MiB archived and 256 MiB unpacked, under 10,000 files,
  cross-platform file names, no `export-ignore` or `filter` git attributes.

## Reviewer holds to expect

A hold is not a rejection; an Anthropic reviewer reads the version before it
goes live, and the same hold can recur on later versions:

- "Scripts the validator couldn't follow": the node launcher, see above.
- "Uses a credential from the user's machine": the server reads
  `~/.stackql/.env` and sends provider credentials to the provider APIs. The
  README states it; the reviewer confirms the behaviour matches.
- A first submission that fails the security scan is rejected outright, so
  the README's "What the plugin runs, fetches and sends" section must stay
  accurate: the launcher downloads and executes the pinned `stackql` binary
  from `releases.stackql.io` (SHA-256 verified), provider definitions come
  from the StackQL registry, queries go to the providers the user queries,
  nothing goes to StackQL Studios.

## What the validator reports, and what to leave alone

The 2026-11-05 validation of `stackql/stackql-claude-plugin` at `main` passed its
seven checks with five warnings and one policy hold. None of them is a defect:

- Policy hold, "Scripts the validator couldn't follow" on
  `server/bin/stackql-mcp.js`: inherent to a node launcher. The validator follows
  only plain shell scripts, and a file a shell wrapper runs in turn is not
  followed either, so a wrapper would not remove it. The portal says leaving
  the script and waiting for the review is fine. Expect it on every version.
- "Unrecognized field in plugin.json" for `documentationUrl`, `supportUrl`,
  `privacyPolicyUrl` and `termsOfServiceUrl`, and "Field from another tool's
  manifest" for `icon`: these are the directory listing fields. Claude Code
  ignores them at load time and the directory reads them for the listing; the
  portal marks each "No action needed". Removing them would drop the links
  and the icon from the listing.
- "Local MCP server: not on claude.ai" and "Images and fonts passed without a
  code check" (`icon.png`): informational.
- "Local MCP servers accepted only from Verified organizations": not refused
  today, but the directory intends to accept servers that run on the user's
  computer only from Verified organizations, and StackQL Studios is not one.
  The docs describe no application ("Anthropic decides during review");
  directory@anthropic.com was asked on 2026-11-05 how to become one.
- `classification` is filled in by the directory after submission. The portal
  mentions a `classification` block in `plugin.json` but the manifest reference
  does not document one, so it is left to the automatic fill.

## Outside the repository

One-time:

- [x] The June 2026 Desktop Extension listing was made through the submission
      Google Form and appears in neither claude.ai/directory/manage nor
      platform.claude.com/plugins/submissions, so there is nothing to withdraw.
      Email directory@anthropic.com only if **Submit for review** is refused
      with "Already submitted by another organization" (it was not).
- [x] Submitted 2026-11-05 from the StackQL Studios owner's claude.ai account
      (paid plan, GitHub account with push access to the mirror connected).
      Keep submitting from that account; the listing belongs to it.
- [x] stackql.io `/terms`, `/support`, `/privacy` live (2026-11-05). `/docs`
      redirects to the homepage, so the manifest links point at `/getting-started`.
- [x] Mirror created 2026-11-05 (`gh repo create stackql/stackql-claude-plugin --public`)
      and `make claude-plugin-mirror VERSION=1.0.888` pushed the root plugin to
      `main` with tag `v1.0.888`.
- [x] Portal submission 2026-11-05: **Submit new** > **Plugin bundle**, repository
      `stackql/stackql-claude-plugin`, plugin path empty, branch `main`;
      **Validate** (see the section above). Listing details are read from
      `plugin.json` and the README; **Listed on** was set to Claude Code and
      Cowork only, because the local server does not run in the Claude apps and
      the setup skill alone would mislead there. Data handling answers: personal
      data **Reads only** (queries return whatever the user's own tenancy holds,
      such as IAM or directory users; nothing is stored beyond the local session),
      skills send data elsewhere **No** (the binary and provider downloads are the
      server's and are declared in the README), retention **Not retained**
      (StackQL Studios runs no service that receives data from Claude), under 18
      **No**. Compliance: four acknowledgements, contact javen@stackql.io.
      **Auto-publish passing versions** on. New versions: **GitHub push webhook**.
- [x] Webhook configured 2026-11-05 on the mirror (Settings > Webhooks: the
      payload URL and secret the portal issued after submission, content type
      `application/json`, SSL verification on, push event only; the ping
      delivery returned 200). It is a repository webhook, not an Actions
      secret. The directory also polls `main` about every 6 hours.
- [ ] Reviewer decision on the launcher hold: shows on the plugin's page at
      claude.ai/directory/manage. The first version is published by us or the
      reviewer; later versions go live automatically (auto-publish is on).
- [ ] Verified organization: reply from directory@anthropic.com (asked
      2026-11-05) and whatever it requires, before local servers are gated on it.

Per release: docs/stackql_release_process.md step 12. In short, after the
pin PR for `X.Y.Z` merges, `make claude-plugin-mirror VERSION=X.Y.Z`; the
directory scans the new commit, and it goes live per the auto-publish setting
(or after **Publish** in the portal).

## Re-submission triggers

The directory re-scans every commit on the tracked branch, so nothing needs
re-submitting for version bumps. Material changes to the listing (privacy
policy, scope of the tools, publisher identity) still deserve a note to
directory@anthropic.com with the version that carries them. A metadata-only
fix (README, listing URLs) ships with the next release: the plugin version is
the StackQL version, and the directory serves a new version only when the
version field changes.
