import { CLOUD_SHELL_SCRIPTS, type CloudShellScript } from "./cloud-shell-scripts";

const GITHUB_REPO = "stackql/stackql";
const RELEASE_BASE = `https://github.com/${GITHUB_REPO}/releases/latest/download`;
const RELEASES_URL = `https://github.com/${GITHUB_REPO}/releases/latest`;
const DOCS_URL = "https://stackql.io/docs";

// Browser / known-OS callers send a User-Agent that identifies the platform, so
// the root URL can redirect them straight to the correct release asset. CLI
// download tools (curl, wget, PowerShell) don't carry the OS reliably, so we
// expose installer endpoints that detect OS + arch locally instead:
//
//   /install      - auto-detects the caller's shell from the User-Agent and
//                   serves the matching installer (sh for curl/wget, PowerShell
//                   for irm/iwr). This is the universal one-liner.
//   /install.sh   - always the POSIX sh installer (verbose / explicit).
//   /install.ps1  - always the PowerShell installer (verbose / explicit).
//
// Cloud shell / web terminal helpers add a provider segment:
//
//   /install/<provider>   - installs ./stackql plus the matching cloud shell
//   /install.sh/<provider>  helper (e.g. stackql-aws-cloud-shell.sh), both made
//                           executable. Providers: aws, google, azure, databricks.
//                           These web terminals are all Linux, so the helpers are
//                           only served to Linux callers; a macOS / Windows
//                           User-Agent gets a short "Linux downloads only" message
//                           instead. The generated installer also re-checks
//                           `uname` at runtime as a backstop.
//
// The explicit endpoints also guard against being run in the wrong shell: if the
// sh installer is fetched by PowerShell (or the PowerShell installer by
// curl/wget), we serve a short message - written in the language of the shell
// that's about to run it - that points at the right command. Each installer also
// guards at runtime (uname / $PSVersionTable) for cases the User-Agent missed,
// e.g. curl inside Git Bash on Windows.
//
// Asset shapes differ from a plain binary archive: Linux and Windows ship as zip
// archives containing the `stackql`/`stackql.exe` binary, while macOS ships only
// as a multi-arch `.pkg` installer. The sh installer extracts the binary from the
// pkg payload so the `curl | sh` flow still lands a `./stackql` in the cwd.

function getAssetName(ua: string): string {
  if (/windows/i.test(ua)) return "stackql_windows_amd64.zip";
  if (/darwin|macintosh|mac os/i.test(ua)) return "stackql_darwin_multiarch.pkg";
  return "stackql_linux_amd64.zip";
}

// True when the caller is PowerShell (Invoke-WebRequest / Invoke-RestMethod),
// which sets a User-Agent like "Mozilla/5.0 (Windows NT...) WindowsPowerShell/5.1".
function isPowerShell(ua: string): boolean {
  return /powershell/i.test(ua);
}

// True when the caller is a POSIX download tool that will pipe the body into sh.
function isPosixShellTool(ua: string): boolean {
  return /\bcurl\b|\bwget\b/i.test(ua);
}

