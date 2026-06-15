# Installing the StackQL MCP server

There are six ways to get the StackQL MCP server into a client (Claude Desktop, Cursor, VS Code, Cline, etc.). The first three are ordered by how much trust and how little effort each takes; npx, uvx, and Docker suit npx-shaped client directories and containerised environments respectively.

## 1. From a marketplace / directory (recommended once listings are live)

- **Claude Desktop -> Browse extensions** - search for "StackQL". Click install. (Available once the Anthropic Desktop Extensions submission is accepted; see [anthropic-submission.md](anthropic-submission.md).)
- **Cursor / VS Code / Cline marketplaces** - search for "StackQL"; listings auto-ingest from the Official MCP Registry.

These flows are signed by the directory's own review process and verified against the SHA-256 we publish.

## 2. Direct `.mcpb` install (Claude Desktop, manual)

Download the `.mcpb` for your platform from the latest stackql release:

- Linux x86_64: `https://github.com/stackql/stackql/releases/latest/download/stackql-mcp-linux-x64.mcpb`
- Linux arm64: `https://github.com/stackql/stackql/releases/latest/download/stackql-mcp-linux-arm64.mcpb`
- Windows x86_64: `https://github.com/stackql/stackql/releases/latest/download/stackql-mcp-windows-x64.mcpb`
- macOS (universal): `https://github.com/stackql/stackql/releases/latest/download/stackql-mcp-darwin-universal.mcpb`

Each `.mcpb` has a matching `.sha256` next to it on the release page. To verify before installing:

```bash
# Linux / macOS
sha256sum -c stackql-mcp-linux-x64.mcpb.sha256
# Windows (PowerShell)
(Get-FileHash stackql-mcp-windows-x64.mcpb -Algorithm SHA256).Hash -eq (Get-Content stackql-mcp-windows-x64.mcpb.sha256).Split()[0]
```

Then drag the `.mcpb` onto Claude Desktop (or open it from the Extensions panel). Claude Desktop will currently show the bundle as unsigned - that is expected (see [Trust model](#trust-model) below); the embedded binary inside is fully Apple-notarised / Authenticode-signed depending on platform.

## 3. Manual `claude_desktop_config.json` (any client that supports stdio MCP)

If you'd rather wire the existing `stackql` binary on your machine directly (no bundle, no Claude Desktop install flow), add an entry to your client's MCP config. For Claude Desktop the file is:

- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`
- Linux: `~/.config/Claude/claude_desktop_config.json`

Add the `stackql` server entry. Adjust `command` to the absolute path of your `stackql` binary if it is not on `PATH`, and replace `/Users/you` with your actual home directory (no variable substitution happens in this file):

```json
{
  "mcpServers": {
    "stackql": {
      "command": "stackql",
      "args": [
        "mcp",
        "--mcp.server.type=stdio",
        "--approot", "/Users/you/.stackql",
        "--mcp.config", "{\"server\": {\"audit\": {\"disabled\": true}}}"
      ]
    }
  }
}
```

All three extra arguments matter:

- `--mcp.server.type=stdio` is required - without it the server starts but does not produce JSON-RPC on stdout.
- `--approot` must point somewhere writable. Claude Desktop launches MCP servers with cwd `/` (read-only on macOS), and stackql's default approot is `<cwd>/.stackql`, so without this flag provider downloads fail.
- The `--mcp.config` audit setting is required for the same reason: the audit sink defaults to a file in the cwd and the server exits if it cannot open it (`failure_mode` defaults to `strict`). Alternatively set `{"server": {"audit": {"file": {"path": "/Users/you/.stackql/stackql-mcp-audit.log"}}}}` to keep auditing with an explicit writable path.

### With cloud provider credentials

The server picks up provider credentials through stackql's normal `--auth` flag. For example, to query AWS with an environment-backed access key and GitHub in no-auth mode:

```json
{
  "mcpServers": {
    "stackql": {
      "command": "stackql",
      "args": [
        "mcp",
        "--mcp.server.type=stdio",
        "--auth={\"aws\":{\"type\":\"aws_signing_v4\",\"credentialsenvvar\":\"AWS_SECRET_ACCESS_KEY\",\"keyID\":\"AWS_ACCESS_KEY_ID\"},\"github\":{\"type\":\"null_auth\"}}"
      ]
    }
  }
}
```

See https://stackql.io/docs for the full provider auth catalogue.

## 4. npx (any stdio MCP client, no install)

The `@stackql/mcp-server` package downloads the signed binary on first run
(sha256-verified against pins baked into the package) and caches it under
`~/.stackql/mcp-server-bin/`:

```json
{
  "mcpServers": {
    "stackql": {
      "command": "npx",
      "args": ["-y", "@stackql/mcp-server"]
    }
  }
}
```

The launcher sets `--approot` and disables the audit sink automatically (the
cwd-safety flags from section 3), so no extra arguments are needed. Pass
`--auth=...` and other stackql flags as additional args.

## 5. uvx / pip (Python)

The `stackql-mcp-server` PyPI package works the same way as the npm wrapper
(pure stdlib, no dependencies); the two share the binary cache at
`~/.stackql/mcp-server-bin/`:

```json
{
  "mcpServers": {
    "stackql": {
      "command": "uvx",
      "args": ["stackql-mcp-server"]
    }
  }
}
```

Or `pip install stackql-mcp-server` and use `stackql-mcp` as the command.

## 6. Docker

```bash
docker run -i --rm stackql/stackql-mcp
```

Runs the MCP server on stdio as a non-root user; amd64 and arm64. As an MCP
client entry:

```json
{
  "mcpServers": {
    "stackql": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "stackql/stackql-mcp"]
    }
  }
}
```

Add `-e` flags before the image name to pass credential environment variables
referenced by your `--auth` config.

## Trust model

What you get with a fresh StackQL `.mcpb` install:

1. The bundled binary is **Apple-notarised** (macOS) or **Authenticode-signed** (Windows). Gatekeeper and SmartScreen validate it at run time. Linux binaries are unsigned by convention.
2. The `.mcpb` envelope is **not currently signed** - the EV code-signing cert that signs the upstream `stackql.exe` is HSM-resident and incompatible with the `mcpb sign` CLI today. The Claude Desktop install dialog will note "unsigned" for the bundle envelope. The binary inside it is still fully signed.
3. The `.mcpb` SHA-256 is **published next to the release asset** and **pinned in the Official MCP Registry** entry. Marketplaces and the `mcp-publisher` flow verify it before install.

If you want envelope-signed bundles before installing, build from source with `make signed VERSION=X.Y.Z` and a self-signed cert, or wait for the upstream `@anthropic-ai/mcpb` CLI to add HSM support so the production EV cert can sign envelopes directly.
