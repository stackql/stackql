#!/usr/bin/env bash
#
# render-pypi-manifest.sh - render pypi/src/stackql_mcp_server/platforms.json
# (bundle URLs + sha256 pins) and stamp the version into pypi/pyproject.toml.
#
# Fetches the canonical .sha256 files from the published GitHub release, so it
# must run AFTER the .mcpb assets for the version are published - the same
# rule as render-server-json.sh and render-npm-manifest.sh.
#
# Usage:
#   scripts/render-pypi-manifest.sh --version 0.10.500
#   VERSION=0.10.500 scripts/render-pypi-manifest.sh
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PYPI_DIR="${PYPI_DIR:-$ROOT_DIR/pypi}"
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
    -h|--help)   sed -n '2,13p' "$0"; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done
[ -n "$VERSION" ] || { echo "error: --version required (or VERSION=X.Y.Z)" >&2; exit 2; }

sha_base="$RELEASE_BASE/v$VERSION"   # canonical GitHub release - for the pins
base_url="$DOWNLOAD_BASE/v$VERSION"  # proxy front door - written to platforms.json

fetch_sha() {
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

cat > "$PYPI_DIR/src/stackql_mcp_server/platforms.json" <<EOF
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

# stamp the project version (first version line in [project])
sed -i.bak "s/^version = \".*\"/version = \"$VERSION\"/" "$PYPI_DIR/pyproject.toml"
rm -f "$PYPI_DIR/pyproject.toml.bak"

echo "wrote $PYPI_DIR/src/stackql_mcp_server/platforms.json (version $VERSION)"
echo "  linux-x64        $SHA_LINUX_X64"
echo "  linux-arm64      $SHA_LINUX_ARM64"
echo "  windows-x64      $SHA_WINDOWS_X64"
echo "  darwin-universal $SHA_DARWIN_UNIVERSAL"
