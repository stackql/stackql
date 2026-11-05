#!/usr/bin/env bash
#
# render-platforms.sh - the one renderer for every wrapper/SDK/plugin vector.
#
# Fetches the five canonical .mcpb.sha256 files from the published GitHub
# release, writes platforms.json (bundle names + sha256 pins + proxy baseUrl)
# into each vector directory, and stamps the version into each vector's
# manifest file. platforms.json is the ONLY pin source for every vector; the
# per-language pin tables (pins.rs, pins.go, the .NET pins.json placeholders)
# were deleted in favour of it.
#
# Must run AFTER the .mcpb assets for the version are published (same rule as
# render-server-json.sh). Locally built bundles have different bytes than the
# CI-published ones; never pin local hashes.
#
# Vectors and what gets written:
#   npm     npm/platforms.json                                 npm/package.json "version"
#   pypi    pypi/src/stackql_mcp_server/platforms.json         pypi/pyproject.toml version
#   cargo   cargo/platforms.json                               cargo/Cargo.toml [package] version
#   go      go/embed/platforms.json                            (module version is the git tag)
#   dotnet  dotnet/src/StackQL.Mcp/platforms.json              dotnet/Directory.Build.props <Version>
#   openai-plugin  ../openai-plugin/plugins/stackql: plugin payload (below) + plugin.json
#                  "version"; .codex-plugin/plugin.json is DERIVED from plugin.json
#   claude-plugin  ../claude-plugin: plugin payload (below) + .claude-plugin/plugin.json "version"
#
# The plugin payload is identical for both plugin vectors and is committed
# (the directories read the plugin from git): server/bin/stackql-mcp.js (the
# npm launcher, verbatim - it is dependency-free for this reason),
# server/platforms.json, LICENSE, the icon and skills/stackql-setup/SKILL.md.
# The validators assert byte-equality against these sources.
#
# Usage:
#   scripts/render-platforms.sh --version 0.10.601            # all vectors
#   scripts/render-platforms.sh --version 0.10.601 --vector cargo
#   VERSION=0.10.601 scripts/render-platforms.sh --vector npm
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# Per-vector dir overrides (kept for the npm/pypi renderers' existing contract).
NPM_DIR="${NPM_DIR:-$ROOT_DIR/npm}"
PYPI_DIR="${PYPI_DIR:-$ROOT_DIR/pypi}"
CARGO_DIR="${CARGO_DIR:-$ROOT_DIR/cargo}"
GO_DIR="${GO_DIR:-$ROOT_DIR/go}"
DOTNET_DIR="${DOTNET_DIR:-$ROOT_DIR/dotnet}"
PLUGIN_DIR="${PLUGIN_DIR:-$ROOT_DIR/../openai-plugin/plugins/stackql}"
CLAUDE_PLUGIN_DIR="${CLAUDE_PLUGIN_DIR:-$ROOT_DIR/../claude-plugin}"
# Canonical source for the .sha256 pins - always the GitHub release, the source
# of truth. Overridable for testing but normally left alone.
RELEASE_BASE="${RELEASE_BASE:-https://github.com/stackql/stackql/releases/download}"
# Front door the wrappers download the .mcpb bytes from at runtime. Proxies to
# the same release assets, so the sha256 pins (fetched from RELEASE_BASE) still
# hold. This is the baseUrl written into platforms.json.
DOWNLOAD_BASE="${DOWNLOAD_BASE:-https://releases.stackql.io/stackql}"
# PR validation only; publishing always uses the release checksums.
CHECKSUM_DIR="${CHECKSUM_DIR:-}"

ALL_VECTORS="npm pypi cargo go dotnet openai-plugin claude-plugin"

VERSION="${VERSION:-}"
VECTOR="all"
while [ $# -gt 0 ]; do
  case "$1" in
    --version)   VERSION="$2"; shift 2 ;;
    --version=*) VERSION="${1#*=}"; shift ;;
    --vector)    VECTOR="$2"; shift 2 ;;
    --vector=*)  VECTOR="${1#*=}"; shift ;;
    -h|--help)   sed -n '2,36p' "$0"; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done
[ -n "$VERSION" ] || { echo "error: --version required (or VERSION=X.Y.Z)" >&2; exit 2; }
case " $ALL_VECTORS all " in
  *" $VECTOR "*) ;;
  *) echo "error: unknown --vector '$VECTOR' (one of: $ALL_VECTORS all)" >&2; exit 2 ;;
esac

sha_base="$RELEASE_BASE/v$VERSION"   # canonical GitHub release (v-prefixed) - for the pins
base_url="$DOWNLOAD_BASE/$VERSION"   # proxy front door (no v prefix) - written to platforms.json

