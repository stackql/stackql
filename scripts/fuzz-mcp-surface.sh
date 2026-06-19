#!/usr/bin/env bash
# Start the MCP fuzz fixture and run mcp-fuzzer against the streamable HTTP surface.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

FUZZ_IMAGE="${MCP_FUZZER_IMAGE:-princekrroshan01/mcp-fuzzer:v0.4.0}"
FUZZ_RUNS="${MCP_FUZZ_RUNS:-3}"
FUZZ_TIMEOUT="${MCP_FUZZ_TIMEOUT:-30}"
FUZZ_PORT="${MCP_FUZZ_PORT:-19992}"
OUTPUT_DIR="${MCP_FUZZ_OUTPUT:-$ROOT/fuzz-output}"
SERVER_LOG="${TMPDIR:-/tmp}/stackql-fuzz-server-$$.log"

mkdir -p "$OUTPUT_DIR"

cleanup() {
  if [[ -n "${SERVER_PID:-}" ]] && kill -0 "$SERVER_PID" 2>/dev/null; then
    kill "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi
  rm -f "$SERVER_LOG"
}
trap cleanup EXIT

echo "starting StackQL MCP fuzz fixture on port ${FUZZ_PORT}..."
MCP_FUZZ_PORT="$FUZZ_PORT" go run ./scripts/fuzz_mcp_fixture/ >"$SERVER_LOG" 2>&1 &
SERVER_PID=$!

READY_JSON=""
for _ in $(seq 1 60); do
  if READY_JSON="$(grep -E '^\{.*"endpoint".*\}$' "$SERVER_LOG" 2>/dev/null | tail -1)"; then
    break
  fi
  if ! kill -0 "$SERVER_PID" 2>/dev/null; then
    echo "fuzz fixture server exited before ready:" >&2
    cat "$SERVER_LOG" >&2
    exit 1
  fi
  sleep 1
done

if [[ -z "$READY_JSON" ]]; then
  echo "timed out waiting for fuzz fixture server (60s)" >&2
  cat "$SERVER_LOG" >&2
  exit 1
fi

MCP_ENDPOINT="$(node -e 'console.log(JSON.parse(process.argv[1]).endpoint)' "$READY_JSON")"
echo "mcp endpoint: $MCP_ENDPOINT"

DOCKER_ARGS=(--rm)
if [[ "$(uname -s)" == "Linux" ]]; then
  DOCKER_ARGS+=(--network host)
else
  MCP_ENDPOINT="${MCP_ENDPOINT//localhost/host.docker.internal}"
  MCP_ENDPOINT="${MCP_ENDPOINT//127.0.0.1/host.docker.internal}"
fi

echo "pulling $FUZZ_IMAGE"
docker pull "$FUZZ_IMAGE"

echo "running mcp-fuzzer (runs=$FUZZ_RUNS timeout=${FUZZ_TIMEOUT}s)"
docker run "${DOCKER_ARGS[@]}" \
  -v "$OUTPUT_DIR:/output:rw" \
  "$FUZZ_IMAGE" \
  --mode all \
  --protocol streamablehttp \
  --endpoint "$MCP_ENDPOINT" \
  --auth-audit \
  --security-audit \
  --fail-if-no-tools \
  --runs "$FUZZ_RUNS" \
  --timeout "$FUZZ_TIMEOUT" \
  --output-dir /output

echo "fuzz complete; reports in $OUTPUT_DIR"
