#!/usr/bin/env bash
#
# package.sh - build (optionally signed) .mcpb bundles for the StackQL MCP server.
#
# Reads the platform binaries (and the notarised macOS .pkg) you drop into ./bin,
# packs one .mcpb per target into ./dist, optionally signs each bundle, and writes
# a matching .sha256 next to it. You then attach the ./dist artefacts to the same
# GitHub release as the corresponding stackql build.
#
# The server packed into each bundle is the stackql binary itself, launched as
# `stackql mcp` (see manifest/manifest.template.json). The stackql_mcp_client
# binary is a test client and is NOT packaged.
#
# Usage:
#   scripts/package.sh --version 1.2.3
#   VERSION=1.2.3 scripts/package.sh
#
# Optional bundle signing (separate from OS code signing of the binaries):
#   self-signed (testing):
#     MCPB_SELF_SIGN=true scripts/package.sh --version 1.2.3
#   production cert:
#     MCPB_SIGN_CERT=cert.pem MCPB_SIGN_KEY=key.pem \
#     [MCPB_SIGN_INTERMEDIATES="intermediate-ca.pem root-ca.pem"] \
#     scripts/package.sh --version 1.2.3
#
set -euo pipefail

# --- locations -------------------------------------------------------------
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_DIR="${BIN_DIR:-$ROOT_DIR/bin}"
DIST_DIR="${DIST_DIR:-$ROOT_DIR/dist}"
TEMPLATE="${TEMPLATE:-$ROOT_DIR/manifest/manifest.template.json}"

# --- args ------------------------------------------------------------------
VERSION="${VERSION:-}"
while [ $# -gt 0 ]; do
  case "$1" in
    --version)   VERSION="$2"; shift 2 ;;
    --version=*) VERSION="${1#*=}"; shift ;;
    -h|--help)   sed -n '2,30p' "$0"; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done
[ -n "$VERSION" ] || { echo "error: version required (--version X.Y.Z or VERSION=X.Y.Z)" >&2; exit 2; }

mkdir -p "$DIST_DIR"

# --- mcpb cli wrapper (prefer installed mcpb, else npx) --------------------
if command -v mcpb >/dev/null 2>&1; then
  mcpb() { command mcpb "$@"; }
else
  mcpb() { npx --yes @anthropic-ai/mcpb "$@"; }
fi

# --- helpers ---------------------------------------------------------------
sha_file() {
  # Write "<hash>  <basename>" so the checksum matches the released filename.
  local f="$1" dir base
  dir="$(dirname "$f")"; base="$(basename "$f")"
  ( cd "$dir"
    if command -v sha256sum >/dev/null 2>&1; then
      sha256sum "$base" > "${base}.sha256"
    else
      shasum -a 256 "$base" > "${base}.sha256"
    fi
    cat "${base}.sha256"
  )
}

sign_bundle() {
  local f="$1"
  if [ "${MCPB_SELF_SIGN:-false}" = "true" ]; then
    echo "  signing bundle (self-signed): $(basename "$f")"
    mcpb sign "$f" --self-signed
    mcpb verify "$f" || true
  elif [ -n "${MCPB_SIGN_CERT:-}" ] && [ -n "${MCPB_SIGN_KEY:-}" ]; then
    echo "  signing bundle (production cert): $(basename "$f")"
    if [ -n "${MCPB_SIGN_INTERMEDIATES:-}" ]; then
      # shellcheck disable=SC2086
      mcpb sign "$f" --cert "$MCPB_SIGN_CERT" --key "$MCPB_SIGN_KEY" --intermediate $MCPB_SIGN_INTERMEDIATES
    else
      mcpb sign "$f" --cert "$MCPB_SIGN_CERT" --key "$MCPB_SIGN_KEY"
    fi
    mcpb verify "$f"
  else
    echo "  bundle signing skipped (set MCPB_SELF_SIGN=true or MCPB_SIGN_CERT + MCPB_SIGN_KEY)"
  fi
}

