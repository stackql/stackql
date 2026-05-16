
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

To run a read-only server (mutation and lifecycle tools refuse to execute):

```bash

./build/stackql mcp --mcp.server.type=http --mcp.config '{"server": {"transport": "http", "address": "127.0.0.1:9992", "read_only": true} }'

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

## Mutation (INSERT/UPDATE/REPLACE/DELETE).  Refused in read-only mode.
## Tread carefully -- real side effects.
# ./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action run_mutation_query --exec.args '{"sql": "delete from google.compute.networks where project = '"'"'stackql-demo'"'"' and network = '"'"'returning-test-03'"'"' ;"}'

## Lifecycle EXEC operation.  Refused in read-only mode.
# ./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action run_lifecycle_operation --exec.args '{"sql": "EXEC google.compute.instances.start @project = '"'"'mutable-project'"'"', @zone = '"'"'us-central1-a'"'"', @instance = '"'"'demo'"'"';"}'

```


## Canonical agent tools

The server publishes 11 tools.  Each returns both rendered text (for the LLM) and a typed structured payload (for programmatic clients).  Rendering is fixed per tool: a markdown table for uniform multi-row results, a markdown KV block for sparse / single-record / mixed-shape results.

| Tool | Renderer | Description |
|---|---|---|
| `server_info` | KV | Server identity and runtime: stackql version, backing SQL engine, provider registry location, read-only flag.  Call once at session start. |
| `list_providers` | Table | Available cloud/SaaS providers (top of the hierarchy).  No inputs. |
| `list_services` | Table | Services under a provider.  Requires `provider`. |
| `list_resources` | Table | Resources under a `provider`.`service`.  Requires `provider` and `service`. |
| `list_methods` | Table | Access methods (HTTP operations) for a resource.  **Call before writing any query.** Requires `provider`, `service`, `resource`. |
| `describe_resource` | KV | Output fields for a resource's primary read method.  Requires `provider`, `service`, `resource`. |
| `describe_method` | KV | Full I/O contract for one method (always EXTENDED).  Requires `provider`, `service`, `resource`, `method`. |
| `validate_select_query` | KV | Parse and plan a SELECT without executing.  Returns `{valid, errors}`.  SELECT only. |
| `run_select_query` | Table | Execute a SELECT.  Returns `{rows}`.  Reads only. |
| `run_mutation_query` | KV | Execute INSERT/UPDATE/REPLACE/DELETE.  **Real side effects.** Returns `{messages, timestamp}`.  Refused in read-only mode. |
| `run_lifecycle_operation` | KV | Execute a stackql `EXEC` lifecycle operation.  Returns `{messages, timestamp}`.  Refused in read-only mode. |

## Canonical agent prompts

One static prompt is published:

- `write_safe_select` -> guidance for writing safe SELECT queries against stackql resources.  Explains how to use `SHOW METHODS IN <provider>.<service>.<resource>` to discover the best read method and the required `WHERE` parameters.

`EnabledTools` and `EnabledPrompts` on `Config` are independent allowlists.  When omitted or empty everything is published; when populated they restrict the published surface to the named items.  See [the `pkg/mcp_server` README](/pkg/mcp_server/README.md) for details.


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


## Mutations and lifecycle operations have real side effects.  Refused on
## a read-only server.  Response shape is {messages, timestamp}.
$ ./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action run_mutation_query --exec.args '{"sql": "delete from google.compute.networks where project = '"'"'stackql-demo'"'"' and network = '"'"'returning-test-01'"'"' ;"}' 2>/dev/null | jq
{
  "messages": [
    "The operation was despatched successfully"
  ],
  "timestamp": "2026-05-16T10:47:33+10:00 AEST"
}

```