const INSTALL_SH = `#!/bin/sh
# stackql installer - https://get-stackql.io/install.sh
# Detects OS + architecture and downloads the matching release binary into the
# current directory.
set -eu

base="${RELEASE_BASE}"
os=$(uname -s)
arch=$(uname -m)

case "$os" in
  Darwin)
    # macOS ships only as a multi-arch .pkg installer. Extract the universal
    # binary from the pkg payload instead of running a system-wide install.
    asset="stackql_darwin_multiarch.pkg"
    echo "Downloading $asset ..."
    tmp=$(mktemp -d)
    curl -fsSL "$base/$asset" -o "$tmp/stackql.pkg"
    pkgutil --expand-full "$tmp/stackql.pkg" "$tmp/expanded" >/dev/null 2>&1 || true
    bin=$(find "$tmp/expanded" -type f -name stackql 2>/dev/null | head -n1)
    if [ -z "$bin" ]; then
      echo "stackql: couldn't extract the binary from the macOS package." >&2
      echo "Install with Homebrew instead:  brew install stackql" >&2
      echo "Or run the installer directly:  $base/$asset" >&2
      rm -rf "$tmp"
      exit 1
    fi
    cp "$bin" ./stackql
    chmod +x ./stackql
    rm -rf "$tmp"
    echo "Installed ./stackql (macOS/$arch). Run it with ./stackql or move it onto your PATH."
    exit 0
    ;;
  Linux)
    case "$arch" in
      x86_64 | amd64)        asset="stackql_linux_amd64.zip" ;;
      aarch64 | arm64)       asset="stackql_linux_arm64.zip" ;;
      *)
        echo "stackql: there's no prebuilt Linux binary for your CPU ($arch)." >&2
        echo "Prebuilt Linux builds cover x86_64 (amd64) and arm64 (aarch64)." >&2
        echo "Browse all downloads: ${RELEASES_URL}" >&2
        exit 1
        ;;
    esac
    ;;
  MINGW* | MSYS* | CYGWIN* | Windows_NT)
    echo "stackql: this looks like Windows in a POSIX shell ($os)." >&2
    echo "The Windows build installs from PowerShell. Open PowerShell and run:" >&2
    echo "" >&2
    echo "    irm https://get-stackql.io/install.ps1 | iex" >&2
    echo "" >&2
    echo "Already in WSL and want the Linux build? Run this from your WSL shell." >&2
    exit 1
    ;;
  *)
    echo "stackql: this installer doesn't recognize your system ($os $arch)." >&2
    echo "See the install guide for every option: ${DOCS_URL}" >&2
    echo "Or download a binary directly: ${RELEASES_URL}" >&2
    exit 1
    ;;
esac

# Linux: download the zip and extract the stackql binary into the cwd.
echo "Downloading $asset ..."
if ! command -v unzip >/dev/null 2>&1; then
  echo "stackql: 'unzip' is required to extract $asset but was not found." >&2
  echo "Install it (e.g. apt-get install unzip) and re-run, or download directly:" >&2
  echo "  $base/$asset" >&2
  exit 1
fi
tmp=$(mktemp -d)
curl -fsSL "$base/$asset" -o "$tmp/stackql.zip"
unzip -oq "$tmp/stackql.zip" stackql -d "$tmp" 2>/dev/null || unzip -oq "$tmp/stackql.zip" -d "$tmp"
bin=$(find "$tmp" -type f -name stackql 2>/dev/null | head -n1)
if [ -z "$bin" ]; then
  echo "stackql: couldn't find the stackql binary inside $asset." >&2
  echo "Download it directly: $base/$asset" >&2
  rm -rf "$tmp"
  exit 1
fi
cp "$bin" ./stackql
chmod +x ./stackql
rm -rf "$tmp"
echo "Installed ./stackql ($os/$arch). Run it with ./stackql or move it onto your PATH."
`;

const INSTALL_PS1 = `#Requires -Version 5
# stackql installer - https://get-stackql.io/install.ps1
# Downloads the latest Windows release archive and extracts stackql.exe into the
# current directory.
# Usage: irm https://get-stackql.io/install.ps1 | iex
$ErrorActionPreference = 'Stop'

# Runtime guard: PowerShell also runs on macOS/Linux, where this Windows build
# won't help. Point those callers at the sh installer instead.
if ($PSVersionTable.Platform -eq 'Unix') {
  Write-Host "stackql: this is the Windows installer, but you're on PowerShell on a non-Windows OS."
  Write-Host "Install with:"
  Write-Host ""
  Write-Host "    curl -fsSL https://get-stackql.io/install.sh | sh"
  Write-Host ""
  return
}

$base = "${RELEASE_BASE}"
$asset = 'stackql_windows_amd64.zip'
$arch = $env:PROCESSOR_ARCHITECTURE
if ($arch -ne 'AMD64' -and $arch -ne 'ARM64') {
  Write-Host "stackql: there's no prebuilt Windows binary for your CPU ($arch)."
  Write-Host "Prebuilt Windows builds cover x64 (AMD64) and ARM64 (via emulation)."
  Write-Host "Browse all downloads: ${RELEASES_URL}"
  return
}

$dest = (Get-Location).Path
$zip = Join-Path $dest $asset
Write-Host "Downloading $asset ..."
Invoke-WebRequest -Uri "$base/$asset" -OutFile $zip
Expand-Archive -Path $zip -DestinationPath $dest -Force
Remove-Item $zip
Write-Host "Installed .\\stackql.exe. Run it with .\\stackql.exe or move it onto your PATH."
`;

