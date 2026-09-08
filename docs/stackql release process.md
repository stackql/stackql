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
git tag v0.11.669
git push origin v0.11.669
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

The `mcp-packaging` workflow does the following:

- attaches the `.mcpb` bundles (and `.sha256` files) to the release
- pushes the multi-arch OCI image to Docker Hub
- publishes packages to crates.io (rust), nuget (dotnet), and the go mirror

requires secrets: `CARGO_REGISTRY_TOKEN`, `NUGET_API_KEY`, `SDK_MIRROR_TOKEN`  

manual fallback:

```bash
cd packaging/mcpb
make cargo-publish  VERSION=X.Y.Z   # CARGO_REGISTRY_TOKEN
make go-publish     VERSION=X.Y.Z   # SDK_MIRROR_TOKEN, or an already-authorised git
make dotnet-publish VERSION=X.Y.Z   # NUGET_API_KEY
```

7. Make the release available via **releases.stackql.io**

Push a tag to [releases.stackql.io](https://github.com/stackql/releases.stackql.io), eg:

> note: the tag needs to match the tag for the release

```
git tag v0.11.669 && git push origin v0.11.669
```

8. Publish the MCP wrapper packages (manual last mile)

8a. npm (`@stackql/mcp-server`)

Requires an `npm login` session as a user with publish rights on the `@stackql` scope; the publish prompts for an OTP.

```bash
cd packaging/mcpb
make npm-pack VERSION=X.Y.Z
cd npm
npm publish stackql-mcp-server-X.Y.Z.tgz --access public
```

8b. PyPI (`stackql-mcp-server`)

Requires a PyPI API token with upload rights on the project; twine username is `__token__`, password is the token. On Debian/Ubuntu (including WSL) `pip install` outside a venv is blocked by PEP 668, hence the venv.

```bash
cd packaging/mcpb
python3 -m venv ~/.venvs/pypi-pub && source ~/.venvs/pypi-pub/bin/activate
pip install --upgrade build twine
make pypi-build VERSION=X.Y.Z
python -m twine check pypi/dist/*
python -m twine upload pypi/dist/*
```

9. Official MCP Registry (`io.github.stackql/stackql-mcp`) - dispatch last

Manual fallback (if the workflow fails or an out-of-band publish is needed):

Requires the latest `mcp-publisher` CLI and a classic GitHub PAT (scope `read:org` only, no repo scopes) created by a `stackql` org Owner at https://github.com/settings/tokens/new. The server.json renderer reads the four `.sha256` files from the local `dist/` directory, so download the published checksum files from the release first.

Install/upgrade the CLI (linux amd64 shown; assets exist per platform):

```bash
curl -fsSL "https://github.com/modelcontextprotocol/registry/releases/latest/download/mcp-publisher_linux_amd64.tar.gz" | tar xz mcp-publisher
sudo install -m 0755 mcp-publisher /usr/local/bin/mcp-publisher
```

Publish:

```bash
cd packaging/mcpb
for t in linux-x64 linux-arm64 windows-x64 darwin-universal; do
  curl -fsSL -o dist/stackql-mcp-$t.mcpb.sha256 \
    "https://github.com/stackql/stackql/releases/download/vX.Y.Z/stackql-mcp-$t.mcpb.sha256"
done
export MCP_GITHUB_TOKEN=<classic PAT with read:org scope>
mcp-publisher login github
make registry-publish VERSION=X.Y.Z
```

10. Update the ChatGPT/Codex stdio plugin

After 7a confirms `@stackql/mcp-server@X.Y.Z` is live, update the local plugin in a follow-up PR:

a. Set `version` in `packaging/openai-plugin/plugins/stackql/.codex-plugin/plugin.json` to `X.Y.Z`.
b. Set the pinned `@stackql/mcp-server@X.Y.Z` in `packaging/openai-plugin/plugins/stackql/bin/stackql-mcp.js`.
c. Run:

```bash
npm view @stackql/mcp-server@X.Y.Z version
python3 packaging/openai-plugin/scripts/validate.py
python3 packaging/openai-plugin/scripts/smoke-test.py
```

The marketplace entry is version-independent. This step does not change the MCPB, Anthropic, PyPI, OCI, or MCP Registry artifacts. See `packaging/mcpb/README.md` for the detailed addendum.

## Assurance

To ensure the mcp server for latest release is published across all surfaces run:

```bash
# from within a python venv created previously
# in the `packaging/mcpb` directory, run...
make check-published
```