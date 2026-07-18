# MCPB packaging (packaging/mcpb)

Packages the StackQL MCP server into per-platform [MCPB](https://github.com/anthropics/mcpb) bundles (`.mcpb`), publishes them to the matching `stackql/stackql` GitHub release, and pushes the metadata to the Official MCP Registry. Listings on aggregators and the Anthropic Desktop Extensions directory flow from there.

This directory was imported (with full history) from the standalone `stackql/stackql-mcpb-packaging` repo. In-repo, the version pin (`release.yaml`) and the cross-repo release token are gone: the packaging workflow at [.github/workflows/mcp-packaging.yml](/.github/workflows/mcp-packaging.yml) validates `packaging/**` PRs against the latest published release, and publishing is a MANUAL `workflow_dispatch` with the target version. Manual by design: the consumed release assets include the Authenticode-signed Windows exe and the Apple-notarised darwin `.pkg`, both signed outside CI, so the flow is push tag -> create release -> attach the signed assets -> dispatch the workflow. Bundle upload is same-repo via the default `GITHUB_TOKEN`. Local `make` targets are unchanged but now require an explicit `VERSION=X.Y.Z`.

This is a scripted post-release step. It does not build or sign the stackql binaries - that happens upstream in the normal stackql build, code-signing, and notarisation process. Here the already-signed release artefacts are pulled, packaged, and shipped.

The end-user install story is in [docs/install.md](docs/install.md). The marketplace submission checklist is in [docs/anthropic-submission.md](docs/anthropic-submission.md). The broader list of registries and aggregators is in [listings.md](listings.md).

## Table of contents

- [What gets packaged](#what-gets-packaged)
- [Layout](#layout)
- [Prerequisites](#prerequisites)
- [CI release flow (GitHub Actions)](#ci-release-flow-github-actions)
- [Release runbook (local fallback)](#release-runbook-local-fallback)
  - [Step 0 - one-time setup, per machine](#step-0---one-time-setup-per-machine)
  - [Step 1 - build and publish bundles (Machine A: workstation)](#step-1---build-and-publish-bundles-machine-a-workstation)
  - [Step 2 - build and publish darwin (Machine B: Mac)](#step-2---build-and-publish-darwin-machine-b-mac)
  - [Step 3 - render and publish the MCP Registry entry](#step-3---render-and-publish-the-mcp-registry-entry)
  - [Step 4 - submit / refresh aggregator listings](#step-4---submit--refresh-aggregator-listings)
- [Batch commands (the short version)](#batch-commands-the-short-version)
- [Credentials and env vars, at each step](#credentials-and-env-vars-at-each-step)
- [Trust model](#trust-model)
- [Makefile reference](#makefile-reference)
- [Troubleshooting](#troubleshooting)

## What gets packaged

The server is the `stackql` binary itself, launched as `stackql mcp --mcp.server.type=stdio` (see [manifest/manifest.template.json](manifest/manifest.template.json)). The `--mcp.server.type=stdio` flag is required - without it the server starts but does not produce JSON-RPC on stdout. The separate `stackql_mcp_client` binary is a test client and is not packaged.

One bundle is produced per target:

- `stackql-mcp-linux-x64.mcpb`
- `stackql-mcp-linux-arm64.mcpb`
- `stackql-mcp-windows-x64.mcpb`
- `stackql-mcp-darwin-universal.mcpb` (one universal binary covers both Mac arches)

## Layout

```
stackql-mcpb-packaging/
  release.yaml                      # pins the stackql release this repo packages
  .github/workflows/build.yml       # reusable: build + smoke-test all bundles
  .github/workflows/ci.yml          # PRs / main: build + test, no publish
  .github/workflows/publish.yml     # v* tag: verify, build, test, publish
  manifest/manifest.template.json   # MCPB manifest, tokenised (__VERSION__, __BINARY_NAME__)
  registry/server.template.json     # Official MCP Registry server.json, tokenised SHAs + VERSION
  oci/Dockerfile                    # stackql/stackql-mcp image (multi-arch via TARGETARCH)
  npm/                              # @stackql/mcp-server npx wrapper package
  pypi/                             # stackql-mcp-server uvx/pip wrapper package
  scripts/package.sh                # build bundles from bin/ -> dist/
  scripts/clean.sh                  # wipe dist/
  scripts/render-server-json.sh     # pin SHAs into registry/server.json
  scripts/render-npm-manifest.sh    # pin bundle SHAs into npm/platforms.json
  scripts/render-pypi-manifest.sh   # pin bundle SHAs into the pypi package
  scripts/sign.sh                   # envelope-sign dist/*.mcpb + regen .sha256
  scripts/append-signature.py       # frame an externally-produced CMS signature
  scripts/smoke-test.py             # deterministic MCP smoke test (stdlib only)
  scripts/gemini-smoke.py           # optional Gemini Flash agent smoke test
  docs/install.md                   # end-user install guide
  docs/anthropic-submission.md      # Desktop Extensions submission checklist
  listings.md                       # registers and aggregators worth listing on
  Makefile                          # operator entry point
  bin/                              # downloaded release artefacts (gitignored)
  dist/                             # generated bundles + sha256 (gitignored)
```

## Prerequisites

On any machine that builds bundles:

- **Node.js** (the `@anthropic-ai/mcpb` CLI is invoked via `npx`)
- **bash**, **curl**, **unzip**, **sha256sum**/**shasum**
- **GNU Make**
- **Python 3** (stdlib only - for the smoke tests)
- **gh CLI** - https://cli.github.com

Extra, darwin-only:

- **macOS** with **`pkgutil`** (preinstalled) - to extract the universal binary from the notarised `.pkg`. The other three targets have no macOS dependency.

For Step 3:

- **`mcp-publisher`** CLI - https://github.com/modelcontextprotocol/registry/releases/latest

## CI release flow (GitHub Actions)

The primary release path. The stackql release being packaged is pinned in [release.yaml](release.yaml) as `stackql_release: vX.Y.Z`, which is the single source of truth for local `make` defaults, PR CI, and tag publishing.

The sequence:

1. **Upstream release happens** - `stackql/stackql` publishes `vX.Y.Z` with the core assets (per-arch zips and the notarised `.pkg`).
2. **PR bumps the pin** - raise a PR to main changing `stackql_release` in `release.yaml`. [ci.yml](.github/workflows/ci.yml) builds all four bundles against the real release assets and runs the deterministic smoke test on a native runner per platform (`ubuntu-latest`, `ubuntu-24.04-arm`, `windows-latest`, `macos-latest` - the darwin slice runs `pkgutil` on the macos runner). A green PR means the bundles build and the embedded binaries speak MCP.
3. **Merge to main** - nothing is published yet.
4. **Push the matching tag** - `git tag vX.Y.Z && git push origin vX.Y.Z`. [publish.yml](.github/workflows/publish.yml) fails fast if the tag does not exactly match `release.yaml`, rebuilds and re-tests everything, then uploads all `.mcpb` + `.sha256` files to the `stackql/stackql` `vX.Y.Z` release via `make publish` (idempotent `--clobber`).

To re-publish the pinned release without moving the tag (e.g. after enabling signing secrets), run the publish workflow from the Actions tab (`workflow_dispatch`); the `confirm_release` input must be typed exactly as pinned in `release.yaml` (e.g. `v0.10.500`). It runs from current main and clobbers the existing release assets.

**If the MCP Registry entry was already published, a re-publish breaks its SHA pins.** Rebuilt bundles have new SHA-256s, and registry versions are immutable (`cannot publish duplicate version`). Recovery, after the new assets land:

```bash
# refresh the canonical sha files from the release, re-render
for t in linux-x64 linux-arm64 windows-x64 darwin-universal; do
  curl -fsSL -o dist/stackql-mcp-$t.mcpb.sha256 \
    https://github.com/stackql/stackql/releases/download/v0.10.500/stackql-mcp-$t.mcpb.sha256
done
make server-json VERSION=0.10.500
# bump ONLY the registry version fields (URLs keep v0.10.500), e.g. 0.10.500.1,
# then publish and tombstone the stale version:
mcp-publisher publish   # from registry/, after editing version fields
mcp-publisher status --status deleted \
  --message "Release assets rebuilt in place; sha pins stale. Superseded by 0.10.500.1." \
  io.github.stackql/stackql-mcp 0.10.500
```

One-time setup: add a repo secret `STACKQL_RELEASE_TOKEN` - a fine-grained PAT (or GitHub App token) with `contents:write` on `stackql/stackql`. The default `GITHUB_TOKEN` cannot upload assets to another repo. Optionally add `GEMINI_API_KEY` to enable the agent smoke test on the linux-x64 job; without it that step soft-skips.

Optional envelope signing: if the repo secrets `MCPB_SIGNING_CERT` and `MCPB_SIGNING_KEY` (PEM contents, plus optional `MCPB_SIGNING_INTERMEDIATES`) are set, the publish job runs `make sign` to `mcpb sign` every bundle and regenerate its `.sha256` before upload. Without the secrets the step prints a notice and skips, and unsigned bundles ship as before. Note `mcpb verify` in the current CLI is broken upstream (node-forge cannot verify PKCS#7, so every signed bundle reports as unsigned); `make sign` treats it as advisory and asserts the appended signature block instead.

Two more vectors ship from the same pipeline:

- **OCI image** (`docker.io/stackql/stackql-mcp`): built and smoke-tested in PR CI; pushed multi-arch (amd64 + arm64) by the publish workflow when the `DOCKERHUB_USERNAME`/`DOCKERHUB_TOKEN` secrets are set (soft-skips otherwise). The image carries the `io.modelcontextprotocol.server.name` label the MCP Registry requires for oci package validation.
- **PyPI wrapper** (`stackql-mcp-server`): same launcher pattern in stdlib-only Python (uvx/pip), sharing the npm wrapper's binary cache. PR CI smoke-tests it; the publish workflow builds sdist+wheel as a run artifact. Publishing is manual (2FA): `make pypi-build VERSION=X.Y.Z` then `python -m twine upload pypi/dist/*`. The registry validates pypi packages via the `mcp-name:` marker in the README.
- **npm wrapper** (`@stackql/mcp-server`): an npx-able launcher that downloads the platform's published `.mcpb` (sha256-verified against pins baked into the package) and spawns the binary. PR CI tests the wrapper against a locally built bundle; the publish workflow renders the real pins from the published assets and uploads the tarball as a run artifact. Publishing to npmjs is deliberately manual (2FA): download the artifact or run `make npm-pack VERSION=X.Y.Z` locally, then `cd npm && npm publish --access public`. The package carries the `mcpName` field the MCP Registry requires for npm package validation.

A GitHub Action consumes these published bundles for CI: **[stackql/setup-stackql-mcp](https://github.com/stackql/setup-stackql-mcp)** installs the signed binary (sha256-verified at runtime against the release checksums) and emits an `mcpServers` config for agentic workflows (e.g. `anthropics/claude-code-action`). It lives in its own repo and tracks no version - `version: latest` by default, pin with `version: X.Y.Z` - so this packaging repo no longer carries the action source.

Steps 3 and 4 of the local runbook below (MCP Registry publish and aggregator listings) are still manual after a CI publish. After the OCI image and npm package exist for a version, the registry `server.json` (which now includes oci and npm package entries) can be published - the registry validates the npm `mcpName` and the image label at publish time, so those artifacts must exist first.

## Release runbook (local fallback)

The pre-CI flow, kept as a supported fallback (and for the registry/listings steps CI does not cover). Releases are produced locally on two machines because the darwin target needs `pkgutil` (macOS-only). Each machine independently uploads what it built; `--clobber` makes order irrelevant and re-runs safe.

Throughout, `VERSION` is the stackql release minus the leading `v`. For example, tag `v0.10.500` -> `VERSION=0.10.500`. If `VERSION` is omitted, `make` defaults it from `release.yaml`.

### Step 0 - one-time setup, per machine

Done once per machine, not per release.

```bash
git clone https://github.com/stackql/stackql-mcpb-packaging
cd stackql-mcpb-packaging

# Authenticate gh for the upload step
gh auth login
# When prompted, choose: GitHub.com -> HTTPS -> login with web browser.
# The token MUST have 'contents:write' on stackql/stackql.

# Verify the token can write to the right repo
gh release view v0.10.500 --repo stackql/stackql >/dev/null && echo "auth ok"
```

On the Mac additionally install Node.js (`brew install node` or the macOS installer from nodejs.org). `pkgutil`, `make`, `curl`, `unzip`, `shasum`, and `find` are preinstalled.

On the workstation that will run Step 3, also install `mcp-publisher`:

```bash
# macOS / Linux
curl -L "https://github.com/modelcontextprotocol/registry/releases/latest/download/mcp-publisher_$(uname -s | tr '[:upper:]' '[:lower:]')_$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/').tar.gz" \
  | tar xz mcp-publisher && sudo mv mcp-publisher /usr/local/bin/

# log in (one-time, browser flow against your GitHub account)
mcp-publisher login github
```

Your GitHub account must be a member of the `stackql` org with permission to publish under the `io.github.stackql` namespace.

### Step 1 - build and publish bundles (Machine A: workstation)

This machine builds three of four bundles: linux-x64, linux-arm64, windows-x64. The darwin target will skip cleanly with a `pkgutil not found` notice - Machine B handles that one.

```bash
# 1.1 Build everything that can be built here.
make all VERSION=0.10.500

# 1.2 Smoke-test a bundle whose binary this machine can execute.
#     On Linux/WSL: linux-x64. On Windows + Git Bash: windows-x64.
python scripts/smoke-test.py dist/stackql-mcp-linux-x64.mcpb

# 1.3 Optional: Gemini Flash agent soft check. Soft-skips without the key.
GEMINI_API_KEY=... python scripts/gemini-smoke.py dist/stackql-mcp-linux-x64.mcpb

# 1.4 Upload the three bundles + .sha256 files to the stackql/stackql release.
make publish VERSION=0.10.500
```

After this, the release page on `stackql/stackql` has three new `.mcpb` assets and three `.sha256` companions.

### Step 2 - build and publish darwin (Machine B: Mac)

Builds only the darwin slice. Typically a MacInCloud session.

```bash
# 2.1 Pull just the .pkg, extract the universal binary, pack the bundle.
make one TARGET=darwin-universal VERSION=0.10.500

# 2.2 Smoke-test it (this machine can execute the darwin binary).
python scripts/smoke-test.py dist/stackql-mcp-darwin-universal.mcpb

# 2.3 Upload to the same release. Safe before or after Step 1.
make publish VERSION=0.10.500
```

After this, all four bundles are attached to the GitHub release.

### Step 3 - render and publish the MCP Registry entry

Run this on Machine A **after** Machine B has finished its upload, so all four `.sha256` files can be fetched.

```bash
# 3.1 Fetch the darwin .sha256 that Machine B uploaded.
mkdir -p dist
gh release download v0.10.500 \
  --repo stackql/stackql \
  --pattern 'stackql-mcp-darwin-universal.mcpb.sha256' \
  --dir dist --clobber

# 3.2 Render registry/server.json (pins the 4 platform SHAs + version).
make server-json VERSION=0.10.500

# Optional: open registry/server.json and eyeball the SHAs and identifier URLs.

# 3.3 Publish to the Official MCP Registry.
make registry-publish VERSION=0.10.500
```

The first call to `mcp-publisher publish` will require an authenticated session - if you skipped `mcp-publisher login github` in Step 0, do it now.

This call is idempotent at the version level: re-publishing the same `version` overwrites the registry entry; bumping `VERSION` creates a new version that supersedes the previous one (the old one is preserved as historical).

### Step 4 - submit / refresh aggregator listings

Several aggregators **auto-ingest** from the Official MCP Registry. After Step 3, the following appear (or update) within hours to a day without further action:

- mcp.directory
- PulseMCP
- GitHub MCP Registry
- mcpservers.org

The rest are **self-submit** and are tracked in [listings.md](listings.md):

- **Anthropic Desktop Extensions directory** - the high-trust UI signal. See [docs/anthropic-submission.md](docs/anthropic-submission.md) for the full checklist. Submit once; re-submit only on material listing changes (privacy policy, scope, maintainer contact).
- **mcp.so** - largest aggregator, self-submit on the site.
- **Smithery.ai** - `smithery mcp publish` with our `.mcpb` bundle.
- **Glama.ai/mcp** - GitHub auto-discovery; claim the listing for a verified badge.
- **Cursor / VS Code MCP / Cline** - in-client directories; submission paths in [listings.md](listings.md).
- **awesome-mcp-servers**, **mpak.dev**, others as listed.

These are one-shot submissions; routine version bumps don't require resubmission as long as you keep the listing pointing at `releases/latest/`.

## Batch commands (the short version)

If you just want the minimum commands per machine, no smoke test:

```bash
# Machine A (workstation)
make all VERSION=0.10.500 && \
  make publish VERSION=0.10.500

# Machine B (Mac)
make one TARGET=darwin-universal VERSION=0.10.500 && \
  make publish VERSION=0.10.500

# Machine A, after Machine B has finished
gh release download v0.10.500 --repo stackql/stackql \
  --pattern 'stackql-mcp-darwin-universal.mcpb.sha256' \
  --dir dist --clobber && \
  make server-json VERSION=0.10.500 && \
  make registry-publish VERSION=0.10.500
```

## Credentials and env vars, at each step

| Step                                            | What it needs                                                  | Form                          | Notes                                                  |
| ----------------------------------------------- | -------------------------------------------------------------- | ----------------------------- | ------------------------------------------------------ |
| `make all` / `make one`                         | nothing                                                        | -                             | Downloads from public GitHub release assets.           |
| `scripts/smoke-test.py`                         | nothing                                                        | -                             | Talks only to the embedded MCP server.                 |
| `scripts/gemini-smoke.py`                       | `GEMINI_API_KEY`                                               | env var                       | Optional. Soft-skips with exit 0 if unset.             |
| `make publish`                                  | GitHub token with `contents:write` on `stackql/stackql`        | `gh auth login` (token in gh) | Same login on both Machine A and Machine B.            |
| `make server-json`                              | all four `dist/*.sha256` files                                 | files on disk                 | Step 3.1 fetches the darwin one from the release page. |
| `make registry-publish`                         | `mcp-publisher login github` for an account on the stackql org | token in `mcp-publisher` config | Browser-flow OAuth; refresh annually.                  |
| CI publish (`publish.yml`)                      | `STACKQL_RELEASE_TOKEN` repo secret                            | fine-grained PAT              | `contents:write` on `stackql/stackql`. Default `GITHUB_TOKEN` cannot upload cross-repo. |
| Anthropic Desktop Extensions submission        | privacy policy, logo, screenshots, contacts                    | filled into web form          | See [docs/anthropic-submission.md](docs/anthropic-submission.md). |

No secrets are passed via env vars in the build/publish commands themselves - tokens live in the per-tool config of `gh` and `mcp-publisher`. The one exception is `GEMINI_API_KEY` for the optional Gemini soft check.

## Trust model

Layers in place today:

1. **Embedded platform signatures on the bundled binary.** Windows: Authenticode-signed `stackql.exe` (upstream, EV cert on YubiKey). macOS: Developer ID Application signature + Apple notarisation keyed to the binary's cdhash (upstream, via the notarised `.pkg`). Linux: no platform signing by convention.
2. **SHA-256 on the bundle envelope**, written next to every `.mcpb` and published alongside the release assets. Pinned in the Official MCP Registry `server.json`.
3. **Official MCP Registry entry** with verified namespace (`io.github.stackql`). Several aggregators auto-ingest from here.
4. **Anthropic Desktop Extensions directory listing** - the editorial "vetted by Claude" signal users see in Claude Desktop.

Dormant, intentionally:

5. **MCPB envelope signing (`mcpb sign`).** The hooks are in `scripts/package.sh` (`MCPB_SELF_SIGN`, `MCPB_SIGN_CERT`/`MCPB_SIGN_KEY`/`MCPB_SIGN_INTERMEDIATES`) but no envelope signature is applied. Reason: the production EV code-signing cert lives on a YubiKey and the current `@anthropic-ai/mcpb` CLI requires PEM-on-disk for `--cert`/`--key`. There is no PKCS#11 / KSP / engine option, so the YubiKey cannot drive `mcpb sign` directly. The published SHA-256 is what marketplaces verify against today; that is in line with what other third-party MCPB publishers ship. If/when `@anthropic-ai/mcpb` gains HSM support, the hooks are ready and `make signed` will swap from `MCPB_SELF_SIGN=true` to the production cert path with no script changes.

For end users, this means: the binary that actually runs is fully signed by the platform's trust authority; the bundle wrapping it is hash-pinned and registry-verified; the editorial vetting is layered on top via the Anthropic directory. Self-signed envelopes (`make signed`) are for local testing only and not suitable for release.

### Envelope signing with the hardware token

`mcpb sign` requires a PEM key on disk, which the token cannot export. The workaround is to produce the detached CMS signature externally and frame it with [scripts/append-signature.py](scripts/append-signature.py), which emits the same byte layout as `mcpb sign` (`MCPB_SIG_V1` + 4-byte LE length + DER PKCS#7 + `MCPB_SIG_END`) and regenerates the `.sha256`:

```bash
# 1. Sign the unsigned bundle bytes with the token via the PKCS#11 engine
#    (prompts for the token PIN; cert.pem/chain.pem are the public materials
#    exported from the token).
openssl cms -sign -binary -in dist/stackql-mcp-linux-x64.mcpb \
  -signer cert.pem -certfile chain.pem \
  -keyform engine -engine pkcs11 -inkey "pkcs11:type=private" \
  -outform DER -out sig.der

# 2. Frame and append it, regenerating the .sha256.
python scripts/append-signature.py dist/stackql-mcp-linux-x64.mcpb sig.der

# 3. Re-upload (idempotent).
make publish VERSION=X.Y.Z
```

This is interactive (PIN prompt), so it is a local flow, not a CI step. Requires an OpenSSL build with the PKCS#11 engine (libp11) pointed at the token vendor's PKCS#11 module. `--strip-only` removes an existing signature block if you need the unsigned bytes back.

## Publishing the wrapper packages (npm, PyPI, OCI)

Explicit command sequences for the three downstream package vectors. All are
manual publishes (2FA / tokens). The `make` targets are convenience wrappers;
the raw commands below are what they run, useful on machines without `make`
(e.g. Git Bash / WSL on Windows). Run every command FROM THE REPO ROOT unless
a `cd` says otherwise.

Ordering rule for all three: the manifest/pin render steps fetch the canonical
`.sha256` files from the PUBLISHED GitHub release, so they must run AFTER the
`.mcpb` assets for the version are published. Never pin locally built hashes.

### npm (`@stackql/mcp-server`)

```bash
# auth: browser-based login; account must have publish rights on @stackql
npm login

# render pins from the published release, install the one dep, dry-run to inspect
bash scripts/render-npm-manifest.sh --version X.Y.Z
cd npm && npm install --no-audit --no-fund
npm publish --dry-run --access public      # confirm: 4 files, name, version, mcpName
npm publish --access public                # the real publish (prompts for 2FA)
cd ..

# verify from a clean cache (forces download + sha256 verify of the bundle)
rm -rf ~/.stackql/mcp-server-bin
python3 scripts/smoke-test.py --cmd "npx -y @stackql/mcp-server"
```

Note: the npm registry CDN can lag a few minutes after first publish - if
`npx` returns 404 immediately after publishing, confirm with
`npm view @stackql/mcp-server version` and retry once it reports the version.

### PyPI (`stackql-mcp-server`)

PyPI build/publish tooling is not stdlib, and modern Debian/Ubuntu (incl. WSL)
block `pip install` into the system Python (PEP 668), so use a venv:

```bash
# one-time per machine: a venv holding build + twine + uv
python3 -m venv ~/.venvs/stackql-publish
source ~/.venvs/stackql-publish/bin/activate
pip install --upgrade build twine uv

# render pins, build sdist+wheel, validate (run from repo root)
bash scripts/render-pypi-manifest.sh --version X.Y.Z
cd pypi && python3 -m build && python3 -m twine check dist/*

# upload. Auth is a PyPI API token (username __token__, password pypi-...).
# pypi/.pypirc holds it (gitignored); twine needs to be pointed at it since it
# only auto-reads ~/.pypirc. From the pypi/ dir:
python3 -m twine upload --config-file .pypirc dist/*
# (interactive fallback if no .pypirc: python3 -m twine upload dist/* )
cd ..

# verify from a clean cache (uvx; keep the venv active so uv is on PATH)
rm -rf ~/.stackql/mcp-server-bin
python3 scripts/smoke-test.py --cmd "uvx stackql-mcp-server"
```

### OCI image (`docker.io/stackql/stackql-mcp`)

Push locally - do NOT use the dispatch publish workflow for an existing release
(it re-clobbers the `.mcpb` bundles and invalidates every pin). Requires
`docker login` with push rights on `stackql/stackql-mcp`.

```bash
docker login

# stage the linux binaries into the build context (downloads zips if absent)
make oci-stage VERSION=X.Y.Z       # or run the unzip steps the target wraps

# multi-arch build + push
docker buildx create --use --name mcp-builder 2>/dev/null || docker buildx use mcp-builder
docker buildx build --platform linux/amd64,linux/arm64 --push \
  --build-arg VERSION=X.Y.Z \
  -f oci/Dockerfile \
  -t docker.io/stackql/stackql-mcp:X.Y.Z \
  -t docker.io/stackql/stackql-mcp:latest .

# verify from the registry (not the local build cache)
docker pull stackql/stackql-mcp:X.Y.Z
python3 scripts/smoke-test.py --docker stackql/stackql-mcp:X.Y.Z
```

For future tagged releases, set the `DOCKERHUB_USERNAME` / `DOCKERHUB_TOKEN`
repo secrets and the publish workflow pushes the image automatically.

## Makefile reference

```
make VERSION=X.Y.Z              download release artefacts + build all bundles
make download VERSION=X.Y.Z     fetch release artefacts into bin/
make package VERSION=X.Y.Z      build bundles from whatever is in bin/
make <target> VERSION=X.Y.Z     build a single target from current bin/ state
                                (linux-x64|linux-arm64|windows-x64|darwin-universal)
make one TARGET=<t> VERSION=X.Y.Z   download just one target's artefact and build
                                    that one bundle (use on a Mac for darwin)
make signed VERSION=X.Y.Z       build with MCPB_SELF_SIGN=true (testing only)
make sign                       envelope-sign dist/*.mcpb in place and regenerate
                                .sha256 (MCPB_SELF_SIGN=true or MCPB_SIGN_CERT +
                                MCPB_SIGN_KEY; no-ops with a notice when unset)
make oci VERSION=X.Y.Z          build the stackql/stackql-mcp image locally (amd64)
make oci-push VERSION=X.Y.Z     multi-arch image build + push (needs docker login)
make npm-manifest VERSION=X.Y.Z render npm/platforms.json from PUBLISHED .sha256s
make npm-pack VERSION=X.Y.Z     build the @stackql/mcp-server tarball (publish is
                                manual: cd npm && npm publish --access public)
make pypi-manifest VERSION=X.Y.Z render pypi platforms.json from PUBLISHED .sha256s
make pypi-build VERSION=X.Y.Z   build sdist+wheel (publish is manual:
                                python -m twine upload pypi/dist/*)
make publish VERSION=X.Y.Z      upload dist/* to the stackql/stackql release
make server-json VERSION=X.Y.Z  render registry/server.json (pins 4 SHAs)
make registry-publish VERSION=X.Y.Z   render + publish to the Official MCP Registry
make list                       show artefacts present in bin/
make clean                      wipe dist/
make clean-bin                  wipe downloaded artefacts from bin/
```

All version-taking targets accept `VERSION` as either `make X VERSION=X.Y.Z` or `VERSION=X.Y.Z make X`.

## Troubleshooting

**`make all` reports `skip darwin-universal (no stackql_darwin*.pkg ...)`** - expected on non-macOS machines. Run `make one TARGET=darwin-universal VERSION=...` on a Mac (Step 2).

**`make server-json` fails with `missing sha file: dist/stackql-mcp-darwin-universal.mcpb.sha256`** - Step 3.1 hasn't been run, or Machine B hasn't finished its upload yet. Re-run the `gh release download` line in Step 3.1.

**`mcpb pack` fails with `Manifest schema validation passes!` then a write error** - check `dist/` is writable and Node.js is on PATH.

**Claude Desktop shows the bundle as "unsigned"** - expected. See the [Trust model](#trust-model) section. The binary inside is signed and notarised; the envelope is not.

**`mcp-publisher publish` fails with namespace authorisation** - your GitHub account is not on the `stackql` org with the right scope. Either re-run `mcp-publisher login github` after being added, or have an authorised org member run Step 3.

**MCP server starts but Claude Desktop times out connecting** - check the manifest's `args` include `--mcp.server.type=stdio`. Without it the server runs but does not emit JSON-RPC on stdout.

**Smoke test fails with `pull_provider failed: <error>`** - usually a transient connection to `registry.stackql.app`. Re-run.
