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
# Wrong-shell guards point at the correct command instead of erroring.
check_body "/install.sh (ps)       -> 'use install.ps1'"   "$BASE/install.sh"  "$UA_PS"   "install.ps1 | iex"
check_body "/install.ps1 (curl)    -> 'use install.sh'"    "$BASE/install.ps1" "$UA_CURL" "install.sh | sh"
echo

echo "Root + fallback redirects:"
check_redirect "/ (linux UA)" "$BASE/" "$UA_CURL"                                 "stackql_linux_amd64.zip"
check_redirect "/ (macOS UA)" "$BASE/" "Mozilla/5.0 (Macintosh; Intel Mac OS X)"  "stackql_darwin_multiarch.pkg"
check_redirect "/ (windows UA)" "$BASE/" "Mozilla/5.0 (Windows NT 10.0; Win64)"   "stackql_windows_amd64.zip"
check_redirect "/some/other/path" "$BASE/some/other/path" "$UA_CURL"              "stackql.io"
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