// Served when the sh installer is fetched by PowerShell - valid PowerShell that
// just points the user at the Windows one-liner.
const GUIDE_USE_PS1 = `# stackql - wrong installer for this shell.
Write-Host "stackql: that's the Linux/macOS installer."
Write-Host "On Windows, install with:"
Write-Host ""
Write-Host "    irm https://get-stackql.io/install.ps1 | iex"
Write-Host ""
`;

// Served when the PowerShell installer is fetched by curl/wget - valid sh that
// just points the user at the Linux/macOS one-liner.
const GUIDE_USE_SH = `#!/bin/sh
# stackql - wrong installer for this shell.
echo "stackql: that's the Windows (PowerShell) installer."
echo "On macOS or Linux, install with:"
echo ""
echo "    curl -fsSL https://get-stackql.io/install.sh | sh"
echo ""
exit 1
`;

// Cloud shell helpers target Linux web terminals only. Browsers and CLI tools on
// Windows or macOS carry that OS in their User-Agent; bare curl/wget (the cloud
// shell case) carry no OS, so we default-allow them just like getAssetName()
// treats "/" requests as Linux.
function isLinuxShellTarget(ua: string): boolean {
  return !/windows|darwin|macintosh|mac os/i.test(ua);
}

// Served to non-Linux callers that ask for a cloud shell helper, in the language
// of the shell that's about to run it.
const LINUX_ONLY_SH = `#!/bin/sh
# stackql cloud shell helpers are Linux-only.
echo "stackql: cloud shell helper scripts are supported for Linux downloads only." >&2
echo "Install just the stackql binary with:" >&2
echo "" >&2
echo "    curl -fsSL https://get-stackql.io/install | sh" >&2
echo "" >&2
exit 1
`;

const LINUX_ONLY_PS1 = `# stackql cloud shell helpers are Linux-only.
Write-Host "stackql: cloud shell helper scripts are supported for Linux downloads only."
Write-Host "Install just the stackql binary with:"
Write-Host ""
Write-Host "    irm https://get-stackql.io/install | iex"
Write-Host ""
`;

// Builds an sh installer that drops ./stackql and the given cloud shell helper
// into the current directory, both executable. The helper body is embedded via a
// quoted heredoc so its contents are written verbatim.
function providerInstallSh(script: CloudShellScript): string {
  return `#!/bin/sh
# stackql cloud shell installer - https://get-stackql.io/install/<provider>
# Downloads the stackql binary and the ${script.file} helper into the current
# directory, both executable. For Linux web terminals / cloud shells.
set -eu

os=$(uname -s)
if [ "$os" != "Linux" ]; then
  echo "stackql: cloud shell helper scripts are supported for Linux downloads only." >&2
  echo "Detected $os. Install just the binary with: curl -fsSL https://get-stackql.io/install | sh" >&2
  exit 1
fi

base="${RELEASE_BASE}"
arch=$(uname -m)
case "$arch" in
  x86_64 | amd64)        asset="stackql_linux_amd64.zip" ;;
  aarch64 | arm64)       asset="stackql_linux_arm64.zip" ;;
  *)
    echo "stackql: there's no prebuilt Linux binary for your CPU ($arch)." >&2
    echo "Browse all downloads: ${RELEASES_URL}" >&2
    exit 1
    ;;
esac

if ! command -v unzip >/dev/null 2>&1; then
  echo "stackql: 'unzip' is required to extract $asset but was not found." >&2
  echo "Install it (e.g. apt-get install unzip) and re-run." >&2
  exit 1
fi

echo "Downloading $asset ..."
tmp=$(mktemp -d)
curl -fsSL "$base/$asset" -o "$tmp/stackql.zip"
unzip -oq "$tmp/stackql.zip" stackql -d "$tmp" 2>/dev/null || unzip -oq "$tmp/stackql.zip" -d "$tmp"
bin=$(find "$tmp" -type f -name stackql 2>/dev/null | head -n1)
if [ -z "$bin" ]; then
  echo "stackql: couldn't find the stackql binary inside $asset." >&2
  rm -rf "$tmp"
  exit 1
fi
cp "$bin" ./stackql
chmod +x ./stackql
rm -rf "$tmp"

# Write the cloud shell helper alongside the binary.
cat > ./${script.file} <<'STACKQL_CLOUD_SHELL_EOF'
${script.body.replace(/\n+$/, "")}
STACKQL_CLOUD_SHELL_EOF
chmod +x ./${script.file}

echo "Installed ./stackql and ./${script.file} (Linux/$arch)."
echo "Launch the cloud shell helper with: ./${script.file}"
`;
}

