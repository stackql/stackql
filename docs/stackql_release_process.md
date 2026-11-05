## StackQL release process

This is the source of truth for publishing a StackQL release across every surface. Hardware tokens, 2FA and vendor review portals are involved, so end to end automation is not possible: each step below says whether it is manual, semi-automatic (a script or `make` target you run) or automatic (CI does it once dispatched). Run the steps in order; several later steps pin the artefacts of earlier ones.

Throughout, `X.Y.Z` is the release version without the leading `v` (for example `0.12.778`) and `{build_number}` is the `Z` component.

### Surfaces at a glance

| Surface | Artefact | How it is published | Step |
| --- | --- | --- | --- |
| GitHub release `vX.Y.Z` | tag, Linux zips and `.deb`, Windows MSI and zips, darwin `.pkg`, `.sha256` sidecars | manual tag + scripts on three machines | 1 to 5 |
| MCP bundles | six `.mcpb` + `.sha256` on the release | `mcp-packaging` dispatch | 6 |
| Docker Hub `stackql/stackql-mcp` | multi-arch OCI image | `mcp-packaging` dispatch | 6 |
| crates.io, NuGet, Go mirror | SDK wrappers | `mcp-packaging` dispatch | 6 |
| npm `@stackql/mcp-server` | launcher package | `mcp-packaging` dispatch (trusted publishing) | 6 |
| releases.stackql.io | download proxy for the release assets | manual tag push | 7 |
| PyPI `stackql-mcp-server` | launcher package | manual (API token) | 8 |
| Official MCP Registry | `server.json` | `mcp-registry-publish` dispatch | 9 |
| Version pins in git | npm, PyPI, both plugin packages | `make npm-manifest` + `make pypi-manifest`, PR | 10 |
| Codex repo marketplace | `.agents/plugins/marketplace.json` | automatic once the pin PR merges | 10 |
| ChatGPT plugin directory | ZIP from `make openai-plugin-zip` | manual upload at platform.openai.com/plugins | 11 |
| Anthropic directory (Claude plugin) | `stackql/stackql-claude-plugin` mirror | `make claude-plugin-mirror`, then the directory scans and publishes | 12 |
| Claude Desktop manual `.mcpb` install | release assets from step 6 | automatic | 6 |
| Assurance | `make check-published` | manual, read-only | 13 |

Other listings (aggregators, IDE directories, awesome lists) ingest from the Official MCP Registry or are one-shot submissions tracked in `packaging/mcpb/listings.md`; nothing per release.

### 1. Push a tag and create a release

Manual. Push a tag using the semver, `v{major}.{minor}.{build_number}`, for example `v0.12.778`.

The `build_number` is the latest successful GitHub Actions build number for the `build` job on a merge to `main`.

```bash
git tag v0.12.778
git push origin v0.12.778
```

Create a release from the tag (set to latest).

### 2. Attach the Linux build artifacts to the release

Semi-automatic. From this repo's root, with the `gh` CLI authenticated, run:

```bash
python cicd/python/attach_build_artifacts.py 0.12.778
```

The script finds build `778` of the `Build` workflow on `main` and checks that the tag points at the commit it built. It then downloads the `stackql_linux_amd64`, `stackql_linux_arm64`, `amd64-artifact-deb` and `arm64-artifact-deb` artifact zips exactly as GitHub stores them, writes a `<asset>.sha256` sidecar for each (sha256sum format, the same as the `.mcpb.sha256` sidecars), uploads the assets and sidecars that are not already on the release, and verifies the digest GitHub recorded for every attached asset against the local bytes. Attached assets are never overwritten: if one is already there with different bytes the script fails and names it. It also downloads the unsigned `stackql_windows_amd64` and `stackql_windows_arm64` zips for step 4 into `dist/release-assets/v0.12.778/` without uploading them. Pass `--dry-run` to see the plan without uploading, and `--skip-deb` to leave the Debian packages out.

### 3. Manual fallback for step 2

Download the following artifacts from build `{build_number}` of the `Build` workflow on `main`:

- `amd64-artifact-deb`
- `arm64-artifact-deb`
- `stackql_linux_amd64`
- `stackql_linux_arm64`
- `stackql_windows_amd64` (used in step 4)
- `stackql_windows_arm64` (used in step 4)

