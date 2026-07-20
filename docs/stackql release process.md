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

Push a tag using the semver, `{major}.{minor}.{build_number}`, for example `0.10.557`  

The `build_number` is the latest successful GitHub Actions build number for the `build` job on a merge to `main`  

```
git tag v0.10.557
git push origin v0.10.557
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

7. Publish the MCP wrapper packages (manual last mile)

The `mcp-packaging` workflow attaches the `.mcpb` bundles (and `.sha256` files) to the release and pushes the multi-arch OCI image to Docker Hub automatically. The npm, PyPI and MCP Registry wrappers need interactive (2FA) credentials, so they cannot be automated end to end and are published from a local clone using the steps below (a Linux/WSL/macOS shell, from the repo root).

All of the render targets (`make npm-pack` / `make pypi-build` / `make server-json`) consume the published `.sha256` release assets, so run them only after step 6 completes. The renderers also stamp the version into `packaging/mcpb/npm/package.json` and `packaging/mcpb/pypi/pyproject.toml` in place - commit or revert the stamps afterwards.

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

7c. Official MCP Registry (`io.github.stackql/stackql-mcp`)

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

8. Push the same release version tag to [stackql/releases.stackql.io](https://github.com/stackql/releases.stackql.io)