fetch_sha() {
  # args: target-label -> prints the hex digest from the published .sha256
  local target="$1" line digest
  if [ -n "$CHECKSUM_DIR" ]; then
    line="$(cat "$CHECKSUM_DIR/stackql-mcp-$target.mcpb.sha256")" || return 1
  else
    line="$(curl -fsSL --retry 5 --retry-delay 2 --retry-all-errors "$sha_base/stackql-mcp-$target.mcpb.sha256")" || {
      echo "error: could not fetch sha256 for $target - are the v$VERSION .mcpb assets published?" >&2
      return 1
    }
  fi
  digest="$(echo "$line" | awk '{print $1; exit}')"
  [[ "$digest" =~ ^[[:xdigit:]]{64}$ ]] || { echo "error: invalid sha256 for $target" >&2; return 1; }
  echo "$digest"
}

SHA_LINUX_X64="$(fetch_sha linux-x64)"
SHA_LINUX_ARM64="$(fetch_sha linux-arm64)"
SHA_WINDOWS_X64="$(fetch_sha windows-x64)"
SHA_WINDOWS_ARM64="$(fetch_sha windows-arm64)"
SHA_DARWIN_UNIVERSAL="$(fetch_sha darwin-universal)"

# The one platforms.json shape, byte-identical across vectors.
write_platforms_json() {
  local dest="$1"
  mkdir -p "$(dirname "$dest")"
  cat > "$dest" <<EOF
{
  "version": "$VERSION",
  "baseUrl": "$base_url",
  "platforms": {
    "linux-x64": { "bundle": "stackql-mcp-linux-x64.mcpb", "sha256": "$SHA_LINUX_X64" },
    "linux-arm64": { "bundle": "stackql-mcp-linux-arm64.mcpb", "sha256": "$SHA_LINUX_ARM64" },
    "windows-x64": { "bundle": "stackql-mcp-windows-x64.mcpb", "sha256": "$SHA_WINDOWS_X64" },
    "windows-arm64": { "bundle": "stackql-mcp-windows-arm64.mcpb", "sha256": "$SHA_WINDOWS_ARM64" },
    "darwin-universal": { "bundle": "stackql-mcp-darwin-universal.mcpb", "sha256": "$SHA_DARWIN_UNIVERSAL" }
  }
}
EOF
  echo "wrote $dest (version $VERSION)"
}

# sed -i without the GNU/BSD -i incompatibility.
sed_inplace() {
  local expr="$1" file="$2"
  sed -e "$expr" "$file" > "$file.tmp" && mv "$file.tmp" "$file"
}

# Windows-safe path for node (MSYS paths confuse Windows node).
node_path() {
  if command -v cygpath >/dev/null 2>&1; then cygpath -m "$1"; else echo "$1"; fi
}

# Stamp "version" in a JSON manifest, preserving the rest of the document.
stamp_json_version() {
  local file
  file="$(node_path "$1")"
  PKG_JSON="$file" NEW_VERSION="$VERSION" node -e "
const fs = require('fs');
const p = process.env.PKG_JSON;
const pkg = JSON.parse(fs.readFileSync(p, 'utf8'));
pkg.version = process.env.NEW_VERSION;
fs.writeFileSync(p, JSON.stringify(pkg, null, 2) + '\n');
"
}

# package-lock.json carries the version twice (top level and packages[""]);
# keep both equal to package.json so the lockfile never lags a release.
stamp_lockfile_version() {
  local file
  file="$(node_path "$1")"
  PKG_JSON="$file" NEW_VERSION="$VERSION" node -e "
const fs = require('fs');
const p = process.env.PKG_JSON;
const lock = JSON.parse(fs.readFileSync(p, 'utf8'));
lock.version = process.env.NEW_VERSION;
if (lock.packages && lock.packages['']) lock.packages[''].version = process.env.NEW_VERSION;
fs.writeFileSync(p, JSON.stringify(lock, null, 2) + '\n');
"
}

render_npm() {
  write_platforms_json "$NPM_DIR/platforms.json"
  stamp_json_version "$NPM_DIR/package.json"
  stamp_lockfile_version "$NPM_DIR/package-lock.json"
}

render_pypi() {
  write_platforms_json "$PYPI_DIR/src/stackql_mcp_server/platforms.json"
  # stamp the project version (first version line in [project])
  sed_inplace "s/^version = \".*\"/version = \"$VERSION\"/" "$PYPI_DIR/pyproject.toml"
}

