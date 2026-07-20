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

The `mcp-packaging` workflow attaches the `.mcpb` bundles (and `.sha256` files) to the release and pushes the multi-arch OCI image to Docker Hub automatically. The remaining venues need interactive (2FA) credentials, so the workflow only builds the artifacts; publishing them is manual:

- PyPI (`stackql-mcp-server`): download the `pypi-dist` artifact from the workflow run, or build locally. Upload with twine using username `__token__` and a PyPI API token as the password. On Debian/Ubuntu (including WSL) `pip install` is blocked by PEP 668, so use a venv:

  ```
  python3 -m venv ~/.venvs/pypi-pub && source ~/.venvs/pypi-pub/bin/activate
  pip install --upgrade build twine
  cd packaging/mcpb
  make pypi-build VERSION=X.Y.Z
  python -m twine check pypi/dist/*
  python -m twine upload pypi/dist/*
  ```

- npm (`@stackql/mcp-server`): download the `npm-package` artifact from the workflow run (or build locally with `make npm-pack VERSION=X.Y.Z`), then publish with an `npm login` session (OTP prompted):

  ```
  npm publish stackql-mcp-server-X.Y.Z.tgz --access public
  ```

- Official MCP Registry (`io.github.stackql/stackql-mcp`): the workflow renders the `server-json` artifact for reference, but publication uses the `mcp-publisher` CLI from a workstation. Unlike the pypi/npm renderers, the server.json renderer reads the four `.sha256` files from the local `dist/` directory, so when the bundles were built in CI, download the published checksum files from the release first:

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
  - Log in with a classic PAT (scope `read:org` only, no repo scopes) via `MCP_GITHUB_TOKEN` as above, not the interactive device flow. The registry grants the `io.github.stackql/*` namespace only to `stackql` org Owners, and it checks the role via `GET /user/memberships/orgs`. The device-flow login is a GitHub App user token ("MCP Registry Login") that cannot see the org membership unless that app is installed on the org, so it 403s with a misleading hint about public org membership - the PAT path avoids all of that.
  - The registry JWT minted at login is short-lived - run `make registry-publish` immediately after `mcp-publisher login github`.
  - Old `mcp-publisher` versions (pre-1.2) drop `.mcpregistry_*` token files in the working directory (gitignored); current versions store the token in `~/.config/mcp-publisher/token.json`.

The local `make pypi-build` / `make npm-pack` / `make server-json` targets fetch the published `.sha256` files from the release, so they must run after step 6 completes. The render step stamps the version into `packaging/mcpb/pypi/pyproject.toml` in place - commit or revert the stamp afterwards.

8. Push the same release version tag to [stackql/releases.stackql.io](https://github.com/stackql/releases.stackql.io)
