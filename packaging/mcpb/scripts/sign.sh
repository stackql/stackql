#!/usr/bin/env bash
#
# sign.sh - envelope-sign existing dist/*.mcpb bundles and regenerate their
# .sha256 companions (the signature is appended to the bundle bytes, so the
# checksum must be recomputed after signing).
#
# Same env contract as package.sh:
#   self-signed (testing):
#     MCPB_SELF_SIGN=true scripts/sign.sh
#   production cert:
#     MCPB_SIGN_CERT=cert.pem MCPB_SIGN_KEY=key.pem \
#     [MCPB_SIGN_INTERMEDIATES="intermediate-ca.pem root-ca.pem"] \
#     scripts/sign.sh
#
# With neither configured, prints a notice and exits 0 so CI can call it
# unconditionally as a publish step.
#
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${DIST_DIR:-$ROOT_DIR/dist}"

if command -v mcpb >/dev/null 2>&1; then
  mcpb() { command mcpb "$@"; }
else
  mcpb() { npx --yes @anthropic-ai/mcpb "$@"; }
fi

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

verify_bundle() {
  # 'mcpb verify' is currently broken upstream (the CLI calls node-forge's
  # p7.verify, which is not implemented, so every signed bundle reports as
  # unsigned). Treat its result as advisory and assert the appended
  # signature block directly.
  local f="$1"
  if mcpb verify "$f"; then
    return 0
  fi
  if tail -c 64 "$f" | grep -aq "MCPB_SIG_END"; then
    echo "  warn: 'mcpb verify' failed but the signature block is present (known upstream CLI bug)"
    return 0
  fi
  echo "  error: no signature block found after signing $(basename "$f")" >&2
  return 1
}

if [ "${MCPB_SELF_SIGN:-false}" != "true" ] && { [ -z "${MCPB_SIGN_CERT:-}" ] || [ -z "${MCPB_SIGN_KEY:-}" ]; }; then
  echo "bundle signing skipped (set MCPB_SELF_SIGN=true or MCPB_SIGN_CERT + MCPB_SIGN_KEY)"
  exit 0
fi

shopt -s nullglob
bundles=( "$DIST_DIR"/stackql-mcp-*.mcpb )
if [ ${#bundles[@]} -eq 0 ]; then
  echo "error: no bundles found in $DIST_DIR" >&2
  exit 1
fi

for f in "${bundles[@]}"; do
  if [ "${MCPB_SELF_SIGN:-false}" = "true" ]; then
    echo "==> signing (self-signed): $(basename "$f")"
    mcpb sign "$f" --self-signed
  else
    echo "==> signing (production cert): $(basename "$f")"
    if [ -n "${MCPB_SIGN_INTERMEDIATES:-}" ]; then
      # shellcheck disable=SC2086
      mcpb sign "$f" --cert "$MCPB_SIGN_CERT" --key "$MCPB_SIGN_KEY" --intermediate $MCPB_SIGN_INTERMEDIATES
    else
      mcpb sign "$f" --cert "$MCPB_SIGN_CERT" --key "$MCPB_SIGN_KEY"
    fi
  fi
  verify_bundle "$f"
  sha_file "$f"
done

echo "signed ${#bundles[@]} bundle(s) and regenerated checksums."
