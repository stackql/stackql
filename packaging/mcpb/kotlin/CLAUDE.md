# CLAUDE.md - stackql-mcp-kotlin (embedded StackQL MCP server for Kotlin/JVM)

## What this project is

The JVM member of the StackQL embedded-MCP family: a Kotlin library plus a
build-pipeline product demo. Target repo: `stackql/stackql-mcp-kotlin`,
published to Maven Central as `io.stackql:stackql-mcp` (coordinate
availability and the io.stackql namespace verification on Central are
research task one; Central publishing is manual/2FA-ish via the portal,
consistent with the npm/PyPI stance).

Layers:

1. Library `stackql-mcp`: locate-or-install the binary (classpath/JAR
   resource first if vendored, then shared cache, then download with sha
   verification), spawn over stdio, return a connected client via the
   official MCP Kotlin SDK (`modelcontextprotocol/kotlin-sdk`)
2. Demo: see below - a Gradle plugin, because the JVM community's agentic
   surface area is the BUILD, and nobody is doing agentic checks there yet

## The embedding contract (do not deviate)

Source of truth: stackql/stackql-mcpb-packaging (the packaging repo).

- Per-version sha256 pins from the release .sha256 assets (consolidated
  platforms.json release asset planned - prefer once present); pins baked
  into the library's resources at render time
- Canonical launch args (cwd-independence mandatory):
  `mcp --mcp.server.type=stdio --approot <home>/.stackql
   --mcp.config {"server": {"mode": "<mode>", "audit": {"disabled": true}}}`
- Default `read_only`; escalation is explicit opt-in
- Shared cache: `~/.stackql/mcp-server-bin/<version>/<platform-key>/`
  (same as the npm/pypi wrappers - check before downloading)
- Platform keys: linux-x64, linux-arm64, windows-x64, darwin-universal
- Env overrides honored: STACKQL_MCP_BIN, STACKQL_MCP_BUNDLE
- Conformance: packaging repo scripts/smoke-test.py --cmd passes against
  the library's launcher main; mirror as JUnit integration tests

## Demo: `costgate` - the cloud cost gate for CI/CD pipelines

Business use case: platform teams want deploys blocked (or flagged) when the
infrastructure being shipped will blow the budget - BEFORE it exists.
Today's answer is post-hoc billing surprise; costgate makes cost a build
check, like tests or lint.

Shape: a Gradle plugin + standalone CLI (same core):

1. `gradle costgate` (or `costgate check --budget 500/month`): reads a
   declared resource intent file (start simple: costgate.yaml listing
   resource types/sizes/regions; terraform-plan ingestion is a v2 research
   item, not v1 scope)
2. The agent uses read_only stackql tools to price the intent against live
   provider pricing data and current account context (existing commitments,
   region factors)
3. Emits a cost report (console + JUnit-style XML so CI UIs render it) and
   exits non-zero over budget - the gate
4. `--explain` mode: the agent narrates the top cost drivers and cheaper
   alternatives (different instance family/region), with the SQL shown
5. Demo fixture: pricing queries need no account credentials for some
   providers - verify which pricing surfaces are anonymous; otherwise use a
   sandbox account and the github null_auth fixture for the no-creds CI test

This also bridges to the GitHub Action story: costgate in a workflow is the
JVM-native sibling of setup-stackql-mcp.

## Build and test

- Kotlin 2.x, Gradle (version catalog), JDK 17 baseline; deps:
  modelcontextprotocol/kotlin-sdk + kotlinx-serialization; keep it lean
- Tests: JUnit5 - unit (pins/extract/cache/args), integration (spawn +
  initialize + tools/list via the github null_auth fixture); CI matrix
  linux+macos+windows on GitHub Actions
- Publishing: Maven Central via the central portal; Gradle plugin portal
  for the plugin (separate listing - another uncrowded directory)

## Milestones

1. Library + conformance tests green on 3 OSes; Central namespace verified
2. costgate CLI + Gradle plugin with a worked one-provider demo, recorded
3. Publish library + plugin, announce (Kotlin Weekly, r/Kotlin, a JVM
   meetup talk: "your build now knows what it costs")

## Conventions

- Plain hyphens only (no em dashes); ASCII arrows `->`
- Matter-of-fact tone; no hyperbole
- Stderr for diagnostics, stdout belongs to protocols
- MIT license; mcp-name reference: io.github.stackql/stackql-mcp
