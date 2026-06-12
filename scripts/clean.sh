#!/usr/bin/env bash
#
# clean.sh - remove generated bundles from ./dist
#
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
rm -f "$ROOT_DIR"/dist/*.mcpb "$ROOT_DIR"/dist/*.sha256 2>/dev/null || true
echo "cleaned dist/"
