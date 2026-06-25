# get-stackql.io

Cloudflare Worker that backs `https://get-stackql.io`. It detects the calling
platform and points the caller at the correct `stackql` release asset on GitHub.

Behaviour:

- `GET /` - reads the `User-Agent` and `302`-redirects to the matching release
  asset (`stackql_windows_amd64.zip`, `stackql_darwin_multiarch.pkg`, or
  `stackql_linux_amd64.zip`).
- `GET /install` - universal installer. Detects the calling shell from the
  User-Agent and serves the matching script: the POSIX `sh` installer for
  curl/wget, or the PowerShell installer for `irm`/`iwr`. Use it as
  `curl -fsSL https://get-stackql.io/install | sh` (Linux/macOS) or
  `irm https://get-stackql.io/install | iex` (Windows).
- `GET /install.sh` - always the POSIX `sh` installer. Runs `uname` client-side
  to pick the right OS + arch asset and drops `./stackql` in the current dir.
- `GET /install.ps1` - always the PowerShell installer. Downloads and expands the
  Windows zip (`stackql.exe`) into the current directory.
- `GET /install/<provider>` (also `GET /install.sh/<provider>`) - cloud shell /
  web terminal helper installer. Serves an `sh` installer that drops `./stackql`
  **and** the matching helper script (e.g. `stackql-aws-cloud-shell.sh`) into the
  current directory, both executable. Providers: `aws`, `google`, `azure`,
  `databricks`. Use it as `curl -fsSL https://get-stackql.io/install/aws | sh`.
- Any other path - `301`-redirects to `https://stackql.io`.

### Cloud shell helpers

The helper scripts (`stackql-aws-cloud-shell.sh` etc.) used to be packaged into
the Linux release zip. They are now embedded in the worker
([`src/cloud-shell-scripts.ts`](src/cloud-shell-scripts.ts)) and delivered on
demand via `/install/<provider>`. Each provider installer downloads the Linux
binary, writes the helper alongside it, and makes both executable.

These web terminals (AWS CloudShell, Google Cloud Shell, Azure Cloud Shell, the
Databricks web terminal) are all Linux, so the helpers are served to Linux
callers only. A macOS or Windows `User-Agent` is answered with a short
`cloud shell helper scripts are supported for Linux downloads only` message
instead of a script, and the generated installer re-checks `uname` at runtime as
a backstop. Bare `curl`/`wget` (no OS in the UA, as in a cloud shell) are treated
as Linux. Unknown providers get a message listing the supported ones.

### Asset shapes

stackql release assets are not uniform, so the installers do a little more than a
plain download-and-extract:

- Linux (`stackql_linux_amd64.zip`, `stackql_linux_arm64.zip`) and Windows
  (`stackql_windows_amd64.zip`) ship as zip archives containing the
  `stackql`/`stackql.exe` binary.
- macOS ships only as a multi-arch `.pkg` installer
  (`stackql_darwin_multiarch.pkg`). The `sh` installer extracts the universal
  binary from the pkg payload (`pkgutil --expand-full`) so the `curl | sh` flow
  still lands a `./stackql` in the current directory rather than running a
  system-wide install. `brew install stackql` remains the recommended macOS path.

### Wrong-shell guards

The installers give friendly guidance instead of cryptic interpreter errors when
run in the wrong shell:

- Fetch `/install.sh` with PowerShell -> served a short PowerShell message
  pointing at `irm .../install.ps1 | iex`.
- Fetch `/install.ps1` with curl/wget -> served a short `sh` message pointing at
  `curl -fsSL .../install.sh | sh`.
- `install.sh` run in a POSIX shell on Windows (Git Bash/MSYS - `uname` reports
  `MINGW*`/`MSYS*`/`CYGWIN*`) -> message pointing at the PowerShell command.
- `install.ps1` run under PowerShell on macOS/Linux (`$PSVersionTable.Platform`
  is `Unix`) -> message pointing at the `sh` command.
- Unsupported CPU architectures get a "no prebuilt binary for your CPU" message
  with a link to the releases page, not "unsupported".

## Develop

```sh
npm install
npm run dev        # wrangler dev - serves on http://localhost:8787
```

Test locally:

```sh
curl -A "curl/8.4.0" http://localhost:8787/install            # -> sh installer
curl -A "WindowsPowerShell/5.1" http://localhost:8787/install  # -> PowerShell installer
curl -A "WindowsPowerShell/5.1" http://localhost:8787/install.sh   # -> "use install.ps1" guide
curl -A "curl/8.4.0" http://localhost:8787/install.ps1            # -> "use install.sh" guide
curl -A "curl/8.4.0" http://localhost:8787/install/aws            # -> aws cloud shell installer
curl -A "Mozilla/5.0 (Macintosh)" http://localhost:8787/install/aws  # -> "Linux downloads only"
curl -sI -A "Mozilla/5.0 (Macintosh)" http://localhost:8787/   # -> 302 to macos pkg
curl -sI -A "curl/8.4.0" http://localhost:8787/                # -> 302 to linux zip
```

## Smoke test

The repo root carries client-side smoke tests that exercise every endpoint and
shell guard against the live site, then run the real installer and check the
downloaded binary:

```sh
sh ../test-get-app.sh            # macOS / Linux
```

```powershell
..\test-get-app.ps1              # Windows
```

## Deploy

This Worker is deployed manually (not via CI). One-time auth (uses your
Cloudflare login):

```sh
npx wrangler login
```

Deploy:

```sh
npm run deploy     # wrangler deploy
```

`wrangler.toml` uses a `custom_domain` route for `get-stackql.io`. On the first
deploy Wrangler creates and manages the proxied DNS record for the apex
automatically - no manual DNS entry required. The `get-stackql.io` zone must
already exist in the target Cloudflare account.

If the apex already holds a record (e.g. an older deployment), the first deploy's
custom-domain trigger fails with a `409 Conflict` on `.../domains/records` and the
output reads `No targets deployed`. Clear the conflict via Workers & Pages ->
get-stackql -> Settings -> Domains & Routes -> Add -> Custom Domain ->
`get-stackql.io`, which prompts to override the existing record in a single step,
then re-run `npm run deploy`. The output should report
`get-stackql.io (custom domain)`.

Tail live logs:

```sh
npm run tail
```
