#!/usr/bin/env bash
#
# publish-mirror.sh - publish a git-resolved SDK vector (go) by pushing
# the rendered subtree from packaging/mcpb/<vector> to its mirror repo and
# tagging v<version> there.
#
# Go modules cannot be consumed cleanly from a subdirectory of a monorepo
# (it would need packaging/mcpb/go/vX.Y.Z tags and an ugly import path), so
# the source of truth stays here and the mirror is a publish artefact:
#   go    -> github.com/stackql/stackql-mcp-go    (module path unchanged)
#
# Idempotent: if the mirror already has tag v<version> pointing at an
# identical tree the push is skipped; if it points at a different tree the
# script fails (published versions are immutable - cut a new patch release).
#
# Auth: SDK_MIRROR_TOKEN (fine-grained PAT, contents:write on the mirror
# repos) or, locally, whatever 'git' is already authorised with for GitHub.
#
# Usage:
#   scripts/publish-mirror.sh --vector go --version 0.10.601
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VECTOR="" VERSION="${VERSION:-}"
while [ $# -gt 0 ]; do
  case "$1" in
    --vector)    VECTOR="$2"; shift 2 ;;
    --vector=*)  VECTOR="${1#*=}"; shift ;;
    --version)   VERSION="$2"; shift 2 ;;
    --version=*) VERSION="${1#*=}"; shift ;;
    -h|--help)   sed -n '2,24p' "$0"; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done
[ -n "$VERSION" ] || { echo "error: --version required" >&2; exit 2; }

case "$VECTOR" in
  go)
    MIRROR_REPO="stackql/stackql-mcp-go"
    RENDERED=(embed/platforms.json)
    ;;
  *) echo "error: --vector must be go" >&2; exit 2 ;;
esac
SRC_DIR="$ROOT_DIR/$VECTOR"
TAG="v$VERSION"

for f in "${RENDERED[@]}"; do
  [ -f "$SRC_DIR/$f" ] || { echo "error: $SRC_DIR/$f missing - run 'make $VECTOR-manifest VERSION=$VERSION' first" >&2; exit 1; }
done
grep -q "\"version\": \"$VERSION\"" "$SRC_DIR/${RENDERED[0]}" \
  || { echo "error: ${RENDERED[0]} is not rendered for $VERSION" >&2; exit 1; }

# git auth: token via extraheader (never in the URL) when provided.
GIT=(git)
if [ -n "${SDK_MIRROR_TOKEN:-}" ]; then
  auth="$(printf 'x-access-token:%s' "$SDK_MIRROR_TOKEN" | base64 | tr -d '\n')"
  GIT=(git -c "http.https://github.com/.extraheader=AUTHORIZATION: basic $auth")
fi
MIRROR_URL="https://github.com/$MIRROR_REPO.git"

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
echo "cloning $MIRROR_REPO"
"${GIT[@]}" clone --quiet "$MIRROR_URL" "$work/mirror"
cd "$work/mirror"
default_branch="$(git rev-parse --abbrev-ref HEAD)"
# Identity for the sync commit and the annotated tag: a fresh clone on a CI
# runner has no global identity, and 'git tag -a' needs one just like
# 'git commit' (per-command -c flags would not cover the tag).
git config user.name "stackql-release"
git config user.email "info@stackql.io"

# Replace the working tree with the tracked files of the vector plus the
# rendered manifests (which the vector .gitignore excludes, hence add -f).
find . -mindepth 1 -maxdepth 1 ! -name .git -exec rm -rf {} +
(cd "$SRC_DIR" && git ls-files -z) | (cd "$SRC_DIR" && xargs -0 -I{} sh -c 'mkdir -p "$0/$(dirname "{}")" && cp -p "{}" "$0/{}"' "$work/mirror")
for f in "${RENDERED[@]}"; do
  mkdir -p "$(dirname "$f")"; cp -p "$SRC_DIR/$f" "$f"
done
git add -A
git add -f "${RENDERED[@]}"

src_sha="$(git -C "$ROOT_DIR" rev-parse --short HEAD 2>/dev/null || echo unknown)"
if git diff --cached --quiet; then
  echo "mirror tree already matches packaging/mcpb/$VECTOR"
else
  git commit --quiet -m "sync packaging/mcpb/$VECTOR from stackql/stackql@$src_sha for $TAG"
fi
new_tree="$(git rev-parse HEAD^{tree})"

if git ls-remote --exit-code --tags origin "refs/tags/$TAG" >/dev/null 2>&1; then
  git fetch --quiet origin "refs/tags/$TAG:refs/tags/$TAG"
  if [ "$(git rev-parse "$TAG^{tree}")" = "$new_tree" ]; then
    echo "$MIRROR_REPO already has $TAG with an identical tree - nothing to publish"
    exit 0
  fi
  echo "error: $MIRROR_REPO already has $TAG with a different tree; published versions are immutable" >&2
  exit 1
fi

git tag -a "$TAG" -m "stackql $TAG"
"${GIT[@]}" push --quiet origin "HEAD:$default_branch" "refs/tags/$TAG"
echo "published $MIRROR_REPO@$TAG (branch $default_branch)"
case "$VECTOR" in
  go)    echo "  go get github.com/stackql/stackql-mcp-go@$TAG" ;;
esac
