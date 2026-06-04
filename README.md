# stackql-mcpb-packaging

Packages the StackQL MCP server into per-platform [MCPB](https://github.com/anthropics/mcpb)
bundles (`.mcpb`) for distribution and for listing on the official MCP Registry.

This is a standalone, scripted post-release step. It does not build or sign the
stackql binaries - that happens upstream in the normal stackql build and signing
process. Here you drop the already-signed binaries (and the notarised macOS
`.pkg`) into `bin/`, run one script, and get signed `.mcpb` bundles plus
checksums in `dist/`, which you then attach to the matching GitHub release.

## What gets packaged

The server is the `stackql` binary itself, launched as `stackql mcp` over stdio
(see `manifest/manifest.template.json`). The separate `stackql_mcp_client`
binary is a test client and is not packaged.

One bundle is produced per target:

- `stackql-mcp-linux-x64.mcpb`
- `stackql-mcp-linux-arm64.mcpb`
- `stackql-mcp-windows-x64.mcpb`
- `stackql-mcp-darwin-universal.mcpb` (one universal binary covers both Mac arches)

## Layout

```
stackql-mcpb-packaging/
  manifest/manifest.template.json   # MCPB manifest, tokenised (__VERSION__, __BINARY_NAME__)
  scripts/package.sh                # build bundles from bin/ -> dist/
  scripts/clean.sh                  # wipe dist/
  bin/                              # drop signed binaries here (gitignored)
  dist/                             # generated bundles land here (gitignored)
```

## Prerequisites

- Node.js (the `@anthropic-ai/mcpb` CLI is invoked via `npx`; install it
  globally with `npm i -g @anthropic-ai/mcpb` if you prefer).
- macOS with `pkgutil` for the darwin target (needed to extract the binary from
  the `.pkg`). The Linux and Windows targets have no macOS dependency and can be
  built anywhere. Running the whole thing once on a Mac produces all four.

## Usage

1. Drop the per-release signed binaries into `bin/` (see `bin/README.md` for the
   expected layout).
2. Run the packager with the release version:

   ```bash
   ./scripts/package.sh --version 1.2.3
   ```

3. Attach everything in `dist/` (the `.mcpb` files and their `.sha256` files) to
   the same GitHub release as the stackql build.

Any missing source binary is skipped with a notice, so a partial drop builds a
partial set.

## macOS: extracting from the .pkg

For darwin you drop the notarised `.pkg` rather than a bare binary. The script
runs `pkgutil --expand-full` and pulls the universal `stackql` binary out of the
payload. This works because:

- The code signature is embedded in the Mach-O, so extraction preserves it.
- Notarisation is keyed to the binary's cdhash, which is registered with Apple
  when you notarise the `.pkg`. The identical extracted binary is therefore
  recognised by Gatekeeper online.
- The stapled notarisation ticket lives on the `.pkg` itself (you cannot staple
  a bare binary), so the `.pkg` remains the offline-validating installer and the
  `.mcpb` relies on online validation of the same binary.

## Bundle signing

OS-level code signing of the binaries is done upstream (Authenticode on Windows,
Developer ID + notarisation on macOS). Separately, you can sign the `.mcpb`
bundle itself, which Claude Desktop verifies. Signing is off by default.

Self-signed (testing only):

```bash
MCPB_SELF_SIGN=true ./scripts/package.sh --version 1.2.3
```

Production certificate:

```bash
MCPB_SIGN_CERT=cert.pem \
MCPB_SIGN_KEY=key.pem \
MCPB_SIGN_INTERMEDIATES="intermediate-ca.pem root-ca.pem" \
./scripts/package.sh --version 1.2.3
```

`MCPB_SIGN_INTERMEDIATES` is optional and space-separated. Each bundle is
verified with `mcpb verify` after signing.

## Listing on the MCP Registry

The registry stores metadata only and points at these release assets. After the
bundles are attached to the release, add one package entry per platform to your
`server.json`, each with:

- `"registryType": "mcpb"`
- `"identifier"`: the release download URL of that `.mcpb` (the URL must contain
  the string `mcp`, which the filenames satisfy)
- `"fileSha256"`: the hash from the matching `.sha256` file
- `"transport": { "type": "stdio" }`

Then publish with the `mcp-publisher` CLI. Clients verify the SHA-256 before
installing.

## Note on the launch command

The manifest passes `args: ["mcp"]`. If running stackql as a stdio MCP server
needs more than the bare subcommand (a flag, a registry path, etc.), update
`args` in `manifest/manifest.template.json` to the exact invocation - the client
launches the bundled binary with precisely those arguments.
