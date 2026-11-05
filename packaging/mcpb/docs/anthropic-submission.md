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

- "Uses a credential from the user's machine": the server reads
  `~/.stackql/.env` and sends provider credentials to the provider APIs. The
  README states it; the reviewer confirms the behaviour matches.
- A first submission that fails the security scan is rejected outright, so
  the README's "What the plugin runs, fetches and sends" section must stay
  accurate: the launcher downloads and executes the pinned `stackql` binary
  from `releases.stackql.io` (SHA-256 verified), provider definitions come
  from the StackQL registry, queries go to the providers the user queries,
  nothing goes to StackQL Studios.

## Outside the repository

One-time:

- [ ] The June 2026 Desktop Extension listing: look for it at
      claude.ai/directory/manage and at platform.claude.com/plugins/submissions.
      If it shows **Withdraw**, withdraw it. If it is there without that
      button, or not there at all, email directory@anthropic.com asking what
      happens to the MCPB listing and to attach the plugin submission to the
      same organization. Until it is withdrawn or moved the portal can refuse
      **Submit for review** with "Already submitted by another organization".
- [ ] Submit from the organization that should own the listing long term, on
      a paid plan (Pro, Max, Ultra: your own account; Team or Enterprise: an
      Owner, or a member granted the **Directory** permission), with the
      GitHub account connected on claude.ai that has push access to
      `stackql/stackql-claude-plugin`. The repository must be public before
      the listing goes live.
- [ ] stackql.io `/terms` and `/support` live (stackql/stackql.io#305); the
      manifest already points at them. `/privacy` and `/docs` are live.
- [ ] Create the mirror and push the first render:
      `gh repo create stackql/stackql-claude-plugin --public --description "StackQL plugin for Claude (mirror of packaging/claude-plugin in stackql/stackql)"`
      then `make claude-plugin-mirror VERSION=X.Y.Z` from `packaging/mcpb`.
- [ ] Portal: **Submit new** > **Plugin bundle**; repository
      `stackql/stackql-claude-plugin`, plugin path empty, branch `main`;
      **Validate**, fix anything marked Blocking, re-validate; check the
      listing details (read from `plugin.json` and the README); data handling
      answers (the plugin stores no personal data; data goes to the cloud
      providers the user queries and to GitHub releases for the binary
      download; no retention; not intended for people under 18); the four
      compliance acknowledgements and a monitored contact email; choose
      **GitHub push webhook** and set it up (needs repository admin); submit.
- [ ] After the first version passes, select **Publish**; a reviewer
      publishes it. Then turn on **Publish new versions automatically** on the
      plugin's Settings tab if offered.

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
