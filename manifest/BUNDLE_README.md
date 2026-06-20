# StackQL MCP Server

SQL-native query and provisioning engine for cloud and SaaS infrastructure,
served over the Model Context Protocol. StackQL exposes 40+ cloud and SaaS
providers (AWS, Azure, Google, GitHub, Databricks, and more) as SQL, so an MCP
client can discover providers, explore schemas, and run live queries and
provisioning operations.

This bundle contains the signed `stackql` binary launched as
`stackql mcp --mcp.server.type=stdio`. No separate StackQL installation is
required.

## Cloud provider credentials and configuration

Provider credentials are supplied through StackQL's `--auth` flag (JSON). By
default this bundle runs with no credentials, which is sufficient for providers
that expose public data without auth (for example the `github` provider in
`null_auth` mode). To query a provider that requires credentials, configure
`--auth` - for example, AWS with environment-backed keys and GitHub in no-auth
mode:

```json
{
  "aws": {
    "type": "aws_signing_v4",
    "credentialsenvvar": "AWS_SECRET_ACCESS_KEY",
    "keyID": "AWS_ACCESS_KEY_ID"
  },
  "github": { "type": "null_auth" }
}
```

Credentials are read locally (from environment variables or files you nominate)
and used only to sign requests to the relevant provider's API. They are never
transmitted anywhere else. See https://stackql.io/docs for the full provider
auth catalogue.

## Data and configuration root

StackQL stores its provider registry cache and working data under an
application root ("approot"). This bundle sets it to `${HOME}/.stackql`. Pulled
provider definitions and cache files are written there. The audit sink is
disabled by default in this bundle; query data is not written to disk unless you
enable it.

## Available tools

- `server_info` - identity and runtime of the connected server
- `list_providers` - list available providers
- `pull_provider` - install a provider into the local registry
- `list_registry` - list providers available in the StackQL registry
- `list_services` - list services within a provider
- `list_resources` - list resources within a service
- `list_methods` - list methods for a resource
- `describe_resource` - describe a resource's fields
- `describe_method` - describe a method's parameters
- `validate_select_query` - validate a SELECT query without running it
- `run_select_query` - run a SELECT query against a provider
- `run_mutation_query` - run an INSERT/UPDATE/DELETE (provisioning) query
- `run_lifecycle_operation` - run a resource lifecycle operation

Tool availability depends on the server mode (`read_only`, `safe`,
`delete_safe`, `full_access`); the default is read-oriented and safe for agents.

## Privacy

See https://stackql.io/privacy for how cloud credentials and query data are
handled.

## Links

- Documentation: https://stackql.io/docs
- Source: https://github.com/stackql/stackql
- Official MCP Registry: `io.github.stackql/stackql-mcp`
- License: MIT (see LICENSE in this bundle)
