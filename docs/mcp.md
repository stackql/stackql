
## Running the MCP server

If necessary, rebuild stackql with:

```bash
python cicd/python/build.py --build
```

**Note**: before starting an MCP server, remember to export all appropriate auth env vars.

We have a nice debug config for running an MCP server with `vscode`, please see [the `vscode` debug launch config](/.vscode/launch.json) for that.  Otherwise, you can run with stackql (assuming locally built into `./build/stackql`):


```bash

./build/stackql mcp --mcp.server.type=http --mcp.config '{"server": {"transport": "http", "address": "127.0.0.1:9992"} }'

```

The default mode is `safe`: SELECTs proceed, mutations and lifecycle operations require user approval via the [MCP elicitation flow](#server-modes).  An audit log is written to `./stackql_mcp_server_<RFC3339-utc-second>.log` by default (see [Audit Log](#audit-log)).

### Mode examples

Read-only (mutations and lifecycle refused immediately):

```bash

./build/stackql mcp --mcp.server.type=http --mcp.config '{"server": {"transport": "http", "address": "127.0.0.1:9992", "mode": "read_only"} }'

```

Delete-safe (INSERTs/UPDATEs proceed; DELETEs and EXECs need approval):

```bash

./build/stackql mcp --mcp.server.type=http --mcp.config '{"server": {"transport": "http", "address": "127.0.0.1:9992", "mode": "delete_safe"} }'

```

Full access (everything proceeds without approval - use only when the client and operator are trusted):

```bash

./build/stackql mcp --mcp.server.type=http --mcp.config '{"server": {"transport": "http", "address": "127.0.0.1:9992", "mode": "full_access"} }'

```

The legacy `"read_only": true` JSON key is still accepted and is treated as equivalent to `"mode": "read_only"`.  When both are set, `mode` wins.

### Disabling audit

Audit is on by default.  To opt out:

```bash

./build/stackql mcp --mcp.server.type=http --mcp.config '{"server": {"transport": "http", "address": "127.0.0.1:9992", "audit": {"disabled": true}} }'

```


## Using the MCP Client

This is very much a development tool, not currently recommended for production.  That said, it emulates agent actions and allows us to run regression tests.

Build:

```bash
python cicd/python/build.py --build-mcp-client
```

Then, assuming you have a `stackql` MCP server serving streamable HTTP on port `9992`, you can access any endpoint.  The below examples are illustrative of a canonical agent pattern.


```bash

## Server identity and runtime metadata.  Call once at session start.
./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action server_info

## List available providers.
./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action list_providers

## List available services.
## **must** supply <provider>
./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action list_services --exec.args '{"provider": "google"}'

## List available resources.
## **must** supply <provider>, <service>
./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action list_resources --exec.args '{"provider": "google", "service": "compute"}'

## List access methods.  Call before writing any query.
## **must** supply <provider>, <service>, <resource>
./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action list_methods --exec.args '{"provider": "google", "service": "compute", "resource": "networks"}'

## Describe a resource's output fields.
## **must** supply <provider>, <service>, <resource>
./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action describe_resource --exec.args '{"provider": "google", "service": "compute", "resource": "networks"}'

## Describe a single method's I/O contract (always EXTENDED).
## **must** supply <provider>, <service>, <resource>, <method>
./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action describe_method --exec.args '{"provider": "google", "service": "compute", "resource": "networks", "method": "get"}'

## Validate a SELECT (parse + plan) without executing.  SELECT only.
./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action validate_select_query --exec.args '{"sql": "select name from google.compute.networks where project = '"'"'stackql-demo'"'"';"}'

## Run a SELECT.
./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action run_select_query --exec.args '{"sql": "select name from google.compute.networks where project = '"'"'stackql-demo'"'"';"}'

## Mutation (INSERT/UPDATE/REPLACE/DELETE).  Tread carefully -- real side effects.
## Gated by the server mode.  Under the default `safe` mode this call returns
## an error because the bundled client does not advertise elicitation (it
## cannot respond to the server's approval prompt).  Start the server with
## `mode: full_access` to run it without an approval prompt.
# ./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action run_mutation_query --exec.args '{"sql": "delete from google.compute.networks where project = '"'"'stackql-demo'"'"' and network = '"'"'returning-test-03'"'"' ;"}'

## Lifecycle EXEC operation.  Same gating rules as run_mutation_query.
# ./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action run_lifecycle_operation --exec.args '{"sql": "EXEC google.compute.instances.start @project = '"'"'mutable-project'"'"', @zone = '"'"'us-central1-a'"'"', @instance = '"'"'demo'"'"';"}'

```


## Canonical agent tools

The server publishes 11 tools.  Each returns both rendered text (for the LLM) and a typed structured payload (for programmatic clients).  Rendering is fixed per tool: a markdown table for uniform multi-row results, a markdown KV block for sparse / single-record / mixed-shape results.

| Tool | Renderer | Description |
|---|---|---|
| `server_info` | KV | Server identity and runtime: stackql version, backing SQL engine, provider registry location, mode, read-only flag.  Call once at session start. |
| `list_providers` | Table | Available cloud/SaaS providers (top of the hierarchy).  No inputs. |
| `list_services` | Table | Services under a provider.  Requires `provider`. |
| `list_resources` | Table | Resources under a `provider`.`service`.  Requires `provider` and `service`. |
| `list_methods` | Table | Access methods (HTTP operations) for a resource.  **Call before writing any query.** Requires `provider`, `service`, `resource`. |
| `describe_resource` | KV | Output fields for a resource's primary read method.  Requires `provider`, `service`, `resource`. |
| `describe_method` | KV | Full I/O contract for one method (always EXTENDED).  Requires `provider`, `service`, `resource`, `method`. |
| `validate_select_query` | KV | Parse and plan a SELECT without executing.  Returns `{valid, errors}`.  SELECT only. |
| `run_select_query` | Table | Execute a SELECT.  Returns `{rows}`.  Reads only. |
| `run_mutation_query` | KV | Execute INSERT/UPDATE/REPLACE/DELETE.  **Real side effects.** Returns `{messages, timestamp}`.  Gated by the server [mode](#server-modes). |
| `run_lifecycle_operation` | KV | Execute a stackql `EXEC` lifecycle operation.  Returns `{messages, timestamp}`.  Gated by the server [mode](#server-modes). |

## Canonical agent prompts

One static prompt is published:

- `write_safe_select` -> guidance for writing safe SELECT queries against stackql resources.  Explains how to use `SHOW METHODS IN <provider>.<service>.<resource>` to discover the best read method and the required `WHERE` parameters.

`EnabledTools` and `EnabledPrompts` on `Config` are independent allowlists.  When omitted or empty everything is published; when populated they restrict the published surface to the named items.  See [the `pkg/mcp_server` README](/pkg/mcp_server/README.md) for details.


## Server modes

`Config.Server.Mode` picks one of four safety contracts.  All four allow SELECTs and metadata reads; they differ in how they handle mutations and lifecycle operations.

| Mode | SELECT / metadata | INSERT / UPDATE / REPLACE / MERGE / UPSERT | DELETE | EXEC (lifecycle) |
|---|---|---|---|---|
| `read_only` | allow | refuse | refuse | refuse |
| `safe` (default) | allow | needs approval | needs approval | needs approval |
| `delete_safe` | allow | allow | needs approval | needs approval |
| `full_access` | allow | allow | allow | allow |

**Refuse** returns an error immediately.

**Needs approval** uses the MCP elicitation flow:

- If the client advertised elicitation at initialise, the server sends an `elicitation/create` request describing the action (tool name, query class, SQL).  Branch on the user response: `accept` -> proceed; `decline` or `cancel` -> return an error.
- If the client did **not** advertise elicitation, the tool is refused with a message that points the operator at `full_access` mode.

The bundled `stackql_mcp_client` does NOT advertise elicitation, so against a `safe` or `delete_safe` server every mutation/lifecycle call is refused with the no-elicitation message.  This is by design - the bundled client exists for scripting and regression tests, not interactive use.  Elicitation-capable MCP clients (eg Claude Desktop, Cursor) prompt the user normally.

### Breaking change vs PR1

PR1 had a single `read_only: true/false` flag with a default of "no enforcement; mutations proceed."  PR2 replaces that flag with `mode: safe` as the default, which means **mutations now require user approval out of the box.**  Operators running an elicitation-capable client see one approval prompt per mutation.  Operators running automation or the bundled client must explicitly opt into `full_access`.

The legacy `read_only: true` JSON / YAML key still parses for back-compat and is treated as equivalent to `mode: read_only`.  `mode` wins when both are set.


## Audit log

Every tool call writes one JSONL record.  The audit answers "what did the agent do," not "what did the agent see" - result rows from SELECTs are intentionally not recorded.

Recorded per event:

| Field | Notes |
|---|---|
| `timestamp` | RFC3339 start-of-call wall clock |
| `tool` | Tool name (eg `run_select_query`) |
| `mode` | Server mode in effect at call time |
| `decision` | `allow` / `refuse_immediate` / `needs_approval_accepted` / `needs_approval_declined` / `needs_approval_cancelled` / `needs_approval_unavailable` |
| `query_class` | `select` / `mutation_create` / `mutation_delete` / `lifecycle` / `unknown` |
| `sql` | For query tools (`run_select_query`, `run_mutation_query`, `run_lifecycle_operation`, `validate_select_query`) |
| `args` | Hierarchy fields for metadata tools (`list_*`, `describe_*`); SQL + row_limit for query tools |
| `duration_ms` | Wall-clock duration of the gate + handler |
| `error` | Error message if the tool errored or was refused |

### File sink

The only sink shipped in this release is `file`.  One JSON object per line, fsynced after each record.  Lumberjack-style rotation.

```bash
./build/stackql mcp \
  --mcp.server.type=http \
  --mcp.config '{"server": {"transport": "http", "address": "127.0.0.1:9992", "audit": {"file": {"path": "/var/log/stackql-mcp.log", "max_size_mb": 100, "max_backups": 5, "max_age_days": 30}}} }'
```

If `path` is empty the sink picks `stackql_mcp_server_<RFC3339-utc-second>.log` in cwd.  The resolved absolute path is logged to stderr at startup as `sink file: /path/to/file.log`.

### Failure modes

When the sink returns an error, the response behaviour depends on `failure_mode`:

| failure_mode | Effect |
|---|---|
| `strict` (default) | The tool call returns the audit error to the client, even if the underlying tool succeeded.  Intentional: better an ambiguous client than an undetected DELETE. |
| `strict_mutations` | SELECT / metadata reads proceed; mutations and lifecycle ops surface the audit error. |
| `best_effort` | Always log to stderr and proceed. |

### Sequencing

The audit write happens AFTER the tool executes (or is skipped because it was gated out) but BEFORE the response returns to the client.  In strict mode an audit-write failure on a successful DELETE means the row is gone but the client sees an error - by design.

### Breaking change vs PR1

PR1 had no audit subsystem; PR2 enables audit by default.  To preserve PR1 behaviour, pass `"audit": {"disabled": true}` in `mcp.config`.


## Example responses

Responses below show the typed structured payload (what `--exec` prints as JSON).  The MCP `Content` block also carries a rendered text view (markdown table or KV) that the LLM consumes; it is not shown here.

```bash

$ ./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action server_info 2>/dev/null | jq
{
  "version": "0.10.444",
  "commit": "abc1234",
  "build_date": "2026-05-16T10:47:33Z",
  "platform": "linux/amd64",
  "transport": "http",
  "sql_backend": "sqlite3",
  "provider_registry": "https://registry.stackql.app",
  "mode": "safe",
  "is_read_only": false
}


$ ./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action list_providers 2>/dev/null | jq
{
  "rows": [
    { "name": "aws",         "version": "v24.07.00246" },
    { "name": "azure",       "version": "v24.06.00242" },
    { "name": "google",      "version": "v25.11.00355" },
    { "name": "github",      "version": "v25.07.00320" }
  ]
}


$ ./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action list_services --exec.args '{"provider": "google"}' 2>/dev/null | jq
{
  "rows": [
    { "id": "compute:v25.11.00355",     "name": "compute",     "title": "Compute Engine API" },
    { "id": "storage:v25.11.00355",     "name": "storage",     "title": "Cloud Storage API" }
  ]
}


$ ./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action list_resources --exec.args '{"provider": "google", "service": "compute"}' 2>/dev/null | jq
{
  "rows": [
    { "name": "instances" },
    { "name": "networks" },
    { "name": "subnetworks" }
  ]
}


$ ./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action list_methods --exec.args '{"provider": "google", "service": "compute", "resource": "networks"}' 2>/dev/null | jq
{
  "rows": [
    { "MethodName": "get",      "RequiredParams": "network, project",  "SQLVerb": "SELECT" },
    { "MethodName": "list",     "RequiredParams": "project",            "SQLVerb": "SELECT" },
    { "MethodName": "insert",   "RequiredParams": "project",            "SQLVerb": "INSERT" },
    { "MethodName": "delete",   "RequiredParams": "network, project",  "SQLVerb": "DELETE" }
  ]
}


$ ./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action describe_resource --exec.args '{"provider": "google", "service": "compute", "resource": "networks"}' 2>/dev/null | jq
{
  "rows": [
    { "name": "id",          "type": "string" },
    { "name": "name",        "type": "string" },
    { "name": "description", "type": "string" },
    { "name": "selfLink",    "type": "string" }
  ]
}


$ ./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action describe_method --exec.args '{"provider": "google", "service": "compute", "resource": "networks", "method": "get"}' 2>/dev/null | jq
{
  "rows": [
    { "name": "project", "type": "string", "param_type": "input_required", "shape": "string",  "description": "Project ID for this request." },
    { "name": "network", "type": "string", "param_type": "input_required", "shape": "string",  "description": "Name of the network to return." },
    { "name": "id",      "type": "string", "param_type": "output",         "shape": "string",  "description": "[Output Only] The unique identifier..." }
  ]
}


$ ./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action validate_select_query --exec.args '{"sql": "select name from google.compute.networks where project = '"'"'stackql-demo'"'"';"}' 2>/dev/null | jq
{
  "valid": true
}


$ ./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action run_select_query --exec.args '{"sql": "select name from google.compute.networks where project = '"'"'stackql-demo'"'"';"}' 2>/dev/null | jq
{
  "rows": [
    { "name": "pathfinders-test-01" },
    { "name": "pathfinders-test-02" },
    { "name": "returning-test-01" },
    { "name": "returning-test-03" }
  ]
}


## Mutations and lifecycle operations have real side effects.  Their behaviour
## depends on the server [mode](#server-modes):
##   - read_only         -> refused immediately
##   - safe              -> elicits user approval; refused if the client does
##                          not advertise elicitation (the bundled client does
##                          NOT, so this returns an error)
##   - delete_safe       -> mutation_create proceeds; DELETE / EXEC elicit
##   - full_access       -> proceeds without prompting
##
## Example below assumes a full_access server.  Response shape is
## {messages, timestamp}.
$ ./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action run_mutation_query --exec.args '{"sql": "delete from google.compute.networks where project = '"'"'stackql-demo'"'"' and network = '"'"'returning-test-01'"'"' ;"}' 2>/dev/null | jq
{
  "messages": [
    "The operation was despatched successfully"
  ],
  "timestamp": "2026-05-16T10:47:33+10:00 AEST"
}


## Example audit log line (`mcp-audit.log`) for the call above.
{"timestamp":"2026-05-16T00:47:33Z","tool":"run_mutation_query","mode":"full_access","decision":"allow","query_class":"mutation_delete","sql":"delete from google.compute.networks where project = 'stackql-demo' and network = 'returning-test-01' ;","args":{"sql":"delete from google.compute.networks where project = 'stackql-demo' and network = 'returning-test-01' ;","row_limit":0},"duration_ms":42}

```
