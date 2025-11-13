
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


## Using the MCP Client

This is very much a development tool, not currently recommended for production.

Build:

```bash
python cicd/python/build.py --build-mcp-client
```

Then, assuming you have a `stackql` MCP server serving streamable HTTP on port `9992`, you ca access any edpoint.  The below examples are somewhat illustrative of a canonical agent pattern for agent behaviour.


```bash

## List available providers.
./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action      list_providers

## List available services.  
## **must** supply <provider>
./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action      list_services --exec.args '{"provider": "google"}'

## List available resources.  
## **must** supply <provider>, <service>
./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action      list_resources --exec.args '{"provider": "google", "service": "compute"}'

## List access methods.  
## **must** supply <provider>, <service>, <resource>
./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action      list_methods --exec.args '{"provider": "google", "service": "compute", "resource": "networks"}'

## Describe published relation
## **must** supply <provider>, <service>, <resource>
./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action      meta_describe_table --exec.args '{"provider": "google", "service": "compute", "resource": "networks"}'

## Validate query AOT.  Only works for SELECT at this stage.
./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action validate_query_json_v2      --exec.args '{"sql": "select name from google.compute.networks where project = '"'"'stackql-demo'"'"';"}'

## Run query
./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action query_json_v2      --exec.args '{"sql": "select name from google.compute.networks where project = '"'"'stackql-demo'"'"';"}'

## Exec query pattern; for non-read operations
## Tread carefully!!!
## These are almost always mutations
# /build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action exec_query_json_v2      --exec.args '{"sql": "delete from google.compute.networks where project = '"'"'<my-bucket-name>'"'"';"}'

```