render_cargo() {
  write_platforms_json "$CARGO_DIR/platforms.json"
  # only [package] has a top-level 'version = "..."' line (deps are inline tables)
  sed_inplace "s/^version = \".*\"/version = \"$VERSION\"/" "$CARGO_DIR/Cargo.toml"
}

render_go() {
  # go:embed cannot reach outside the package dir, so the manifest lives next
  # to the embed package. The module version is the mirror repo's git tag.
  write_platforms_json "$GO_DIR/embed/platforms.json"
}

render_dotnet() {
  write_platforms_json "$DOTNET_DIR/src/StackQL.Mcp/platforms.json"
  local props="$DOTNET_DIR/Directory.Build.props"
  grep -q "<Version>" "$props" || { echo "error: $props has no <Version> element to stamp" >&2; exit 1; }
  sed_inplace "s#<Version>[^<]*</Version>#<Version>$VERSION</Version>#" "$props"
}

# The shared plugin payload: the npm launcher verbatim in the npm package
# layout (server/bin/stackql-mcp.js next to server/platforms.json, which the
# launcher resolves as ../platforms.json), the licence, the icon and the
# onboarding skill. No package.json, lockfile or node_modules: the launcher
# needs only Node.js built-ins, and both directory validators hold a plugin
# for review when it ships a package install.
render_plugin_payload() {
  local dir="$1" icon_dest="$2"
  mkdir -p "$dir/server/bin" "$dir/skills/stackql-setup" "$(dirname "$dir/$icon_dest")"
  cp -p "$NPM_DIR/bin/stackql-mcp.js" "$dir/server/bin/stackql-mcp.js"
  write_platforms_json "$dir/server/platforms.json"
  cp -p "$ROOT_DIR/LICENSE" "$dir/LICENSE"
  cp -p "$ROOT_DIR/assets/icon.png" "$dir/$icon_dest"
  cp -p "$ROOT_DIR/skills/stackql-setup/SKILL.md" "$dir/skills/stackql-setup/SKILL.md"
}

render_openai_plugin() {
  render_plugin_payload "$PLUGIN_DIR" "assets/icon.png"
  stamp_json_version "$PLUGIN_DIR/plugin.json"
  # Derive the Codex compatibility overlay (.codex-plugin/plugin.json) from
  # the portable manifest so the two cannot drift: root fields carried over,
  # the Codex-only skills/mcpServers paths, requires_local_executor and
  # interface hoisted to the top level, and the remaining com.openai settings
  # (onboardingSkill, review, publication) kept under extensions.
  mkdir -p "$PLUGIN_DIR/.codex-plugin"
  SRC="$(node_path "$PLUGIN_DIR/plugin.json")" DST="$(node_path "$PLUGIN_DIR/.codex-plugin/plugin.json")" node -e "
const fs = require('fs');
const portable = JSON.parse(fs.readFileSync(process.env.SRC, 'utf8'));
const openai = (portable.extensions && portable.extensions['com.openai']) || {};
const { interface: iface, requires_local_executor, ...rest } = openai;
const overlay = {
  name: portable.name,
  version: portable.version,
  description: portable.description,
  author: portable.author,
  homepage: portable.homepage,
  repository: portable.repository,
  license: portable.license,
  keywords: portable.keywords,
  skills: './skills/',
  mcpServers: './.mcp.json',
};
if (requires_local_executor !== undefined) overlay.requires_local_executor = requires_local_executor;
overlay.interface = iface;
overlay.extensions = { 'com.openai': rest };
fs.writeFileSync(process.env.DST, JSON.stringify(overlay, null, 2) + '\n');
"
  echo "stamped $PLUGIN_DIR (version $VERSION)"
}

render_claude_plugin() {
  render_plugin_payload "$CLAUDE_PLUGIN_DIR" "icon.png"
  stamp_json_version "$CLAUDE_PLUGIN_DIR/.claude-plugin/plugin.json"
  echo "stamped $CLAUDE_PLUGIN_DIR (version $VERSION)"
}

if [ "$VECTOR" = "all" ]; then
  vectors="$ALL_VECTORS"
else
  vectors="$VECTOR"
fi
for v in $vectors; do
  "render_${v//-/_}"
done

echo "pins (v$VERSION):"
echo "  linux-x64        $SHA_LINUX_X64"
echo "  linux-arm64      $SHA_LINUX_ARM64"
echo "  windows-x64      $SHA_WINDOWS_X64"
echo "  windows-arm64    $SHA_WINDOWS_ARM64"
echo "  darwin-universal $SHA_DARWIN_UNIVERSAL"
