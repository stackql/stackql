#!/usr/bin/env bash
# Attach the ChatGPT plugin ZIP and a SHA-256 checksum to a GitHub release.
# Email blocks archives that contain .js files, so the ZIP travels as a release asset.
set -euo pipefail

zip="${1:?usage: attach-release-asset.sh <plugin-zip> <release-tag>}"
tag="${2:?usage: attach-release-asset.sh <plugin-zip> <release-tag>}"

if command -v sha256sum >/dev/null 2>&1; then
  sha256sum "${zip}" | awk '{print $1}' > "${zip}.sha256"
else
  shasum -a 256 "${zip}" | awk '{print $1}' > "${zip}.sha256"
fi
gh release upload "${tag}" "${zip}" "${zip}.sha256" --clobber
