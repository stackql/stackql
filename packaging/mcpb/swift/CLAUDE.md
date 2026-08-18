# CLAUDE.md - stackql-mcp-swift (embedded StackQL MCP server for Swift/macOS)

## What this project is

The Swift member of the StackQL embedded-MCP family, with a uniquely strong
macOS story: the stackql darwin binary is Developer ID signed and Apple
NOTARISED (verified end to end June 2026), which means a signed Mac app can
bundle it inside its own .app and still pass notarisation - something the
Node/Python MCP server crowd structurally cannot do. Target repo:
`stackql/stackql-mcp-swift` (SwiftPM package + demo app).

Layers:

1. SwiftPM package `StackQLMCP`: locate-or-install the binary (app bundle
   resource first, then shared cache, then download with sha verification),
   spawn over stdio, return a connected client via the official MCP Swift
   SDK (`modelcontextprotocol/swift-sdk`)
2. Demo app (separate target in the repo): see below

## The embedding contract (do not deviate)

Source of truth: stackql/stackql-mcpb-packaging (the packaging repo).

- darwin-universal bundle covers arm64 + x86_64 in one binary
- Per-version sha256 pins from the release .sha256 assets (consolidated
  platforms.json release asset planned - prefer once present)
- Canonical launch args (cwd-independence is mandatory - macOS hosts often
  launch with cwd `/`, which is read-only; this exact failure was found and
  fixed in Claude Desktop June 2026):
  `mcp --mcp.server.type=stdio --approot <home>/.stackql
   --mcp.config {"server": {"mode": "<mode>", "audit": {"disabled": true}}}`
- Default `read_only`; escalation is explicit opt-in
- Shared cache: `~/.stackql/mcp-server-bin/<version>/darwin-universal/`
- Conformance: packaging repo scripts/smoke-test.py --cmd must pass against
  the package's launcher; mirror in XCTest

## macOS specifics that ARE the value of this repo

- Bundling: ship the binary at Contents/Resources/ (or Helpers/) of the
  .app; document codesign --deep implications and that the embedded
  binary's own Developer ID signature + notarised cdhash remain valid -
  include the verification transcript commands (codesign --verify, spctl)
- Sandbox reality check (research task): App Sandbox likely blocks
  spawning + outbound network as needed -> the demo app ships
  non-sandboxed/Developer ID distributed (document why; App Store is out of
  scope for v1)
- Quarantine: resources inside a notarised app are not quarantined; the
  download-at-runtime path however must strip/handle com.apple.quarantine -
  prefer the bundled path for shipping apps

## Demo app: `CloudLens` - a menu bar cloud sentinel

Business use case: the always-on glanceable answer to "is anything in our
cloud burning money or newly exposed?" - for the engineering lead who lives
on a Mac and will not open four consoles.

- Menu bar (MenuBarExtra, SwiftUI) app embedding the notarised binary
- On a schedule (and on demand), an agent runs a small read_only check
  suite: spend pulse (top movers), exposure pulse (public buckets, open
  security groups), and a "new since yesterday" diff
- Surfaces: menu bar icon state (calm/attention), a popover with the three
  pulses, and native notifications for new findings ("S3 bucket made public
  12 minutes ago") - notification includes the SQL behind the finding
- Demo fixture: github provider in null_auth mode (org posture pulse) so
  the app demos with zero cloud creds; AWS/Azure/GCP via Keychain-stored
  credentials as the real configuration
- Agent calls via URLSession to the Anthropic API; key in Keychain

## Build and test

- Swift 6.1+/Xcode 16.3+ (the MCP swift-sdk is a tools-version 6.1 package,
  so 6.1 is the real floor), macOS 13+; SwiftPM for the package, Xcode
  project for the app; deps: modelcontextprotocol/swift-sdk only
- XCTest: locate/extract/cache unit tests + spawn/handshake/tools-list
  integration against the github fixture; CI on macos-latest runners
- Release engineering doc: signing + notarising the demo app WITH the
  embedded binary (this doc is half the reason the repo exists)

## Milestones

1. SwiftPM package + conformance tests green; bundling doc with verified
   codesign/spctl transcripts
2. CloudLens demo with github pulse + notifications; screen recording
3. Tag v0.1.0, announce (Swift forums, iOS/macOS dev Slack communities, a
   CocoaHeads-style talk: "a notarised agent in your menu bar")

## Conventions

- Plain hyphens only (no em dashes); ASCII arrows `->`
- Matter-of-fact tone; no hyperbole
- Stderr/os_log for diagnostics, stdout belongs to protocols
- MIT license; mcp-name reference: io.github.stackql/stackql-mcp