// Routes /install[.sh|.ps1]/<provider>. Unknown providers get a short message
// listing the supported ones; non-Linux callers get the Linux-only message.
function cloudShellResponse(provider: string, ua: string): Response {
  const script = CLOUD_SHELL_SCRIPTS[provider];
  if (!script) {
    const supported = Object.keys(CLOUD_SHELL_SCRIPTS).join(", ");
    // Don't reflect arbitrary path content back into the served script.
    const safe = provider.replace(/[^a-z0-9_-]/gi, "").slice(0, 40);
    if (isPowerShell(ua)) {
      return ps1Response(`# stackql - unknown cloud shell helper.
Write-Host "stackql: no cloud shell helper named '${safe}'. Supported: ${supported}."
`);
    }
    return shResponse(`#!/bin/sh
echo "stackql: no cloud shell helper named '${safe}'. Supported: ${supported}." >&2
exit 1
`);
  }
  if (!isLinuxShellTarget(ua)) {
    return isPowerShell(ua) ? ps1Response(LINUX_ONLY_PS1) : shResponse(LINUX_ONLY_SH);
  }
  return shResponse(providerInstallSh(script));
}

function shResponse(body: string): Response {
  return new Response(body, {
    headers: { "content-type": "text/x-shellscript; charset=utf-8" },
  });
}

function ps1Response(body: string): Response {
  return new Response(body, {
    headers: { "content-type": "text/plain; charset=utf-8" },
  });
}

export default {
  fetch(req: Request): Response {
    const url = new URL(req.url);
    const ua = req.headers.get("user-agent") ?? "";

    // Cloud shell helpers: /install/<provider>, /install.sh/<provider>, and
    // /install.ps1/<provider> (the last only to give Windows callers the
    // Linux-only message rather than a 301 to stackql.io).
    const segments = url.pathname.split("/").filter(Boolean);
    if (
      segments.length === 2 &&
      (segments[0] === "install" || segments[0] === "install.sh" || segments[0] === "install.ps1")
    ) {
      return cloudShellResponse(segments[1].toLowerCase(), ua);
    }

    // Universal installer: pick the script that matches the calling shell.
    if (url.pathname === "/install") {
      return isPowerShell(ua) ? ps1Response(INSTALL_PS1) : shResponse(INSTALL_SH);
    }

    // Explicit POSIX installer. If PowerShell fetched it, hand back a PowerShell
    // message instead of sh it can't run.
    if (url.pathname === "/install.sh") {
      return isPowerShell(ua) ? ps1Response(GUIDE_USE_PS1) : shResponse(INSTALL_SH);
    }

    // Explicit PowerShell installer. If curl/wget fetched it (i.e. it's about to
    // be piped into sh), hand back an sh message instead of PowerShell.
    if (url.pathname === "/install.ps1") {
      return isPosixShellTool(ua) ? shResponse(GUIDE_USE_SH) : ps1Response(INSTALL_PS1);
    }

    if (url.pathname !== "/") {
      return Response.redirect("https://stackql.io", 301);
    }

    const asset = getAssetName(ua);
    return Response.redirect(`${RELEASE_BASE}/${asset}`, 302);
  },
} satisfies ExportedHandler;
