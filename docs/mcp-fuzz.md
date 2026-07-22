# MCP security fuzzing

StackQL exposes MCP over streamable HTTP (see [docs/mcp.md](mcp.md)). This repo includes a lightweight fuzz fixture and script that exercise the MCP tool surface without cloud credentials.

## Local run

```bash
bash scripts/fuzz-mcp-surface.sh
```

Reports land in `./fuzz-output/`. Override behavior with environment variables:

| Variable | Default | Purpose |
| -------- | ------- | ------- |
| `MCP_FUZZER_IMAGE` | `princekrroshan01/mcp-fuzzer:v0.4.0` | Docker image tag |
| `MCP_FUZZ_RUNS` | `3` | Fuzz iterations per tool |
| `MCP_FUZZ_TIMEOUT` | `30` | Per-request timeout (seconds) |
| `MCP_FUZZ_PORT` | `19992` | Fixture listen port |
| `MCP_FUZZ_OUTPUT` | `./fuzz-output` | Host directory mounted at `/output` |

The fixture (`scripts/fuzz_mcp_fixture/`) boots `pkg/mcp_server` over HTTP in `read_only` mode with representative tool responses so `mcp-fuzzer` can list and call the canonical StackQL MCP tools (`server_info`, `list_providers`, query tools, etc.).

## CI

The `mcp-fuzz` workflow (`.github/workflows/mcp-fuzz.yml`) runs on pull requests and pushes to `main` / `version*`, uploading `fuzz-output/` as an artifact with a smaller run budget (`MCP_FUZZ_RUNS=2`).

## What gets exercised

- Streamable HTTP transport in `pkg/mcp_server/server.go`
- Tool registration, rendering, and mode gating (`read_only` fixture)
- MCP query-tool argument handling (`validate_select_query`, `run_select_query`, hierarchy tools)

This is a smoke-level pass on the MCP server package, not a substitute for robot tests against a full `stackql mcp` deployment with live provider mocks.
