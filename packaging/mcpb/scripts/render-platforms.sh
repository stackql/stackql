#!/usr/bin/env bash
#
# render-platforms.sh - the one renderer for every wrapper/SDK vector.
#
# Fetches the four canonical .mcpb.sha256 files from the published GitHub
# release, writes platforms.json (bundle names + sha256 pins + proxy baseUrl)
# into each vector directory, and stamps the version into each vector's
# manifest file. platforms.json is the ONLY pin source for every vector; the
# per-language pin tables (pins.rs, pins.go, Pins.kt, Pins.swift, the Gleam
# const, the .NET pins.json placeholders) were deleted in favour of it.
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
#   gleam   gleam/platforms.json + src/stackql_mcp/pins.gleam  gleam/gleam.toml version
#   kotlin  kotlin/stackql-mcp/src/main/resources/platforms.json  kotlin/gradle.properties version
#   swift   swift/Sources/StackQLMCP/Resources/platforms.json  swift/Sources/StackQLMCP/Version.swift
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
GLEAM_DIR="${GLEAM_DIR:-$ROOT_DIR/gleam}"
KOTLIN_DIR="${KOTLIN_DIR:-$ROOT_DIR/kotlin}"
SWIFT_DIR="${SWIFT_DIR:-$ROOT_DIR/swift}"
# Canonical source for the .sha256 pins - always the GitHub release, the source
# of truth. Overridable for testing but normally left alone.
RELEASE_BASE="${RELEASE_BASE:-https://github.com/stackql/stackql/releases/download}"
# Front door the wrappers download the .mcpb bytes from at runtime. Proxies to
# the same release assets, so the sha256 pins (fetched from RELEASE_BASE) still
# hold. This is the baseUrl written into platforms.json.
DOWNLOAD_BASE="${DOWNLOAD_BASE:-https://releases.stackql.io/stackql}"

ALL_VECTORS="npm pypi cargo go dotnet gleam kotlin swift"

VERSION="${VERSION:-}"
VECTOR="all"
while [ $# -gt 0 ]; do
  case "$1" in
    --version)   VERSION="$2"; shift 2 ;;
    --version=*) VERSION="${1#*=}"; shift ;;
    --vector)    VECTOR="$2"; shift 2 ;;
    --vector=*)  VECTOR="${1#*=}"; shift ;;
    -h|--help)   sed -n '2,29p' "$0"; exit 0 ;;
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

render_npm() {
  write_platforms_json "$NPM_DIR/platforms.json"
  # stamp the package version to match (cygpath: MSYS paths confuse Windows node)
  local pkg_json="$NPM_DIR/package.json"
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

render_gleam() {
  write_platforms_json "$GLEAM_DIR/platforms.json"
  sed_inplace "s/^version = \".*\"/version = \"$VERSION\"/" "$GLEAM_DIR/gleam.toml"
  # No build-time file reads on the BEAM: the renderer stamps a module that
  # mirrors platforms.json field for field.
  local pins="$GLEAM_DIR/src/stackql_mcp/pins.gleam"
  cat > "$pins" <<EOF
//// GENERATED by packaging/mcpb/scripts/render-platforms.sh - do not edit.
//// Mirrors platforms.json (the shared pin manifest) field for field.

/// The stackql release this library is version-locked to.
pub const version = "$VERSION"

/// Front door the .mcpb bundle is downloaded from (attribution proxy).
pub const base_url = "$base_url"

/// (bundle name, sha256) for a platform key, or Error for an unknown key.
pub fn pin(platform_key: String) -> Result(#(String, String), Nil) {
  case platform_key {
    "linux-x64" ->
      Ok(#(
        "stackql-mcp-linux-x64.mcpb",
        "$SHA_LINUX_X64",
      ))
    "linux-arm64" ->
      Ok(#(
        "stackql-mcp-linux-arm64.mcpb",
        "$SHA_LINUX_ARM64",
      ))
    "windows-x64" ->
      Ok(#(
        "stackql-mcp-windows-x64.mcpb",
        "$SHA_WINDOWS_X64",
      ))
    "darwin-universal" ->
      Ok(#(
        "stackql-mcp-darwin-universal.mcpb",
        "$SHA_DARWIN_UNIVERSAL",
      ))
    _ -> Error(Nil)
  }
}
EOF
  echo "wrote $pins"
}

render_kotlin() {
  write_platforms_json "$KOTLIN_DIR/stackql-mcp/src/main/resources/platforms.json"
  local props="$KOTLIN_DIR/gradle.properties"
  grep -q "^version=" "$props" || { echo "error: $props has no 'version=' line to stamp" >&2; exit 1; }
  sed_inplace "s/^version=.*/version=$VERSION/" "$props"
}

render_swift() {
  write_platforms_json "$SWIFT_DIR/Sources/StackQLMCP/Resources/platforms.json"
  local vfile="$SWIFT_DIR/Sources/StackQLMCP/Version.swift"
  cat > "$vfile" <<EOF
// GENERATED by packaging/mcpb/scripts/render-platforms.sh - do not edit.

/// The stackql release this package is version-locked to (also the package's
/// own version: the mirror repo is tagged v<version>).
public enum StackQLMCPVersion {
    public static let current = "$VERSION"
}
EOF
  echo "wrote $vfile"
}

if [ "$VECTOR" = "all" ]; then
  vectors="$ALL_VECTORS"
else
  vectors="$VECTOR"
fi
for v in $vectors; do
  "render_$v"
done

echo "pins (v$VERSION):"
echo "  linux-x64        $SHA_LINUX_X64"
echo "  linux-arm64      $SHA_LINUX_ARM64"
echo "  windows-x64      $SHA_WINDOWS_X64"
echo "  darwin-universal $SHA_DARWIN_UNIVERSAL"
