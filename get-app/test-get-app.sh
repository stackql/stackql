#!/bin/sh
# Client-side smoke test for the stackql installer (mac/linux).
# Confirms the origin is Cloudflare, exercises every installer path / shell-guard,
# then runs the real installer and checks the binary is the right platform build,
# executable, and runnable. Prints a green PASS / red FAIL per check and a final
# colored summary.

set -u

BIN=stackql
BASE=https://get-stackql.io
INSTALL_URL="$BASE/install.sh"

# User-Agents the worker routes on: a POSIX download tool vs PowerShell.
UA_CURL="curl/8.4.0"
UA_PS="Mozilla/5.0 (Windows NT 10.0) WindowsPowerShell/5.1"

# Colors, only when stdout is a terminal (keeps piped/redirected output clean).
if [ -t 1 ]; then
  GREEN=$(printf '\033[32m'); RED=$(printf '\033[31m')
  BOLD=$(printf '\033[1m'); RESET=$(printf '\033[0m')
else
  GREEN=''; RED=''; BOLD=''; RESET=''
fi

FAILURES=0

pass() { printf '  %s%sPASS%s %s\n' "$BOLD" "$GREEN" "$RESET" "$1"; }
fail() { printf '  %s%sFAIL%s %s\n' "$BOLD" "$RED" "$RESET" "$1"; FAILURES=$((FAILURES + 1)); }

rm -f stackql
rm -f stackql.exe
rm -f stackql.zip
rm -f stackql.pkg
rm -f stackql-*-shell.sh

print_box() {
  msg="$1"
  width=$(( ${#msg} + 4 ))
  line=$(printf '%*s' "$width" '' | tr ' ' '-')
  printf '+%s+\n' "$line"
  printf '|  %s  |\n' "$msg"
  printf '+%s+\n' "$line"
}

# Fetch a body with a given User-Agent and assert it contains a substring.
check_body() {
  name="$1"; url="$2"; ua="$3"; expect="$4"
  body=$(curl -fsSL -A "$ua" "$url" 2>/dev/null) || { fail "$name (request failed)"; return; }
  case "$body" in
    *"$expect"*) pass "$name" ;;
    *) fail "$name (expected '$expect', got '$(printf '%s' "$body" | sed -n '1p')')" ;;
  esac
}

# Invalid versions must be rejected without reflecting the supplied value.
check_invalid_version() {
  name="$1"; url="$2"; ua="$3"
  response=$(curl -sS -w '\n%{http_code}' -A "$ua" "$url" 2>/dev/null) || { fail "$name (request failed)"; return; }
  status=$(printf '%s\n' "$response" | tail -n1)
  body=$(printf '%s\n' "$response" | sed '$d')
  if [ "$status" = "200" ] && printf '%s' "$body" | grep -Fq "invalid version" \
    && ! printf '%s' "$body" | grep -Fq "0.12.824"; then
    pass "$name"
  else
    fail "$name (expected an error script without reflecting the supplied value)"
  fi
}

# Force the generated sh installer to select a known-missing Linux/ARM64 asset.
check_missing_asset() {
  name="$1"
  tmp=$(mktemp -d)
  mkdir "$tmp/bin"
  curl -fsSL "$BASE/install.sh?version=0.5.418" -o "$tmp/install.sh" || {
    fail "$name (could not fetch pinned installer)"
    rm -rf "$tmp"
    return
  }
  cat > "$tmp/bin/uname" <<'EOF'
#!/bin/sh
case "$1" in
  -s) echo Linux ;;
  -m) echo aarch64 ;;
esac
EOF
  printf '#!/bin/sh\nexit 0\n' > "$tmp/bin/unzip"
  chmod +x "$tmp/bin/uname" "$tmp/bin/unzip"
  if (cd "$tmp" && PATH="$tmp/bin:$PATH" sh ./install.sh) > "$tmp/output" 2>&1; then
    fail "$name (installer unexpectedly succeeded)"
  elif grep -Fq "stackql 0.5.418 has no stackql_linux_arm64.zip" "$tmp/output" \
    && grep -Fq "https://github.com/stackql/stackql/releases/tag/v0.5.418" "$tmp/output"; then
    pass "$name"
  else
    fail "$name (expected readable missing-asset message and release link)"
  fi
  rm -rf "$tmp"
}

# Assert a path redirects (no -L) to a Location containing a substring.
check_redirect() {
  name="$1"; url="$2"; ua="$3"; expect="$4"
  loc=$(curl -fsS -o /dev/null -D - -A "$ua" "$url" 2>/dev/null \
    | awk -F': ' 'tolower($1)=="location"{print $2}' | tr -d '\r')
  case "$loc" in
    *"$expect"*) pass "$name -> $loc" ;;
    *) fail "$name (expected Location '$expect', got '${loc:-<none>}')" ;;
  esac
}

