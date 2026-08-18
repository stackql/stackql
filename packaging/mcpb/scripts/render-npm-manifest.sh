#!/usr/bin/env bash
#
# render-npm-manifest.sh - render npm/platforms.json (bundle URLs + sha256
# pins) and stamp the version into npm/package.json.
#
# Thin wrapper over render-platforms.sh (the one renderer for every vector);
# kept for its existing CLI. Same ordering rule: run AFTER the .mcpb assets
# for the version are published - never pin locally built bundle hashes.
#
# Usage:
#   scripts/render-npm-manifest.sh --version 0.10.500
#   VERSION=0.10.500 scripts/render-npm-manifest.sh
set -euo pipefail
exec bash "$(dirname "${BASH_SOURCE[0]}")/render-platforms.sh" --vector npm "$@"
