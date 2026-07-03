#!/usr/bin/env bash
#
# render-npm-manifest.sh - render npm/platforms.json (bundle URLs + sha256
# pins) and stamp the version into npm/package.json.
#
# Fetches the canonical .sha256 files from the published GitHub release, so it
# must run AFTER the .mcpb assets for the version are published - the same
# rule as render-server-json.sh. Locally built bundles have different bytes
# than the CI-published ones; never pin local hashes.
#
# Usage:
#   scripts/render-npm-manifest.sh --version 0.10.500
#   VERSION=0.10.500 scripts/render-npm-manifest.sh
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
NPM_DIR="${NPM_DIR:-$ROOT_DIR/npm}"
# Canonical source for the .sha256 pins - always the GitHub release, the source
# of truth. Overridable for testing but normally left alone.
RELEASE_BASE="${RELEASE_BASE:-https://github.com/stackql/stackql/releases/download}"
# Front door the wrapper downloads the .mcpb bytes from at runtime. Proxies to
# the same release assets, so the sha256 pins (fetched from RELEASE_BASE) still
# hold. This is the baseUrl written into platforms.json.
DOWNLOAD_BASE="${DOWNLOAD_BASE:-https://releases.stackql.io/stackql}"

VERSION="${VERSION:-}"
while [ $# -gt 0 ]; do
  case "$1" in
    --version)   VERSION="$2"; shift 2 ;;
    --version=*) VERSION="${1#*=}"; shift ;;
    -h|--help)   sed -n '2,14p' "$0"; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done
[ -n "$VERSION" ] || { echo "error: --version required (or VERSION=X.Y.Z)" >&2; exit 2; }

sha_base="$RELEASE_BASE/v$VERSION"   # canonical GitHub release (v-prefixed) - for the pins
base_url="$DOWNLOAD_BASE/$VERSION"   # proxy front door (no v prefix) - written to platforms.json

fetch_sha() {
  # args: target-label -> prints the hex digest from the published .sha256
  local target="$1" line
  line="$(curl -fsSL "$sha_base/stackql-mcp-$target.mcpb.sha256")" || {
    echo "error: could not fetch sha256 for $target - are the v$VERSION .mcpb assets published?" >&2
    exit 1
  }
  echo "$line" | awk '{print $1; exit}'
}

SHA_LINUX_X64="$(fetch_sha linux-x64)"
SHA_LINUX_ARM64="$(fetch_sha linux-arm64)"
SHA_WINDOWS_X64="$(fetch_sha windows-x64)"
SHA_DARWIN_UNIVERSAL="$(fetch_sha darwin-universal)"

cat > "$NPM_DIR/platforms.json" <<EOF
{
  "version": "$VERSION",
  "baseUrl": "$base_url",
  "platforms": {
    "linux-x64": { "bundle": "stackql-mcp-linux-x64.mcpb", "sha256": "$SHA_LINUX_X64" },
    "linux-arm64": { "bundle": "stackql-mcp-linux-arm64.mcpb", "sha256": "$SHA_LINUX_ARM64" },
    "windows-x64": { "bundle": "stackql-mcp-windows-x64.mcpb", "sha256": "$SHA_WINDOWS_X64" },
    "darwin-universal": { "bundle": "stackql-mcp-darwin-universal.mcpb", "sha256": "$SHA_DARWIN_UNIVERSAL" }
  }
}
EOF

# stamp the package version to match (cygpath: MSYS paths confuse Windows node)
pkg_json="$NPM_DIR/package.json"
if command -v cygpath >/dev/null 2>&1; then
  pkg_json="$(cygpath -m "$pkg_json")"
fi
PKG_JSON="$pkg_json" NEW_VERSION="$VERSION" node -e "
const fs = require('fs');
const p = process.env.PKG_JSON;
const pkg = JSON.parse(fs.readFileSync(p, 'utf8'));
pkg.version = process.env.NEW_VERSION;
fs.writeFileSync(p, JSON.stringify(pkg, null, 2) + '\n');
"

echo "wrote $NPM_DIR/platforms.json (version $VERSION)"
echo "  linux-x64        $SHA_LINUX_X64"
echo "  linux-arm64      $SHA_LINUX_ARM64"
echo "  windows-x64      $SHA_WINDOWS_X64"
echo "  darwin-universal $SHA_DARWIN_UNIVERSAL"