# Print $3 bytes of file $1 from offset $2 as one lowercase hex string.
hex_at() { od -An -tx1 -j "$2" -N "$3" "$1" | tr -d ' \n'; }

print_box "Installing StackQL for MacOS/Linux"

echo "Origin check:"
server=$(curl -fsSL -D - -o /dev/null "$INSTALL_URL" 2>/dev/null \
  | awk -F': ' 'tolower($1)=="server"{print $2}' | tr -d '\r')
case "$(printf '%s' "$server" | tr 'A-Z' 'a-z')" in
  *cloudflare*) pass "served by Cloudflare (server: ${server:-<none>})" ;;
  *) fail "expected Cloudflare origin, got '${server:-<none>}'" ;;
esac
echo

echo "Endpoint routing:"
# /install auto-detects the calling shell.
check_body "/install (curl)        -> sh installer"        "$BASE/install"     "$UA_CURL" "#!/bin/sh"
check_body "/install (powershell)  -> ps1 installer"       "$BASE/install"     "$UA_PS"   "#Requires -Version 5"
# Explicit endpoints serve their real script for the matching shell.
check_body "/install.sh (curl)     -> sh installer"        "$BASE/install.sh"  "$UA_CURL" "#!/bin/sh"
check_body "/install.ps1 (ps)      -> ps1 installer"       "$BASE/install.ps1" "$UA_PS"   "#Requires -Version 5"
check_body "/install (pinned query) -> pinned sh installer" "$BASE/install?version=0.12.824" "$UA_CURL" 'base="https://releases.stackql.io/stackql/0.12.824"'
check_body "/install.sh (v-prefixed path) -> pinned sh" "$BASE/v0.12.824/install.sh" "$UA_CURL" 'base="https://releases.stackql.io/stackql/0.12.824"'
check_body "/install.ps1 (pinned query) -> pinned ps1" "$BASE/install.ps1?version=v0.12.824" "$UA_PS" 'https://releases.stackql.io/stackql/0.12.824'
check_body "/install/aws (pinned path) -> pinned helper" "$BASE/v0.12.824/install/aws" "$UA_CURL" 'base="https://releases.stackql.io/stackql/0.12.824"'
check_invalid_version "invalid version rejected without reflection" "$BASE/install?version=0.12.824%3Becho%20pwned" "$UA_CURL"
check_invalid_version "invalid version path rejected" "$BASE/not-a-version/install" "$UA_CURL"
# Wrong-shell guards point at the correct command instead of erroring.
check_body "/install.sh (ps)       -> 'use install.ps1'"   "$BASE/install.sh"  "$UA_PS"   "install.ps1 | iex"
check_body "/install.ps1 (curl)    -> 'use install.sh'"    "$BASE/install.ps1" "$UA_CURL" "install.sh | sh"
echo

echo "Cloud shell helper routing:"
# Provider routes serve an sh installer that bundles ./stackql + the helper.
check_body "/install/aws (curl)        -> aws helper"        "$BASE/install/aws"        "$UA_CURL" "stackql-aws-cloud-shell.sh"
check_body "/install.sh/google (curl)  -> google helper"     "$BASE/install.sh/google"  "$UA_CURL" "stackql-google-cloud-shell.sh"
check_body "/install/azure (curl)      -> azure helper"      "$BASE/install/azure"      "$UA_CURL" "stackql-azure-cloud-shell.sh"
check_body "/install/databricks (curl) -> databricks helper" "$BASE/install/databricks" "$UA_CURL" "stackql-databricks-shell.sh"
# Helpers are Linux-only: macOS / Windows User-Agents get a friendly message, not a script.
check_body "/install/aws (macOS UA)    -> Linux-only msg"    "$BASE/install/aws" "Mozilla/5.0 (Macintosh; Intel Mac OS X)" "Linux downloads only"
check_body "/install/aws (windows UA)  -> Linux-only msg"    "$BASE/install/aws" "$UA_PS"   "Linux downloads only"
# Unknown providers are rejected, listing the supported ones.
check_body "/install/bogus (curl)      -> unknown provider"  "$BASE/install/bogus" "$UA_CURL" "no cloud shell helper"
echo

echo "Root + fallback redirects:"
check_redirect "/ (linux UA)" "$BASE/" "$UA_CURL"                                 "stackql_linux_amd64.zip"
check_redirect "/ (macOS UA)" "$BASE/" "Mozilla/5.0 (Macintosh; Intel Mac OS X)"  "stackql_darwin_multiarch.pkg"
check_redirect "/ (windows UA)" "$BASE/" "Mozilla/5.0 (Windows NT 10.0; Win64)"   "stackql_windows_amd64.zip"
check_redirect "/v0.12.824 (pinned path)" "$BASE/v0.12.824" "$UA_CURL"           "stackql_linux_amd64.zip"
check_redirect "/ (pinned query)" "$BASE/?version=0.12.824" "$UA_CURL"            "/stackql/0.12.824/stackql_linux_amd64.zip"
check_redirect "/some/other/path" "$BASE/some/other/path" "$UA_CURL"              "stackql.io"
echo

