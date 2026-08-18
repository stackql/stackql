# CLAUDE.md - stackql-mcp-go (embedded StackQL MCP server for Go)

## What this project is

The Go member of the StackQL embedded-MCP family: a Go module that gives
compiled agentic apps an embedded StackQL MCP server (cloud queries and
provisioning over SQL) with no runtime dependencies, no npx, no separate
install. Target repo: `stackql/stackql-mcp-go` (public).

Two layers, built in this order:

1. `embed` package: vendoring kit. Ships helpers to carry the signed stackql
   binary inside a Go program via `go:embed` (caller embeds the platform
   binary; we provide extract-to-cache, sha verification, and a
   `StartServer(ctx, opts) (*Client, error)` that spawns it over stdio and
   returns a connected MCP client using the official
   `github.com/modelcontextprotocol/go-sdk`)
2. (later, upstream-gated) in-process mode: when stackql core exposes a
   public `pkg/mcpserver` API, add `StartInProcess` with the same interface.
   An upstream discussion proposing that API is part of this project's
   milestone 1 - draft it from the embedding contract below.

## The embedding contract (do not deviate)

Source of truth: stackql/stackql-mcpb-packaging (the packaging repo).

- Binaries come from the release .mcpb bundles; per-version sha256 pins are
  published in the release's .sha256 assets (a consolidated platforms.json
  release asset is planned - prefer it once present)
- Canonical launch args (cwd-independence is mandatory; MCP hosts may launch
  with cwd `/` which is read-only on macOS):
  `mcp --mcp.server.type=stdio --approot <home>/.stackql
   --mcp.config {"server": {"mode": "<mode>", "audit": {"disabled": true}}}`
- Default mode: `read_only`. Escalation to safe/delete_safe/full_access is
  an explicit caller opt-in, never a default
- Shared binary cache: `~/.stackql/mcp-server-bin/<version>/<platform-key>/`
  (same path as the npm/pypi wrappers - check before extracting)
- Platform keys: linux-x64, linux-arm64, windows-x64, darwin-universal
- Conformance: the packaging repo's scripts/smoke-test.py `--cmd` mode must
  pass against any launcher this module produces; port it as a Go test too

## Demo app: `sandboxctl` - the on-demand infrastructure concierge

Business use case: developers ask for ephemeral cloud sandboxes in plain
language; the agent plans, prices, provisions, and schedules teardown - with
a cost pre-flight gate before anything is created.

A single compiled binary (the whole point) containing: embedded stackql MCP
server + Claude (anthropic-sdk-go) agent loop.

Flow to demo:

1. `sandboxctl request "a small linux vm and a bucket in sydney for 2 days"`
2. Agent uses read_only stackql tools to: resolve regions/instance types,
   query current quotas, and estimate monthly/diem cost from pricing data
3. Prints a plan + cost estimate; requires `--approve` (or interactive y/N)
   to proceed - the approval boundary is part of the demo's story
4. On approval, re-spawns the embedded server in `safe` mode and provisions
   via lifecycle/mutation tools; tags everything `sandboxctl:expires=<ts>`
5. `sandboxctl reap` finds and tears down expired sandboxes (cron-able)
6. Every step prints the SQL the agent ran - inspectability is the brand

Keep the demo to ONE provider first (google or aws - pick whichever has the
cleanest lifecycle coverage; verify before committing) and structure for
more.

## Build and test

- Go 1.22+, modules; deps: modelcontextprotocol/go-sdk,
  anthropics/anthropic-sdk-go; nothing else without good reason
- `go:embed` the binary per-platform behind build tags; document the
  `go generate` step that downloads + sha-verifies bundles from the release
- Tests: unit (extract/cache/args), integration (spawn + MCP handshake +
  tools/list - mirror the packaging repo smoke), demo e2e gated behind env
  creds. CI: GitHub Actions, linux+macos+windows matrix
- The github null_auth provider is the no-credentials test fixture: pull
  github, list services - same as the packaging repo gate

## Milestones

1. embed package + conformance tests green on 3 OSes + upstream
   `pkg/mcpserver` discussion filed on stackql/stackql
2. sandboxctl demo working against one provider, recorded demo (asciinema)
3. README + GoDoc polish, tag v0.1.0, announce (Go newsletters, r/golang,
   a Go meetup lightning talk: "an agent in a single binary")

## Conventions

- Plain hyphens only (no em dashes); ASCII arrows `->`; no unicode bullets
- Matter-of-fact tone in all docs; no hyperbole
- Stderr for diagnostics, stdout belongs to protocols
- MIT license; mcp-name reference: io.github.stackql/stackql-mcp
