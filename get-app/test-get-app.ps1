# Client-side smoke test for the stackql installer (Windows).
# Confirms the origin is Cloudflare, exercises every installer path / shell-guard,
# then runs the real installer and checks the binary downloads and runs. Prints a
# green PASS / red FAIL per check and a final colored summary.
# Works on Windows PowerShell 5.1 and PowerShell 7+.

$ErrorActionPreference = 'Stop'

$Bin = 'stackql.exe'
$Base = 'https://get-stackql.io'
$InstallUrl = "$Base/install.ps1"

# User-Agents the worker routes on: PowerShell vs a POSIX download tool.
$UaPs = 'Mozilla/5.0 (Windows NT 10.0) WindowsPowerShell/5.1'
$UaCurl = 'curl/8.4.0'

$script:Failures = 0

function Pass { param([string]$Name) Write-Host "  PASS " -ForegroundColor Green -NoNewline; Write-Host $Name }
function Fail { param([string]$Name) Write-Host "  FAIL " -ForegroundColor Red -NoNewline; Write-Host $Name; $script:Failures++ }

foreach ($f in @('stackql', 'stackql.exe', 'stackql.zip', 'stackql.pkg')) {
  if (Test-Path $f) { Remove-Item $f -Force }
}

Add-Type -AssemblyName System.Net.Http

# Fetch a URL with a given User-Agent without following redirects. Returns the
# status, Location header, Server header, and body so each check can assert.
function Get-Resp {
  param([string]$Url, [string]$Ua)
  $handler = [System.Net.Http.HttpClientHandler]::new()
  $handler.AllowAutoRedirect = $false
  $client = [System.Net.Http.HttpClient]::new($handler)
  try {
    $msg = [System.Net.Http.HttpRequestMessage]::new([System.Net.Http.HttpMethod]::Get, $Url)
    [void]$msg.Headers.TryAddWithoutValidation('User-Agent', $Ua)
    $resp = $client.SendAsync($msg).Result
    $body = $resp.Content.ReadAsStringAsync().Result
    $location = ''
    if ($resp.Headers.Location) { $location = $resp.Headers.Location.ToString() }
    $server = ''
    $vals = $null
    if ($resp.Headers.TryGetValues('Server', [ref]$vals)) { $server = ($vals -join '') }
    [pscustomobject]@{
      Status   = [int]$resp.StatusCode
      Location = $location
      Server   = $server
      Body     = $body
    }
  } finally {
    $client.Dispose()
    $handler.Dispose()
  }
}

function Assert-Body {
  param([string]$Name, [string]$Url, [string]$Ua, [string]$Expect)
  try { $resp = Get-Resp -Url $Url -Ua $Ua } catch { Fail "$Name (request failed)"; return }
  if ($resp.Body -like "*$Expect*") {
    Pass $Name
  } else {
    $first = ($resp.Body -split "`n" | Select-Object -First 1)
    Fail "$Name (expected '$Expect', got '$first')"
  }
}

function Assert-Location {
  param([string]$Name, [string]$Url, [string]$Ua, [string]$Expect)
  try { $resp = Get-Resp -Url $Url -Ua $Ua } catch { Fail "$Name (request failed)"; return }
  if ($resp.Location -like "*$Expect*") {
    Pass "$Name -> $($resp.Location)"
  } else {
    $got = if ($resp.Location) { $resp.Location } else { '<none>' }
    Fail "$Name (expected Location '$Expect', got '$got')"
  }
}

function Write-Box {
  param([string]$Msg)
  $line = '-' * ($Msg.Length + 4)
  Write-Host "+$line+"
  Write-Host "|  $Msg  |"
  Write-Host "+$line+"
}

Write-Box "Installing StackQL for Windows"

Write-Host "Origin check:"
try { $origin = Get-Resp -Url $InstallUrl -Ua $UaPs } catch { $origin = $null }
if ($origin -and $origin.Server -like '*cloudflare*') {
  Pass "served by Cloudflare (server: $($origin.Server))"
} else {
  $got = if ($origin) { $origin.Server } else { '<none>' }
  Fail "expected Cloudflare origin, got '$got'"
}
Write-Host ""

Write-Host "Endpoint routing:"
# /install auto-detects the calling shell.
Assert-Body "/install (powershell)  -> ps1 installer"     "$Base/install"     $UaPs   '#Requires -Version 5'
Assert-Body "/install (curl)        -> sh installer"      "$Base/install"     $UaCurl '#!/bin/sh'
# Explicit endpoints serve their real script for the matching shell.
Assert-Body "/install.ps1 (ps)      -> ps1 installer"     "$Base/install.ps1" $UaPs   '#Requires -Version 5'
Assert-Body "/install.sh (curl)     -> sh installer"      "$Base/install.sh"  $UaCurl '#!/bin/sh'
# Wrong-shell guards point at the correct command instead of erroring.
Assert-Body "/install.ps1 (curl)    -> 'use install.sh'"  "$Base/install.ps1" $UaCurl 'install.sh | sh'
Assert-Body "/install.sh (ps)       -> 'use install.ps1'" "$Base/install.sh"  $UaPs   'install.ps1 | iex'
Write-Host ""

Write-Host "Root + fallback redirects:"
Assert-Location "/ (windows UA)"    "$Base/" $UaPs                                       'stackql_windows_amd64.zip'
Assert-Location "/ (linux UA)"      "$Base/" $UaCurl                                     'stackql_linux_amd64.zip'
Assert-Location "/ (macOS UA)"      "$Base/" 'Mozilla/5.0 (Macintosh; Intel Mac OS X)'   'stackql_darwin_multiarch.pkg'
Assert-Location "/some/other/path"  "$Base/some/other/path" $UaCurl                      'stackql.io'
Write-Host ""

Write-Host "Running installer:"
try {
  Invoke-RestMethod $InstallUrl | Invoke-Expression
} catch {
  Fail "installer raised an error: $($_.Exception.Message)"
}
if (Test-Path $Bin) {
  Pass "installer downloaded $Bin"
} else {
  Fail "installer did not produce $Bin"
}
Write-Host ""

if (Test-Path $Bin) {
  Write-Host "Binary:"
  $item = Get-Item $Bin
  Write-Host ("  {0}  {1:N0} bytes" -f $item.Name, $item.Length)
  Write-Host ""

  Write-Host "Execution check:"
  try {
    & ".\$Bin" --version
    Pass "runnable $Bin for Windows/$env:PROCESSOR_ARCHITECTURE"
  } catch {
    Fail "$Bin did not run on this platform"
  }
  Write-Host ""
}

# Final summary.
if ($script:Failures -eq 0) {
  $color = 'Green'; $text = "  PASS - all checks passed  "
} else {
  $color = 'Red'; $text = "  FAIL - $($script:Failures) check(s) failed  "
}
$line = '+' + ('-' * $text.Length) + '+'
Write-Host $line -ForegroundColor $color
Write-Host "|$text|" -ForegroundColor $color
Write-Host $line -ForegroundColor $color

if ($script:Failures -ne 0) { exit 1 }
