# Setup StackQL MCP Server (GitHub Action)

Installs the signed [StackQL](https://stackql.io) binary (sha256-verified
against the release checksums) and emits an `mcpServers` JSON config that
plugs straight into MCP-capable actions like
[anthropics/claude-code-action](https://github.com/anthropics/claude-code-action) -
giving CI agents live SQL query (and optionally provisioning) access to AWS,
Azure, Google, GitHub, Databricks, and 40+ other providers.

Defaults to `read_only` server mode - the safe default for agentic CI.

## Inputs

| Input | Default | Description |
|---|---|---|
| `version` | `latest` | stackql release version (`X.Y.Z`) or `latest` |
| `mode` | `read_only` | MCP server mode: `read_only`, `safe`, `delete_safe`, `full_access` |
| `auth` | (none) | stackql `--auth` JSON for provider credentials |
| `bundle-path` | (none) | install from a local `.mcpb` instead of downloading (testing) |

## Outputs

| Output | Description |
|---|---|
| `binary-path` | absolute path to the installed stackql binary |
| `mcp-config` | `mcpServers` JSON for `claude-code-action`'s `mcp_config` input |

Also exports `STACKQL_MCP_BIN` to the job env (the `@stackql/mcp-server` npm
and `stackql-mcp-server` PyPI wrappers detect it and skip their own download)
and adds the install dir to `PATH`.

## Example: agentic cloud audit on a schedule

```yaml
name: nightly-cloud-audit
on:
  schedule:
    - cron: "0 6 * * *"

jobs:
  audit:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      issues: write
    steps:
      - id: stackql
        uses: stackql/setup-stackql-mcp@v1
        with:
          mode: read_only
          auth: '{"aws":{"type":"aws_signing_v4","credentialsenvvar":"AWS_SECRET_ACCESS_KEY","keyID":"AWS_ACCESS_KEY_ID"}}'

      - uses: anthropics/claude-code-action@v1
        env:
          AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
          AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
        with:
          anthropic_api_key: ${{ secrets.ANTHROPIC_API_KEY }}
          mcp_config: ${{ steps.stackql.outputs.mcp-config }}
          allowed_tools: "mcp__stackql__*"
          prompt: |
            Using the stackql tools, audit our AWS account: list S3 buckets
            without encryption, security groups open to 0.0.0.0/0, and any
            unattached EBS volumes. File a GitHub issue summarising findings
            with the SQL you used as evidence.
```

## Example: use the binary directly

```yaml
      - id: stackql
        uses: stackql/setup-stackql-mcp@v1
      - run: stackql exec "SHOW PROVIDERS"
```

## Notes

- Pin `version` (and this action's tag) for reproducible runs; the registry
  entry `io.github.stackql/stackql-mcp` attests the per-version sha256s.
- `read_only` mode means the agent cannot mutate cloud resources regardless
  of prompt injection; raise the mode deliberately, never by default.
