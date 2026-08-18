## StackQL release process

1. Download Artifacts from Latest Build

Download the following artifacts from the latest build on the `main` branch including:

- `amd64-artifact-deb`
- `arm64-artifact-deb`
- `stackql_linux_amd64`
- `stackql_linux_arm64`
- `stackql_windows_amd64` (used in step 2)

2. Package and Sign Windows Version

Using the  [stackql/stackql-msi](https://github.com/stackql/stackql-msi) project along with a Microsoft Authenticode CodeSigning hardware token, create the windows packages:

- `stackql_windows_amd64.msi`
- `stackql_windows_amd64.zip`

3. Package, Sign and Notarize the Multi Arch Darwin Version

Using a Mac with the correct certificate chain configured (MacInCloud), run the [stackql/stackql-mac-installer](https://github.com/stackql/stackql-mac-installer), upload the package binary `stackql_darwin_multiarch.pkg` to Google Drive.  

Download the signed, notarized package file:

- `stackql_darwin_multiarch.pkg`

4. Push a tag and create a release

Push a tag using the semver, `{major}.{minor}.{build_number}`, for example `0.10.591`  

The `build_number` is the latest successful GitHub Actions build number for the `build` job on a merge to `main`  

```
git tag v0.10.591
git push origin v0.10.591
```

Create a release from the tag (set to latest)

5. Add the following assets to the release:

- `amd64-artifact-deb` (downloaded in step 1)
- `arm64-artifact-deb` (downloaded in step 1)
- `stackql_linux_amd64.zip` (downloaded in step 1)
- `stackql_linux_arm64.zip` (downloaded in step 1)
- `stackql_windows_amd64.msi` (built in step 2)
- `stackql_windows_amd64.zip` (built in step 2)
- `stackql_darwin_multiarch.pkg` (built in step 3)

6. Build and push MCPB assets to the release

Invoke the `mcp-packaging` workflow in [stackql/stackql](https://github.com/stackql/stackql)

7. Make the release available via **releases.stackql.io**

Push a tag to [releases.stackql.io](https://github.com/stackql/releases.stackql.io), eg:

> note: the tag needs to match the tag for the release

```
git tag v0.10.591 && git push origin v0.10.591
```

8. Publish the MCP wrapper packages (manual last mile)

The `mcp-packaging` workflow attaches the `.mcpb` bundles (and `.sha256` files) to the release, pushes the multi-arch OCI image to Docker Hub, validates the six embedded SDK wrappers, and publishes the three in the published tier (7d-7f) automatically from repo secrets. The npm and PyPI wrappers need interactive (2FA) credentials, so they are published from a local clone using the steps below (a Linux/WSL/macOS shell, from the repo root). The MCP Registry entry is then published by dispatching the `mcp-registry-publish` workflow (7c).

Order matters and is strictly one-way: 7a, 7b and 7d-7f (any order among themselves) only after step 6 completes, and 7c strictly last. The registry validates every package entry in server.json against its upstream registry at the exact version when publishing - if the npm or PyPI wrapper (or the OCI image) for the version is not live yet, the registry publish is rejected. There is no cycle: nothing references the registry entry, it references everything else. The lettering is kept stable for cross-references; the textual order below is the execution order.

Every wrapper follows one contract: wrapper version == stackql release version, and its per-platform sha256 pins are data in a `platforms.json` rendered by `packaging/mcpb/scripts/render-platforms.sh` (`make manifests VERSION=X.Y.Z` renders all nine vectors, `make <vector>-manifest` one) from the published `.sha256` release assets - so render only after step 6 completes, and never pin locally built bundle hashes. The renderer also stamps the version into each vector's manifest file (`npm/package.json`, `pypi/pyproject.toml`, `cargo/Cargo.toml`, `dotnet/Directory.Build.props`, `gleam/gleam.toml`, `kotlin/gradle.properties`, `swift/Sources/StackQLMCP/Version.swift`) in place - commit or revert the stamps afterwards.

Standing rule: release assets are immutable once a wrapper has pinned them. If a `.mcpb` bundle must be rebuilt, cut a new patch release; never re-upload over an existing asset (the v0.10.500 re-upload of 2026-06-20 broke every published sidecar wrapper's sha256 check at once).

7a. npm (`@stackql/mcp-server`)

Requires an `npm login` session as a user with publish rights on the `@stackql` scope; the publish prompts for an OTP.

```
cd packaging/mcpb
make npm-pack VERSION=X.Y.Z
cd npm
npm publish stackql-mcp-server-X.Y.Z.tgz --access public
```

7b. PyPI (`stackql-mcp-server`)

Requires a PyPI API token with upload rights on the project; twine username is `__token__`, password is the token. On Debian/Ubuntu (including WSL) `pip install` outside a venv is blocked by PEP 668, hence the venv.

```
cd packaging/mcpb
python3 -m venv ~/.venvs/pypi-pub && source ~/.venvs/pypi-pub/bin/activate
pip install --upgrade build twine
make pypi-build VERSION=X.Y.Z
python -m twine check pypi/dist/*
python -m twine upload pypi/dist/*
```

7d-7f. Embedded SDK wrappers, published tier (cargo, go, dotnet) - dispatch handles it

The `mcp-packaging` dispatch from step 6 runs the `sdk` matrix for all six SDK vectors (render pins for the version, lint, unit tests, package, then `scripts/smoke-test.py --cmd` against each launcher with a real download through `releases.stackql.io`) and, when every slice is green, one publish job per published-tier vector from repo secrets:

| Step | Vector | Registry / coordinate | Secret(s) | Verify |
|---|---|---|---|---|
| 7d | cargo | crates.io `stackql-mcp` | `CARGO_REGISTRY_TOKEN` | `cargo info stackql-mcp` |
| 7e | go | mirror `stackql/stackql-mcp-go`, tag `vX.Y.Z` | `SDK_MIRROR_TOKEN` | `go list -m github.com/stackql/stackql-mcp-go@vX.Y.Z` |
| 7f | dotnet | NuGet `StackQL.Mcp` + `StackQL.Mcp.AgentFramework` | none - NuGet Trusted Publishing (OIDC) policy `stackql-mcp-packaging` on nuget.org; manual fallback needs a scoped API key in `NUGET_API_KEY` | `nuget list StackQL.Mcp` |

The publish jobs are idempotent where the registry allows (they skip when the version already exists) and each prints the resulting coordinate. `SDK_MIRROR_TOKEN` is a fine-grained PAT with `contents: write` on `stackql/stackql-mcp-go`.

Preview tier (gleam, kotlin, swift): in tree, rendered and CI-validated by the same `sdk` matrix on every packaging PR and dispatch, but NOT published and not part of the release train. `make gleam-publish` / `make kotlin-publish` / `make swift-publish` exist for a deliberate out-of-band publish (hex key; Central Portal + GPG; mirror push to `stackql/stackql-mcp-swift`) - if a preview vector is promoted, add its publish job to `mcp-packaging.yml` and its row above.

Manual fallback (a workflow failure, or an out-of-band publish) - the same Makefile targets from a local clone, with the credentials in the environment:

```
cd packaging/mcpb
make cargo-publish  VERSION=X.Y.Z   # CARGO_REGISTRY_TOKEN
make go-publish     VERSION=X.Y.Z   # SDK_MIRROR_TOKEN, or an already-authorised git
make dotnet-publish VERSION=X.Y.Z   # NUGET_API_KEY
```

Each `<vector>-publish` target renders the manifest for `VERSION` first, so it also carries the ordering rule (after step 6). Validate before publishing with `make <vector>-build VERSION=X.Y.Z` and `make <vector>-smoke VERSION=X.Y.Z` (Swift needs a Mac).

7c. Official MCP Registry (`io.github.stackql/stackql-mcp`) - dispatch last

After 7a and 7b are live, dispatch the `mcp-registry-publish` workflow in [stackql/stackql](https://github.com/stackql/stackql) with the version. It needs no credentials: it authenticates with `mcp-publisher login github-oidc` (the registry grants `io.github.<repository_owner>/*` to GitHub Actions OIDC tokens, so no secret is stored) and renders server.json from the release `.sha256` assets. A preflight step fails fast with a pointer to the missing step if the npm, PyPI or OCI package for the version is not yet live.

Manual fallback (if the workflow fails or an out-of-band publish is needed):

Requires the latest `mcp-publisher` CLI and a classic GitHub PAT (scope `read:org` only, no repo scopes) created by a `stackql` org Owner at https://github.com/settings/tokens/new. The server.json renderer reads the four `.sha256` files from the local `dist/` directory, so download the published checksum files from the release first.

Install/upgrade the CLI (linux amd64 shown; assets exist per platform):

```
curl -fsSL "https://github.com/modelcontextprotocol/registry/releases/latest/download/mcp-publisher_linux_amd64.tar.gz" | tar xz mcp-publisher
sudo install -m 0755 mcp-publisher /usr/local/bin/mcp-publisher
```

Publish:

```
cd packaging/mcpb
for t in linux-x64 linux-arm64 windows-x64 darwin-universal; do
  curl -fsSL -o dist/stackql-mcp-$t.mcpb.sha256 \
    "https://github.com/stackql/stackql/releases/download/vX.Y.Z/stackql-mcp-$t.mcpb.sha256"
done
export MCP_GITHUB_TOKEN=<classic PAT with read:org scope>
mcp-publisher login github
make registry-publish VERSION=X.Y.Z
```

Gotchas (all hit during the v0.10.557 release):

- Use the latest `mcp-publisher` from https://github.com/modelcontextprotocol/registry/releases. The current schema version is baked into the binary, so a stale CLI fails with a misleading "deprecated schema detected" error even when `server.template.json` pins the correct schema.
- Log in with a classic PAT via `MCP_GITHUB_TOKEN` as above, not the interactive device flow. The registry grants the `io.github.stackql/*` namespace only to `stackql` org Owners, and it checks the role via `GET /user/memberships/orgs`. The device-flow login is a GitHub App user token ("MCP Registry Login") that cannot see the org membership unless that app is installed on the org, so it 403s with a misleading hint about public org membership - the PAT path avoids all of that.
- The registry JWT minted at login is short-lived - run `make registry-publish` immediately after `mcp-publisher login github`.
- Old `mcp-publisher` versions (pre-1.2) drop `.mcpregistry_*` token files in the working directory (gitignored); current versions store the token in `~/.config/mcp-publisher/token.json`.

7d. Update the ChatGPT/Codex stdio plugin

After 7a confirms `@stackql/mcp-server@X.Y.Z` is live, update the local plugin in a follow-up PR:

1. Set `version` in `packaging/openai-plugin/plugins/stackql/.codex-plugin/plugin.json` to `X.Y.Z`.
2. Set the pinned `@stackql/mcp-server@X.Y.Z` in `packaging/openai-plugin/plugins/stackql/bin/stackql-mcp.js`.
3. Run:

```
npm view @stackql/mcp-server@X.Y.Z version
python3 packaging/openai-plugin/scripts/validate.py
python3 packaging/openai-plugin/scripts/smoke-test.py
```

The marketplace entry is version-independent. This step does not change the MCPB, Anthropic, PyPI, OCI, or MCP Registry artifacts. See `packaging/mcpb/README.md` for the detailed addendum.

8. Push the same release version tag to [stackql/releases.stackql.io](https://github.com/stackql/releases.stackql.io)
