# Source binaries (drop zone)

Drop the per-release, already-signed artefacts here before running
`scripts/package.sh`. Nothing in this directory is committed (see `.gitignore`).

The packaging script consumes the release artefacts directly - the same zips
and `.pkg` produced by the stackql release pipeline. No manual extraction
required.

Expected files (at the root of `bin/`):

```
bin/
  stackql_linux_amd64.zip        # Linux x86_64 release zip (contains stackql)
  stackql_linux_arm64.zip        # Linux arm64 release zip (contains stackql)
  stackql_windows_amd64.zip      # Windows release zip (contains stackql.exe, Authenticode-signed upstream)
  stackql_darwin_multiarch.pkg   # macOS notarised .pkg (universal binary inside)
```

The darwin filename glob is `stackql_darwin*.pkg`, so the exact suffix
(`_multiarch`, a version, etc.) does not matter.

Notes:

- The Windows `.exe` should already be Authenticode-signed by your normal
  post-build signing step. The bundle is just a zip, so the signature rides
  along unchanged.
- For macOS you drop the notarised `.pkg`, not a bare binary. The packaging
  script extracts the universal binary from the `.pkg` payload with
  `pkgutil --expand-full`. The extracted binary keeps its embedded code
  signature and remains notarisation-recognised online (the stapled ticket
  stays with the `.pkg`, which you publish separately).
- Any target whose source file is absent is skipped, so you can build a
  subset by dropping only some artefacts.
- The macOS target requires `pkgutil`, so run the script on macOS to produce
  the darwin bundle.

Fallback (legacy) layout - still accepted if you have pre-extracted binaries:

```
bin/
  linux-amd64/stackql
  linux-arm64/stackql
  windows-amd64/stackql.exe
  darwin/stackql-<version>.pkg
```
