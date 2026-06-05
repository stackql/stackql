# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

A standalone, scripted post-release step that packages the StackQL MCP server into per-platform [MCPB](https://github.com/anthropics/mcpb) bundles (`.mcpb`) for distribution and listing on the official MCP Registry.

This repo does NOT build or sign the stackql binaries - that happens upstream in the normal stackql build/signing process. Here you drop the already-signed release artefacts (per-arch zips and the notarised macOS `.pkg`) into `bin/`, run one script, and get signed `.mcpb` bundles plus checksums in `dist/` to attach to the matching GitHub release.

The server packed into each bundle is the `stackql` binary itself, launched as `stackql mcp --mcp.server.type=stdio` (see [manifest/manifest.template.json](manifest/manifest.template.json)). The `--mcp.server.type=stdio` flag is required - without it the MCP server does not produce JSON-RPC on stdout. The separate `stackql_mcp_client` binary is a test client and is NOT packaged.

## Common commands

A [Makefile](Makefile) wraps `scripts/package.sh` for the common flows. The script is still the source of truth; `make` is convenience.

One-shot from a clean checkout - downloads the release artefacts from `https://github.com/stackql/stackql/releases/download/v<VERSION>/...` into `bin/`, then builds every available bundle:

```bash
make all VERSION=X.Y.Z
# 'make VERSION=X.Y.Z' is equivalent ('all' is the default target)
```

Just download (skip packaging):

```bash
make download VERSION=X.Y.Z
```

Just package whatever is already in `bin/` (skip downloading):

```bash
make package VERSION=X.Y.Z
# or call the script directly:
./scripts/package.sh --version X.Y.Z
```

Build a single target. Two variants:

```bash
# Download just that target's source artefact and build only that bundle.
# Use this on a Mac to do the darwin slice in the two-machine release flow.
make one TARGET=darwin-universal VERSION=X.Y.Z
make one TARGET=linux-x64        VERSION=X.Y.Z

# Build from already-present artefacts in bin/ (temporarily hides the
# others under bin/.hidden/ and restores them after).
make linux-x64        VERSION=X.Y.Z
make linux-arm64      VERSION=X.Y.Z
make windows-x64      VERSION=X.Y.Z
make darwin-universal VERSION=X.Y.Z
```

Self-signed bundles (testing only - production envelope signing is not currently wired up; see "Trust model" below):

```bash
make signed VERSION=X.Y.Z
```

Upload everything in `dist/` to the matching `stackql/stackql` release (requires `gh auth login` with `contents:write` on `stackql/stackql`; idempotent via `--clobber`):

```bash
make publish VERSION=X.Y.Z
```

Wipe outputs / inputs:

```bash
make clean        # remove dist/*.mcpb and *.sha256
make clean-bin    # remove downloaded artefacts from bin/
```

Show what is currently in the drop-zone:

```bash
make list
```

## Release flow (fully local, no CI)

There is no GitHub Actions workflow. Releases are produced by running `make` on two machines, because the darwin target needs `pkgutil` which only exists on macOS.

### The two-machine flow

**Machine A (your workstation, any OS with bash + node + unzip):**

```bash
make all VERSION=X.Y.Z      # downloads release artefacts, builds linux-x64,
                            # linux-arm64, windows-x64. Darwin skips with a
                            # 'pkgutil not found' notice.
python scripts/smoke-test.py dist/stackql-mcp-linux-x64.mcpb   # gate
make publish VERSION=X.Y.Z  # uploads the 3 bundles + .sha256s to the
                            # stackql/stackql release matching v<VERSION>
```

**Machine B (a Mac - MacInCloud is the typical case):**

```bash
git clone https://github.com/stackql/stackql-mcpb-packaging
cd stackql-mcpb-packaging
make one TARGET=darwin-universal VERSION=X.Y.Z   # downloads only the .pkg,
                                                 # extracts the universal
                                                 # binary, builds 1 bundle
python scripts/smoke-test.py dist/stackql-mcp-darwin-universal.mcpb
make publish VERSION=X.Y.Z   # uploads just that one bundle + sha
```

Each machine runs `gh auth login` once with a token that has `contents:write` on `stackql/stackql`. `make publish` uses `gh release upload --clobber`, so it is idempotent and the order between the two machines does not matter. Re-running either step is safe.

The Mac machine only needs Node.js (for `mcpb` via `npx`) on top of the default macOS toolchain - `make`, `curl`, `unzip`, `pkgutil`, `shasum`, `find` are all preinstalled.

### Smoke tests

Two layers, both in [scripts/](scripts/):

- **[scripts/smoke-test.py](scripts/smoke-test.py)** - deterministic gate. Extracts the `.mcpb`, spawns `stackql mcp --mcp.server.type=stdio --auth='{"github":{"type":"null_auth"}}'`, runs the JSON-RPC handshake, asserts `tools/list` contains `pull_provider`/`list_services`/`list_providers`, calls `pull_provider` for `github`, then `list_services` and confirms real github services come back. Stdlib only. Run before `make publish`:

  ```bash
  python scripts/smoke-test.py dist/stackql-mcp-linux-x64.mcpb
  ```

- **[scripts/gemini-smoke.py](scripts/gemini-smoke.py)** - optional agent check using Gemini Flash. Exposes the MCP tools to Gemini via function calling and asks it to pull github and list services. Skips with exit 0 if `GEMINI_API_KEY` is not set; on failure prints `WARN:` and exits 0. Stdlib only - calls `generativelanguage.googleapis.com` directly via `urllib`. `GEMINI_MODEL` defaults to `gemini-2.0-flash`.

Both scripts use the `github` provider in `null_auth` mode so no credentials are needed - they hit the public github registry endpoints.

## Trust model

The end goal is signed, verifiable, functional MCP binary assets distributed through trusted marketplaces. Today the layers are:

1. **Mach-O / Authenticode signatures on the embedded binary** - applied upstream during the stackql release build. Windows: Authenticode-signed `stackql.exe`. macOS: Developer ID Application signature embedded in the universal `stackql` binary inside the `.pkg`, plus Apple notarisation keyed to the binary's cdhash. Linux: no platform-level signing, by convention.
2. **SHA-256 on the bundle envelope** - written by `package.sh` next to every `.mcpb`. Published with the bundle and pinned in the official MCP Registry `server.json`. Anyone installing the bundle can verify the bytes.
3. **MCPB envelope signature (`mcpb sign`)** - currently *not applied*. The hooks are in `package.sh` (`MCPB_SELF_SIGN`, `MCPB_SIGN_CERT`/`MCPB_SIGN_KEY`/`MCPB_SIGN_INTERMEDIATES`) and remain dormant until envelope signing is wired up.
4. **Anthropic Desktop Extensions directory listing** - the editorial "vetted by Claude" signal that users see in Claude Desktop's Browse Extensions UI. Submission is via the review form at `claude.com/docs/connectors/building/submission`; requirements (privacy policy, logo, screenshots) are in [listings.md](listings.md).
5. **Official MCP Registry entry** - canonical metadata pointing at the GitHub release assets and pinning their SHA-256.

The notarised `.pkg` does the load-bearing trust work for macOS users: Gatekeeper validates the cdhash online when the bundled binary launches, so the binary inside the `.mcpb` is the same trusted binary users get from the `.pkg` installer. The unsigned `.mcpb` envelope is a Claude Desktop UI signal, not a Gatekeeper signal - addressing it requires either a self-signed cert (low value) or a real code-signing cert held in an HSM (the production answer). Until then, the registry SHA-256 plus the embedded platform signatures are what marketplaces verify against.

Self-signed bundle (testing only):

```bash
MCPB_SELF_SIGN=true ./scripts/package.sh --version X.Y.Z
```

Production-signed bundle:

```bash
MCPB_SIGN_CERT=cert.pem \
MCPB_SIGN_KEY=key.pem \
MCPB_SIGN_INTERMEDIATES="intermediate-ca.pem root-ca.pem" \
./scripts/package.sh --version X.Y.Z
```

`MCPB_SIGN_INTERMEDIATES` is optional and space-separated. Bundle signing is OFF by default. When unset, the script prints a notice and skips.

The script invokes `mcpb` if on PATH, otherwise falls back to `npx --yes @anthropic-ai/mcpb`.

## Bin drop-zone layout (required before running package.sh)

`bin/` is gitignored except for its `README.md` and `.gitignore`. `package.sh` reads the release artefacts directly - no manual extraction. Expected files at the root of `bin/`:

```
bin/
  stackql_linux_amd64.zip        # contains stackql
  stackql_linux_arm64.zip        # contains stackql
  stackql_windows_amd64.zip      # contains stackql.exe (Authenticode-signed upstream)
  stackql_darwin_multiarch.pkg   # notarised .pkg, universal binary inside
```

The darwin glob is `stackql_darwin*.pkg`, so any suffix works. Any target whose source artefact is absent is skipped with a notice - partial drops produce partial bundle sets.

A legacy fallback layout (pre-extracted binaries at `bin/<arch>/stackql[.exe]` and `bin/darwin/*.pkg`) is also accepted. The release-artefact layout takes precedence when both are present.

## Architecture and flow

Single bash script ([scripts/package.sh](scripts/package.sh)) drives everything. For each target:

1. Stage a temp dir with `server/<binary-name>` and a per-target `manifest.json` rendered from [manifest/manifest.template.json](manifest/manifest.template.json) by `sed`-substituting `__VERSION__` and `__BINARY_NAME__`.
2. `mcpb validate` the manifest, then `mcpb pack` the staging dir into `dist/stackql-mcp-<label>.mcpb`.
3. Optionally `mcpb sign` (self-signed or with cert/key), then `mcpb verify`.
4. Write `<bundle>.sha256` next to the bundle, with the basename matching the released filename so the checksum line matches the GitHub release asset name.

Per-target labels: `linux-x64`, `linux-arm64`, `windows-x64`, `darwin-universal`.

### macOS extraction (`extract_pkg_binary`)

The darwin target reads a notarised `.pkg`, not a bare binary, because:

- The Mach-O has the code signature embedded, so extraction preserves it.
- Notarisation is keyed to the binary's cdhash (registered with Apple when notarising the `.pkg`); the identical extracted binary is recognised by Gatekeeper online.
- The stapled notarisation ticket lives on the `.pkg` (you cannot staple a bare binary). The `.pkg` remains the offline-validating installer; the `.mcpb` relies on online validation of the same binary.

`extract_pkg_binary` runs `pkgutil --expand-full` and locates `stackql` inside the payload. The darwin target requires `pkgutil`, so it only runs on macOS. The other three targets have no macOS dependency.

### Manifest template

[manifest/manifest.template.json](manifest/manifest.template.json) is tokenised with `__VERSION__` and `__BINARY_NAME__`. The runtime invocation is `${__dirname}/server/<binary>` with `args: ["mcp", "--mcp.server.type=stdio"]`. The `--mcp.server.type=stdio` flag is required: without it the MCP server starts but does not emit JSON-RPC on stdout (the default transport differs). If you ever need to pass more flags (registry path, auth context, etc.), update `args` here - clients launch the bundled binary with precisely those arguments.

## Releasing

After packaging, attach all of `dist/*.mcpb` and `dist/*.sha256` to the same GitHub release as the matching stackql build.

For the official MCP Registry, the `.mcpb` URL must contain the string `mcp` (the filenames already do). Each platform gets one `server.json` package entry: `registryType: mcpb`, `identifier` = release download URL, `fileSha256` = the matching `.sha256`, `transport.type: stdio`. Publish with the `mcp-publisher` CLI.

[listings.md](listings.md) is the working register of every venue worth listing on (registries, aggregators, IDE/client directories, awesome lists) with submission status. Treat the official MCP Registry as the hub - many other venues auto-ingest from it.

## Writing style (from global instructions)

- Plain hyphens only: `-`, never `--`. ASCII arrows `->`, never `->` or `=>` or `<-`.
- Stick to QWERTY characters. No em dashes, smart quotes, or other punctuation that is not on a standard keyboard.
- Matter-of-fact tone. No hyperbole, no sycophancy.
- No stacked headings (an H1 immediately followed by an H2 with no content between).
