# Source binaries (drop zone)

Drop the per-release, already-signed binaries here before running
`scripts/package.sh`. Nothing in this directory is committed (see `.gitignore`).

Expected layout:

```
bin/
  linux-amd64/stackql            # Linux x86_64 binary
  linux-arm64/stackql            # Linux arm64 binary
  windows-amd64/stackql.exe      # Windows binary, Authenticode-signed upstream
  darwin/stackql-<version>.pkg   # macOS notarised .pkg (universal binary inside)
```

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
  subset by dropping only some binaries.
- The macOS target requires `pkgutil`, so run the script on macOS to produce
  the darwin bundle.
