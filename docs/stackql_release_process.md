## StackQL release process

Follow these steps to release a new version of `stackql`, as hardware tokens and 2FA is involved in several steps, end to end automation is not possible.  

### 1. Push a tag and create a release

Push a tag using the semver, `v{major}.{minor}.{build_number}`, for example `v0.12.778`

The `build_number` is the latest successful GitHub Actions build number for the `build` job on a merge to `main`  

```bash
git tag v0.12.778
git push origin v0.12.778
```

Create a release from the tag (set to latest)

### 2. Download Artifacts from Latest Build

Download the following artifacts from the latest build on the `main` branch including:

- `amd64-artifact-deb`
- `arm64-artifact-deb`
- `stackql_linux_amd64`
- `stackql_linux_arm64`
- `stackql_windows_amd64` (used in step 4)
- `stackql_windows_arm64` (used in step 4)

### 3. Add the following assets to the release:

- `amd64-artifact-deb` (downloaded in step 2)
- `arm64-artifact-deb` (downloaded in step 2)
- `stackql_linux_amd64.zip` (downloaded in step 2)
- `stackql_linux_arm64.zip` (downloaded in step 2)

### 4. Package, Sign and Upload Windows x64 and ARM64 Versions

Using the [stackql/stackql-windows-installer](https://github.com/stackql/stackql-windows-installer) project and a Microsoft Authenticode code signing hardware token, create all four Windows release assets:

- `stackql_windows_amd64.msi`
- `stackql_windows_amd64.zip`
- `stackql_windows_arm64.msi`
- `stackql_windows_arm64.zip`

Native Windows ARM64 MSI and ZIP assets are available from `v0.12.778`. Save both unsigned ZIPs from the same build in step 2 as `downloaded\stackql_windows_amd64.zip` and `downloaded\stackql_windows_arm64.zip` in the installer repo. From that repo's root, run:

```powershell
.\build.ps1 true
```

This builds both architectures, signs both executables and both MSIs, and creates ZIPs containing the signed executables. Publish all four assets to the existing release:

```powershell
.\release.ps1
```

The upload script checks that all package versions match and the MSI and EXE signatures are valid. Existing assets are skipped. The Windows MCPB packages in step 6 consume these signed ZIPs. The `get-stackql.io` PowerShell installer downloads `stackql_windows_arm64.zip` on ARM64 machines, so every release needs it.

### 5. Package, Sign, Notarize and Upload the Multi Arch Darwin Version

Using a Mac with the correct certificate chain configured (MacInCloud), from the [stackql/stackql-mac-installer](https://github.com/stackql/stackql-mac-installer) repo, run the following:

```bash
sh create-installer.sh {build_number}
# for instance
sh create-installer.sh 778
```

this will create a signed, notarized package file `stackql_darwin_multiarch.pkg` and add it to the relase assets.

### 6. Build and push MCPB assets to the release

Run the [mcp-packaging workflow](https://github.com/stackql/stackql/actions/workflows/mcp-packaging.yml) from `main`, with `version` set to the release version without `v` (for example, `0.12.778`). Start it only after both Linux ZIPs, both signed Windows ZIPs, and the notarized Darwin PKG are attached to that release.

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
- attaches all six `.mcpb` bundles and their `.mcpb.sha256` files to the release
- pushes the multi-arch OCI image to Docker Hub
- publishes packages to crates.io (rust), nuget (dotnet), and the go mirror

Required repository secrets: `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`, `CARGO_REGISTRY_TOKEN`, and `SDK_MIRROR_TOKEN`. NuGet uses Trusted Publishing (OIDC); the manual fallback below uses `NUGET_API_KEY`.

manual fallback:

```bash
cd packaging/mcpb
make cargo-publish  VERSION=X.Y.Z   # CARGO_REGISTRY_TOKEN
make go-publish     VERSION=X.Y.Z   # SDK_MIRROR_TOKEN, or an already-authorised git
make dotnet-publish VERSION=X.Y.Z   # NUGET_API_KEY
```

### 7.  Make the release available via **releases.stackql.io**

Push a tag to [releases.stackql.io](https://github.com/stackql/releases.stackql.io), eg:

> note: the tag needs to match the tag for the release

```
git tag v0.12.778 && git push origin v0.12.778
```

### 8. Publish the MCP wrapper packages (manual last mile)

Export the environment variables from `packaging/mcpb/.env` first. Run these steps after all five platform bundles and their checksums are published. The manifests pin the published checksums, including `windows-arm64`; npm and PyPI select the matching bundle for the runtime architecture.

#### 8a. npm (`@stackql/mcp-server`)

Requires an `npm login` session as a user with publish rights on the `@stackql` scope; the publish prompts for an OTP.

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

Run [mcp-registry-publish](https://github.com/stackql/stackql/actions/workflows/mcp-registry-publish.yml) from `main` with the same version without `v`. The five platform MCPB packages (including `windows-arm64`), OCI image, npm package, and PyPI package must already be published at that version. The registry entry pins each platform bundle's published checksum.

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

### 10. Commit the ChatGPT/Codex stdio plugin bump

`make npm-pack` in 8a also stamped the plugin: `version` in `packaging/openai-plugin/plugins/stackql/.codex-plugin/plugin.json` and the pinned `@stackql/mcp-server@X.Y.Z` in `packaging/openai-plugin/plugins/stackql/bin/stackql-mcp.js` (`make openai-plugin-manifest VERSION=X.Y.Z` stamps just those two). After 8a confirms `@stackql/mcp-server@X.Y.Z` is live, verify and commit them together with the `package.json` / `pyproject.toml` bumps; the plugin validator fails when the pin and `packaging/mcpb/npm/package.json` disagree:

```bash
npm view @stackql/mcp-server@X.Y.Z version
python3 packaging/openai-plugin/scripts/validate.py
python3 packaging/openai-plugin/scripts/smoke-test.py
```

The marketplace entry is version-independent. This step does not change the MCPB, Anthropic, PyPI, OCI, or MCP Registry artifacts. See `packaging/mcpb/README.md` for the detailed addendum.

## Assurance

To check publication across all MCP distribution targets, including Windows ARM64 bundles, checksums, and registry pins, run:

```bash
# from within a python venv created previously
# in the `packaging/mcpb` directory, run...
make check-published VERSION=0.12.778
```

Use the version being released; omit `VERSION` to check the latest release.
