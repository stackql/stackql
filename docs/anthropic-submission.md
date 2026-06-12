# Anthropic Desktop Extensions submission checklist

This is the editorial review path that puts StackQL in Claude Desktop's "Browse extensions" UI. It is **not** a cryptographic signature - it is Anthropic vetting the listing, and it is the highest-trust user-visible signal we can earn for the bundle.

Submission forms (found June 2026, buried in the Anthropic partner hub; the old "Interest Form: MCP Directory" is deprecated and redirects here):

- **Local servers (.mcpb - this is us): https://forms.gle/d8hAM5GJvxehnG4M6**
- Remote servers (hosted MCP, not applicable): https://forms.gle/fDhN3FQmnLpoY5zm6

These URLs have changed at least twice - if the local-servers form 404s or shows a deprecation notice, follow its redirect and update this doc.

## Hard requirements - get these ready before opening the form

A missing or incomplete privacy policy is the most common single-shot rejection reason. Have all of the following ready before you start.

- [ ] **Published `.mcpb` bundles** attached to the matching `stackql/stackql` GitHub release (`make publish` from the workstation and the Mac), with `.sha256` files alongside.
- [ ] **Published entry in the Official MCP Registry** at `io.github.stackql/stackql-mcp` (`make registry-publish`). Not strictly mandatory for Anthropic, but it's the canonical hub and several other directories auto-ingest from it - have it done first.
- [ ] **Public documentation URL** - a page at `https://stackql.io/docs` (or a subpath) that describes the MCP server, its tools, and how to install. Must be live by review date.
- [ ] **Complete privacy policy URL** - "we don't collect anything" is fine if it's accurate, but the document must explicitly cover: what data the server reads (cloud provider responses), what it sends home (nothing, by default), telemetry posture, and contact for privacy questions. Half-written placeholder pages are auto-rejected.
- [ ] **Logo** - SVG preferred, square aspect, transparent background. The StackQL Studios mark works. Anthropic re-uses this in the Browse extensions tile.
- [ ] **Favicon** - 32x32 or 64x64 PNG/ICO. Often the same as the logo.
- [ ] **Screenshots** - any time the extension shows interactive output to the user. Our case is interesting: we don't have a custom UI, but query results render in Claude's chat. Two or three screenshots of real queries (cloud inventory, resource enumeration, no-auth GitHub) are persuasive even if not strictly required.
- [ ] **Maintainer contact** - email and GitHub handle that Anthropic can reach for review questions.
- [ ] **Security contact** - the same or a dedicated `security@stackql.io`-style address. Anthropic asks for a vulnerability-disclosure channel.
- [ ] **Categories / tags** - choose from Anthropic's taxonomy at submission time. Pick: infrastructure, cloud, database, devops, infrastructure-as-code, observability.
- [ ] **One-line description** and **longer description**. Reuse the strings already in `manifest/manifest.template.json` and `registry/server.template.json` so all surfaces stay aligned.

## What Anthropic will check during review

Roughly, in order of how often each catches submissions:

1. Privacy policy is complete (not a placeholder).
2. Install actually works on a fresh Claude Desktop - they install the `.mcpb` and verify `tools/list` returns sensible tools with non-trivial descriptions.
3. The server does what the description says - they will run a representative query. For us, "pull a provider, list services" is the obvious thing, which our smoke test exercises.
4. No unexpected network egress. Our server only talks to the public stackql provider registry and whichever cloud APIs the user explicitly queries. Document this.
5. Logo / favicon / screenshots are present and not lorem-ipsum.

## After acceptance

Anthropic will ask for a tag/URL pattern they can pin so updates flow through. The cleanest pattern is "always reference the latest release `.mcpb` at `https://github.com/stackql/stackql/releases/latest/download/stackql-mcp-<arch>.mcpb`". This means each new stackql release that goes through `make publish` is automatically reflected in the directory without re-submission.

If they ask for a webhook on new releases, point them at the `stackql/stackql` GitHub release events.

## Re-submission triggers

You only need to re-open the review form for material changes to the **listing**, not for every release:

- Changing tool names, behaviour, or descriptions in a way that contradicts the public docs.
- Changing the privacy policy.
- Changing the maintainer / security contact.
- Pivoting from no-auth to requiring credentials by default.

Routine version bumps that don't change the above are out of scope - the latest-release URL pattern handles them.