echo "Missing asset handling:"
check_missing_asset "pinned missing Linux/ARM64 asset reports release link"
echo

echo "Running installer:"
curl -fsSL "$INSTALL_URL" | sh
if [ -e "$BIN" ]; then
  pass "installer downloaded $BIN"
else
  fail "installer did not produce $BIN (expected on Windows/Git Bash; run on mac/linux for the full path)"
fi
echo

if [ -e "$BIN" ]; then
  echo "Binary:"
  if command -v file >/dev/null 2>&1; then
    file "$BIN"
  else
    echo "  (file not available, skipping arch detail)"
  fi
  echo

  echo "Architecture check:"
  case "$(uname -s)" in
    Darwin)
      # macOS ships one universal (fat) binary, so both slices must be present.
      # Fat header: big-endian magic and slice count, then a 20-byte entry per
      # slice that starts with its CPU type.
      slices=""
      if [ "$(hex_at "$BIN" 0 4)" = "cafebabe" ]; then
        n=$(( 0x$(hex_at "$BIN" 4 4) ))
        i=0
        while [ "$i" -lt "$n" ]; do
          case "$(hex_at "$BIN" $((8 + 20 * i)) 4)" in
            01000007) slices="$slices x86_64" ;;
            0100000c) slices="$slices arm64" ;;
          esac
          i=$((i + 1))
        done
      fi
      case "$slices" in
        *x86_64*arm64* | *arm64*x86_64*) pass "$BIN is a universal macOS binary (x86_64 + arm64)" ;;
        *) fail "$BIN is not a universal x86_64 + arm64 macOS binary (slices:${slices:- none})" ;;
      esac
      ;;
    Linux)
      # Linux ships separate amd64 and arm64 binaries, so the ELF machine field
      # (offset 18, little-endian) must match this host.
      case "$(hex_at "$BIN" 18 2)" in
        3e00) bin_arch=amd64 ;;
        b700) bin_arch=arm64 ;;
        *)    bin_arch=unknown ;;
      esac
      case "$(uname -m)" in
        x86_64 | amd64)  host_arch=amd64 ;;
        aarch64 | arm64) host_arch=arm64 ;;
        *)               host_arch=$(uname -m) ;;
      esac
      if [ "$bin_arch" = "$host_arch" ]; then
        pass "$BIN is the native Linux/$host_arch build"
      else
        fail "$BIN is a Linux/$bin_arch build, expected native Linux/$host_arch"
      fi
      ;;
    *)
      echo "  (skipped: no architecture check for $(uname -s))"
      ;;
  esac
  echo

  echo "Permissions:"
  ls -l "$BIN"
  if [ -x "$BIN" ]; then
    pass "$BIN is executable"
  else
    fail "$BIN is not executable"
  fi
  echo

  echo "Execution check:"
  if ./"$BIN" --version; then
    pass "runnable $BIN for $(uname -s)/$(uname -m)"
  else
    fail "$BIN did not run (wrong binary or exec format error)"
  fi
  echo
fi

# Cloud shell helper install is Linux-only (the web terminals are all Linux), so
# only run the end-to-end unpack where the installer will actually proceed.
echo "Running cloud shell installer:"
if [ "$(uname -s)" = "Linux" ]; then
  rm -f stackql-aws-cloud-shell.sh
  curl -fsSL "$BASE/install/aws" | sh
  helper=stackql-aws-cloud-shell.sh
  if [ -e "$helper" ] && [ -x "$helper" ]; then
    pass "aws installer dropped an executable $helper alongside $BIN"
  else
    fail "aws installer did not produce an executable $helper"
  fi
else
  echo "  (skipped: cloud shell helpers are Linux-only; this host is $(uname -s))"
fi
echo

# Final summary.
if [ "$FAILURES" -eq 0 ]; then
  color=$GREEN; text="  PASS - all checks passed  "
else
  color=$RED; text="  FAIL - $FAILURES check(s) failed  "
fi
line=$(printf '%*s' "${#text}" '' | tr ' ' '-')
printf '%s%s+%s+%s\n' "$BOLD" "$color" "$line" "$RESET"
printf '%s%s|%s|%s\n' "$BOLD" "$color" "$text" "$RESET"
printf '%s%s+%s+%s\n' "$BOLD" "$color" "$line" "$RESET"

[ "$FAILURES" -eq 0 ]
