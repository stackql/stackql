#!/usr/bin/env bash
#
# render-server-json.sh - render registry/server.json from the template by
# substituting __VERSION__ and per-platform SHA-256 tokens from dist/*.sha256.
#
# Reads dist/stackql-mcp-<target>.mcpb.sha256 for each of the four targets and
# writes registry/server.json. Fails hard if any sha file is missing.
#
# Usage:
#   scripts/render-server-json.sh --version 0.10.500
#   VERSION=0.10.500 scripts/render-server-json.sh
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${DIST_DIR:-$ROOT_DIR/dist}"
TEMPLATE="${TEMPLATE:-$ROOT_DIR/registry/server.template.json}"
OUT="${OUT:-$ROOT_DIR/registry/server.json}"

VERSION="${VERSION:-}"
while [ $# -gt 0 ]; do
  case "$1" in
    --version)   VERSION="$2"; shift 2 ;;
    --version=*) VERSION="${1#*=}"; shift ;;
    -h|--help)   sed -n '2,15p' "$0"; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done
[ -n "$VERSION" ] || { echo "error: --version required (or VERSION=X.Y.Z)" >&2; exit 2; }

read_sha() {
  # extract the hex digest from a coreutils-style "<hash>  <name>" file
  local f="$1"
  if [ ! -f "$f" ]; then
    echo "error: missing sha file: $f" >&2
    echo "  run 'make all VERSION=$VERSION' (and the Mac slice for darwin) first" >&2
    exit 1
  fi
  awk '{print $1; exit}' "$f"
}

SHA_LINUX_X64="$(read_sha "$DIST_DIR/stackql-mcp-linux-x64.mcpb.sha256")"
SHA_LINUX_ARM64="$(read_sha "$DIST_DIR/stackql-mcp-linux-arm64.mcpb.sha256")"
SHA_WINDOWS_X64="$(read_sha "$DIST_DIR/stackql-mcp-windows-x64.mcpb.sha256")"
SHA_DARWIN_UNIVERSAL="$(read_sha "$DIST_DIR/stackql-mcp-darwin-universal.mcpb.sha256")"

sed \
  -e "s|__VERSION__|${VERSION}|g" \
  -e "s|__SHA_LINUX_X64__|${SHA_LINUX_X64}|g" \
  -e "s|__SHA_LINUX_ARM64__|${SHA_LINUX_ARM64}|g" \
  -e "s|__SHA_WINDOWS_X64__|${SHA_WINDOWS_X64}|g" \
  -e "s|__SHA_DARWIN_UNIVERSAL__|${SHA_DARWIN_UNIVERSAL}|g" \
  "$TEMPLATE" > "$OUT"

echo "wrote $OUT"
echo "  version: $VERSION"
echo "  linux-x64        $SHA_LINUX_X64"
echo "  linux-arm64      $SHA_LINUX_ARM64"
echo "  windows-x64      $SHA_WINDOWS_X64"
echo "  darwin-universal $SHA_DARWIN_UNIVERSAL"