pack_bundle() {
  # args: label  binary-source-path  binary-name-in-bundle
  local label="$1" src="$2" binname="$3"
  local stage out
  stage="$(mktemp -d)"
  mkdir -p "$stage/server"
  cp "$src" "$stage/server/$binname"
  chmod +x "$stage/server/$binname" 2>/dev/null || true
  sed -e "s/__BINARY_NAME__/${binname}/g" -e "s/__VERSION__/${VERSION}/g" \
      "$TEMPLATE" > "$stage/manifest.json"
  out="$DIST_DIR/stackql-mcp-${label}.mcpb"
  echo "==> $label"
  mcpb validate "$stage/manifest.json"
  mcpb pack "$stage" "$out"
  sign_bundle "$out"
  sha_file "$out"
  rm -rf "$stage"
}

extract_pkg_binary() {
  # args: pkg-path  binary-name  dest-path
  # Extracts the (already signed + notarised) binary from a flat .pkg. The
  # binary's embedded code signature is preserved; its notarisation is still
  # recognised online by cdhash. The stapled ticket stays on the .pkg.
  local pkg="$1" binname="$2" dest="$3"
  if ! command -v pkgutil >/dev/null 2>&1; then
    echo "  error: pkgutil not found - run the darwin target on macOS to extract from the .pkg" >&2
    return 1
  fi
  local tmp found
  tmp="$(mktemp -d)"
  pkgutil --expand-full "$pkg" "$tmp/expanded" >/dev/null
  found="$(find "$tmp/expanded" -type f -name "$binname" -path '*Payload*' 2>/dev/null | head -n1)"
  [ -n "$found" ] || found="$(find "$tmp/expanded" -type f -name "$binname" 2>/dev/null | head -n1)"
  if [ -z "$found" ]; then
    echo "  error: '$binname' not found inside $(basename "$pkg")" >&2
    rm -rf "$tmp"; return 1
  fi
  cp "$found" "$dest"
  rm -rf "$tmp"
}

have() { [ -e "$1" ]; }

# --- run -------------------------------------------------------------------
echo "StackQL MCPB packaging - version $VERSION"
echo "source: $BIN_DIR"
echo "output: $DIST_DIR"
echo

built=0

if have "$BIN_DIR/linux-amd64/stackql"; then
  pack_bundle "linux-x64" "$BIN_DIR/linux-amd64/stackql" "stackql"; built=$((built+1))
else
  echo "skip linux-x64       (missing bin/linux-amd64/stackql)"
fi

if have "$BIN_DIR/linux-arm64/stackql"; then
  pack_bundle "linux-arm64" "$BIN_DIR/linux-arm64/stackql" "stackql"; built=$((built+1))
else
  echo "skip linux-arm64     (missing bin/linux-arm64/stackql)"
fi

if have "$BIN_DIR/windows-amd64/stackql.exe"; then
  pack_bundle "windows-x64" "$BIN_DIR/windows-amd64/stackql.exe" "stackql.exe"; built=$((built+1))
else
  echo "skip windows-x64     (missing bin/windows-amd64/stackql.exe)"
fi

darwin_pkg="$(ls "$BIN_DIR"/darwin/*.pkg 2>/dev/null | head -n1 || true)"
if [ -n "$darwin_pkg" ]; then
  echo "==> darwin-universal (extracting from $(basename "$darwin_pkg"))"
  tmpdir="$(mktemp -d)"; tmpbin="$tmpdir/stackql"
  if extract_pkg_binary "$darwin_pkg" "stackql" "$tmpbin"; then
    pack_bundle "darwin-universal" "$tmpbin" "stackql"; built=$((built+1))
  fi
  rm -rf "$tmpdir"
else
  echo "skip darwin-universal (no .pkg in bin/darwin/)"
fi

echo
echo "done: $built bundle(s)"
ls -1 "$DIST_DIR"/*.mcpb 2>/dev/null || true
