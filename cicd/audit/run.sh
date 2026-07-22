#!/usr/bin/env bash
# Docker audit quickstart bootstrap (this repo's only audit logic).
# Downloads the pinned audit engine — run.sh + queries + scripts — from
# stackql/stackql-audit-action (cicd/audit/quickstart/) and hands off to its
# run.sh. All run logic lives in that engine; here we only pin the version.
# The stackql server + provider pins live in docker-compose.audit.yaml.
set -euo pipefail

python3 /audit/fetch.py

[ -f /tmp/engine/run.sh ] || { echo "FATAL: engine download failed (no run.sh in ${AUDIT_ENGINE_SUBDIR:-cicd/audit/quickstart})"; exit 1; }
exec bash /tmp/engine/run.sh