Add the first four to the release as `amd64-artifact-deb.zip`, `arm64-artifact-deb.zip`, `stackql_linux_amd64.zip` and `stackql_linux_arm64.zip`, each with a `<asset>.sha256` sidecar computed from the file as uploaded, for example `sha256sum stackql_linux_amd64.zip > stackql_linux_amd64.zip.sha256`.

### 4. Package, sign and upload the Windows x64 and ARM64 versions

Manual (hardware token). Using the [stackql/stackql-windows-installer](https://github.com/stackql/stackql-windows-installer) project and a Microsoft Authenticode code signing hardware token, create all four Windows release assets:

- `stackql_windows_amd64.msi`
- `stackql_windows_amd64.zip`
- `stackql_windows_arm64.msi`
- `stackql_windows_arm64.zip`

Native Windows ARM64 MSI and ZIP assets are available from `v0.12.778`. Copy both unsigned ZIPs downloaded in step 2 (`dist/release-assets/v{version}/stackql_windows_amd64.zip` and `stackql_windows_arm64.zip`) to `downloaded\stackql_windows_amd64.zip` and `downloaded\stackql_windows_arm64.zip` in the installer repo. From that repo's root, run:

```powershell
.\build.ps1 true
```

This builds both architectures, signs both executables and both MSIs, and creates ZIPs containing the signed executables. Publish all four assets to the existing release:

```powershell
.\release.ps1
```

The upload script checks that all package versions match and the MSI and EXE signatures are valid, writes a `<asset>.sha256` sidecar for each signed package, uploads the assets and sidecars, and verifies the digest GitHub recorded for each attached asset against the local package. Existing assets are skipped, and a sidecar is only uploaded when the attached asset is byte-identical to the local one. The Windows MCPB packages in step 6 consume these signed ZIPs. The `get-stackql.io` PowerShell installer downloads `stackql_windows_arm64.zip` on ARM64 machines, so every release needs it.

### 5. Package, sign, notarize and upload the multi-arch darwin version

Manual (Apple certificate chain). Using a Mac with the correct certificate chain configured (MacInCloud), from the [stackql/stackql-mac-installer](https://github.com/stackql/stackql-mac-installer) repo, run the following:

```bash
sh create-installer.sh {build_number}
# for instance
sh create-installer.sh 778
```

This creates a signed, notarized package file `stackql_darwin_multiarch.pkg`, computes its `stackql_darwin_multiarch.pkg.sha256` sidecar from the stapled package, adds both to the release assets, and verifies the digest GitHub recorded for the attached package. If the package already exists on the release it is skipped, and the sidecar is only uploaded when the attached package is byte-identical to the local one.

### 6. Build and push the MCP packaging (bundles, OCI, SDKs, npm)

Semi-automatic: one dispatch, then CI. Run the [mcp-packaging workflow](https://github.com/stackql/stackql/actions/workflows/mcp-packaging.yml) from `main`, with `version` set to `X.Y.Z`. Start it only after both Linux ZIPs, both signed Windows ZIPs, and the notarized darwin PKG are attached to that release.

The workflow builds these five platform bundles from the release assets:

| Release asset | MCPB package |
| --- | --- |
| `stackql_linux_amd64.zip` | `stackql-mcp-linux-x64.mcpb` |
| `stackql_linux_arm64.zip` | `stackql-mcp-linux-arm64.mcpb` |
| `stackql_windows_amd64.zip` | `stackql-mcp-windows-x64.mcpb` |
| `stackql_windows_arm64.zip` | `stackql-mcp-windows-arm64.mcpb` |
| `stackql_darwin_multiarch.pkg` | `stackql-mcp-darwin-universal.mcpb` |

It also creates `stackql-mcp-multiplatform.mcpb`, containing all five platform binaries and selecting the native Windows x64 or ARM64 executable at launch. The workflow:

- smoke-tests the bundles and wrappers, including native Windows ARM64 on `windows-11-arm`
- attaches all six `.mcpb` bundles and their `.mcpb.sha256` files to the release (these are what Claude Desktop users install manually, and what every wrapper and plugin pins)
- pushes the multi-arch OCI image to Docker Hub
- publishes packages to crates.io (rust), NuGet (dotnet), the Go mirror, and npm (`@stackql/mcp-server`)

Required repository secrets: `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`, `CARGO_REGISTRY_TOKEN`, and `SDK_MIRROR_TOKEN`. NuGet uses Trusted Publishing (OIDC); the manual fallback below uses `NUGET_API_KEY`. npm (`@stackql/mcp-server`) uses npm Trusted Publishing (OIDC) from `mcp-packaging.yml`; no secret.

The version stamps the workflow renders (`package.json`, `pyproject.toml`, `Cargo.toml`, `Directory.Build.props`) live only on the runners and are discarded; step 10 commits the ones that matter.

Manual fallback:

```bash
cd packaging/mcpb
make cargo-publish  VERSION=X.Y.Z   # CARGO_REGISTRY_TOKEN
make go-publish     VERSION=X.Y.Z   # SDK_MIRROR_TOKEN, or an already-authorised git
make dotnet-publish VERSION=X.Y.Z   # NUGET_API_KEY
```

### 7. Make the release available via releases.stackql.io

Manual. Push a tag to [releases.stackql.io](https://github.com/stackql/releases.stackql.io); the tag must match the release tag:

```bash
git tag v0.12.778 && git push origin v0.12.778
```

The npm, PyPI, SDK and plugin launchers download the `.mcpb` bundles through `https://releases.stackql.io/stackql/X.Y.Z/`, so nothing downstream of here works until this tag is live.

### 8. Publish the PyPI wrapper (manual last mile)

The npm wrapper is published by `mcp-packaging` (see 8a); only the PyPI wrapper remains a manual step. Export the environment variables from `packaging/mcpb/.env` first. Run these steps after all five platform bundles and their checksums are published. The manifests pin the published checksums, including `windows-arm64`; npm and PyPI select the matching bundle for the runtime architecture.

#### 8a. npm (`@stackql/mcp-server`) - published by `mcp-packaging`

The `npm-publish` job publishes `@stackql/mcp-server@X.Y.Z` with npm trusted publishing (GitHub Actions OIDC) after the sdk slices pass; no `npm login`, OTP or token is involved, and npm attaches a provenance attestation. The credential is the trusted publisher configuration on npmjs.com, validated by the `1.0.888` publish on 2026-11-05 and permanent from then on. If it is ever deleted or has to be recreated: on npmjs.com open the package's Settings, Trusted Publisher, GitHub Actions, with organization `stackql`, repository `stackql`, workflow filename `mcp-packaging.yml`, environment blank, `Allow npm publish` ticked (the job runs `npm publish` directly, not `npm stage publish`) and `Allow npm dist-tag` left unticked. Create it immediately before the dispatch: npm discards a new configuration that has not completed a successful publish within 2 days, and its fields cannot be edited afterwards. A missing or lapsed configuration fails the job with `npm error 404 Not Found - PUT https://registry.npmjs.org/@stackql%2fmcp-server` right after the dry run succeeds; recreate the configuration and rerun the failed job (the target skips the publish when the version is already on npm).

Verify:

```bash
npm view @stackql/mcp-server@X.Y.Z version dist.attestations.url
```

The job publishes at the end of the run and the local npm cache can hold the previous packument for a few minutes, so wait for the run to complete and add `--prefer-online` if the version is reported missing.

Manual fallback (publish rights on the `@stackql` scope; prompts for an OTP):

```bash
cd packaging/mcpb
make npm-pack VERSION=X.Y.Z
cd npm
npm publish stackql-mcp-server-X.Y.Z.tgz --access public
```

#### 8b. PyPI (`stackql-mcp-server`)

Requires a PyPI API token with upload rights on the project; twine username is `__token__`, password is the token. On Debian/Ubuntu (including WSL) `pip install` outside a venv is blocked by PEP 668, hence the venv.

```bash
cd packaging/mcpb
python3 -m venv ~/.venvs/pypi-pub && source ~/.venvs/pypi-pub/bin/activate
pip install --upgrade build twine
make pypi-build VERSION=X.Y.Z
python -m twine check pypi/dist/*
python -m twine upload pypi/dist/*
```

### 9. Official MCP Registry (`io.github.stackql/stackql-mcp`) - dispatch last

Semi-automatic. Run [mcp-registry-publish](https://github.com/stackql/stackql/actions/workflows/mcp-registry-publish.yml) from `main` with the same version without `v`. The five platform MCPB packages (including `windows-arm64`), OCI image, npm package (published by `mcp-packaging`), and PyPI package must already be published at that version. The registry entry pins each platform bundle's published checksum. Aggregators that ingest from the registry (GitHub MCP Registry, PulseMCP, mcp.directory, mcpservers.org) update on their own within hours.

Manual fallback (if the workflow fails or an out-of-band publish is needed):

Requires the latest `mcp-publisher` CLI and a classic GitHub PAT (scope `read:org` only, no repo scopes) created by a `stackql` org Owner at https://github.com/settings/tokens/new. The server.json renderer reads the five `.sha256` files from the local `dist/` directory, so download the published checksum files from the release first.

Install/upgrade the CLI (linux amd64 shown; assets exist per platform):

```bash
curl -fsSL "https://github.com/modelcontextprotocol/registry/releases/latest/download/mcp-publisher_linux_amd64.tar.gz" | tar xz mcp-publisher
sudo install -m 0755 mcp-publisher /usr/local/bin/mcp-publisher
```

Publish:

```bash
cd packaging/mcpb
mkdir -p dist
for t in linux-x64 linux-arm64 windows-x64 windows-arm64 darwin-universal; do
  curl -fsSL -o dist/stackql-mcp-$t.mcpb.sha256 \
    "https://github.com/stackql/stackql/releases/download/vX.Y.Z/stackql-mcp-$t.mcpb.sha256"
done
export MCP_GITHUB_TOKEN=<classic PAT with read:org scope>
mcp-publisher login github
make registry-publish VERSION=X.Y.Z
```

### 10. Commit the version pins (npm, PyPI, both plugins) and open the pin PR

Semi-automatic. The `npm-publish` job's version stamp lives only on the runner, so git still carries the previous release in `packaging/mcpb/npm/package.json`, `packaging/mcpb/pypi/pyproject.toml` and the two plugin packages. After 8a confirms `@stackql/mcp-server@X.Y.Z` is live, on a branch from `main`:

```bash
cd packaging/mcpb
make npm-manifest VERSION=X.Y.Z    # npm/package.json + both plugin packages (openai-plugin-manifest, claude-plugin-manifest)
make pypi-manifest VERSION=X.Y.Z   # pypi/pyproject.toml
cd ../..
npm view @stackql/mcp-server@X.Y.Z version
python3 packaging/openai-plugin/scripts/validate.py
python3 packaging/mcpb/scripts/validate-claude-plugin.py
python3 packaging/openai-plugin/scripts/smoke-test.py
python3 packaging/openai-plugin/scripts/smoke-test.py --launcher packaging/claude-plugin/server/bin/stackql-mcp.js
```

What the render changes, all of which is committed:

- `packaging/mcpb/npm/package.json` and `packaging/mcpb/pypi/pyproject.toml`: `version` (their `platforms.json` renders are gitignored and re-rendered by CI).
- `packaging/openai-plugin/plugins/stackql/`: `plugin.json` version, the derived `.codex-plugin/plugin.json`, and the vendored payload `server/bin/stackql-mcp.js` (the npm launcher, verbatim), `server/platforms.json` (the pins for `X.Y.Z`), `LICENSE`, `assets/icon.png`, `skills/stackql-setup/SKILL.md`.
- `packaging/claude-plugin/`: `.claude-plugin/plugin.json` version and the same vendored payload.

Both plugins read the plugin from git, so their `platforms.json` is committed, unlike the wrappers'. The validators fail when any vendored file or version disagrees with `packaging/mcpb/npm`. Open a PR (`chore/pins-X.Y.Z`); the `stdio-plugins` workflow repeats validation and the smoke tests on four platforms, re-renders both plugins from the published checksums and fails if the committed payload differs, runs `claude plugin validate --strict`, and builds the ChatGPT directory ZIP as a run artifact. Merge it. The Codex repo marketplace (`codex plugin marketplace add stackql/stackql --sparse .agents/plugins --sparse packaging/openai-plugin`) serves the new pin as soon as the merge lands; nothing else to do for it.

This step does not change the MCPB, PyPI, OCI or MCP Registry artefacts.

### 11. ChatGPT plugin directory (OpenAI)

Manual, from the merged pin PR. The listing is a Skills-only upload of the plugin ZIP that carries the local MCP configuration (desktop only: the listing shows a Desktop only badge, installs from the ChatGPT desktop app, and works in threads that run locally). Source: developers.openai.com/plugins/deploy/desktop-only and deploy/submission.

One-time setup (tracked in stackql#893 and stackql.io#305):

- The submitting organization is allowlisted for local MCP submissions by the OpenAI representative (send the platform organization ID). Business verification completed; submitters other than owners hold **Apps Management Write**.
- `https://stackql.io/terms` and `https://stackql.io/support` are live alongside `/privacy` and `/docs`; the manifest points at all four.
- `review.demo_recording_url` in `packaging/openai-plugin/plugins/stackql/plugin.json` points at a reviewer-accessible recording of the eight test cases (`validate.py` warns while it is unset); the five positive and three negative cases are already in the manifest.
- If the portal asks for domain verification, host the exact token at `https://stackql.io/.well-known/openai-apps-challenge` as plain text.

Per release:

```bash
cd packaging/mcpb
make openai-plugin-zip VERSION=X.Y.Z   # validates, then writes ../openai-plugin/dist/stackql-plugin-X.Y.Z.zip
```

or download the `stackql-chatgpt-plugin-X.Y.Z` artifact from the `stdio-plugins` run on the pin PR; the bytes are the same. Then at https://platform.openai.com/plugins open the existing StackQL plugin (first time: **Create plugin** > **Skills only**), **Upload plugin** with the ZIP, fix anything the Metadata & Skills and MCPs tabs flag (**Copy issues**, fix in `plugin.json`, re-render, re-upload), **Submit for review** with the policy attestations, and after approval **Publish plugin**. Only one review can be active per plugin; reviewer feedback arrives by email.

### 12. Anthropic directory (Claude plugin bundle)

Semi-automatic, from the merged pin PR. Anthropic's directory no longer accepts desktop extension (`.mcpb`) listings; the local server is listed inside the Claude plugin bundle under `packaging/claude-plugin`, which the directory reads from the root of the mirror repository `stackql/stackql-claude-plugin` (a subfolder plugin would be held for a reviewer on every version). The listing loads the server in Claude Code and in Cowork sessions on the user's computer; Claude Desktop chat keeps using the manually installed `.mcpb` from step 6. First-time setup (mirror repository, portal submission, migration of the June 2026 Desktop Extension listing) is in `packaging/mcpb/docs/anthropic-submission.md`.

Per release, after the pin PR merges, with `git` authorised for `stackql/stackql-claude-plugin` (or `SDK_MIRROR_TOKEN` exported):

```bash
cd packaging/mcpb
make claude-plugin-mirror VERSION=X.Y.Z   # validates, then pushes packaging/claude-plugin to the mirror's main and tags vX.Y.Z
```

The push is idempotent and immutable: an existing `vX.Y.Z` tag with a different tree fails (cut the next release instead). The directory picks up the commit through the GitHub push webhook (or its scheduled check), validates and scans it, and the version goes live per the plugin's auto-publish setting; if a reviewer publishes each version, select **Publish** on the plugin's page at https://claude.ai/directory/manage. A version held for a reviewer is not a rejection; the expected hold (credentials read from the user's machine) is documented in the submission checklist.

### 13. Assurance

Manual, read-only. To check publication across all MCP distribution targets, including Windows ARM64 bundles, checksums, and registry pins, run:

```bash
# from within a python venv created previously
# in the `packaging/mcpb` directory, run...
make check-published VERSION=0.12.778
```

Use the version being released; omit `VERSION` to check the latest release. The GitHub MCP Registry row can trail the registry publish by up to two hours. The two vendor directories are not probed: confirm them in their portals (platform.openai.com/plugins, claude.ai/directory/manage).

### Credentials and access, by step

| Step | Needs | Where it lives |
| --- | --- | --- |
| 1 to 3 | push rights on `stackql/stackql`, `gh auth login` | local |
| 4 | Authenticode hardware token, `stackql-windows-installer` | Windows workstation |
| 5 | Apple Developer ID chain, `stackql-mac-installer` | MacInCloud |
| 6 | repo secrets `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`, `CARGO_REGISTRY_TOKEN`, `SDK_MIRROR_TOKEN`; OIDC for npm and NuGet | GitHub Actions |
| 7 | push rights on `stackql/releases.stackql.io` | local |
| 8a | trusted publisher configuration for `@stackql/mcp-server` (package owner) | npmjs.com |
| 8b | PyPI API token (`__token__`) | `packaging/mcpb/.env`, gitignored |
| 9 | OIDC (workflow) or `mcp-publisher login github` with a `read:org` PAT | GitHub Actions or local |
| 10 | push rights on `stackql/stackql` | local |
| 11 | OpenAI platform organization owner or Apps Management Write; local MCP allowlist | platform.openai.com |
| 12 | push rights on `stackql/stackql-claude-plugin` (or `SDK_MIRROR_TOKEN`); a paid claude.ai plan with the GitHub account connected | local, claude.ai/directory/manage |
