# Discovery workflow

Before composing SQL from scratch, check `query_library_search` for a vetted template: `query_library_get` validates the params and renders the SQL, and a rendered template needs no further discovery. For everything else, drill down in order, skipping any level already known, and stop as soon as you hold the method's contract:

1. `list_providers` - providers installed on the server.
2. `list_services` - services within the provider.
3. `list_resources` - resources ("things" you query and/or operate on) within a service.
4. `list_methods` - the resource's methods with their mapped SQL verb (`SELECT`, `INSERT`, `UPDATE`, `DELETE`, `EXEC`) and required params. Pick the method whose verb matches the intent and whose required params you can supply: those params become the mandatory exact-match `WHERE` predicates, and they are how the server routes the statement to that method.
5. `describe_method` - the full I/O contract of the chosen method: every input with its `param_type` (`input_required` / `input_optional`) and every `output` field, which are the columns a `SELECT` (or `RETURNING`) can reference. Call it once per method before the first query against it. There is no resource-level field list: each method returns its own shape, so column names always come from the method the query will route to.

Then run the statement with `run_select_query`, `run_mutation_query` or `run_lifecycle_operation`.

If a provider is not installed, check availability with `list_registry` then install with `pull_provider`. The registry in use is reported in the `provider_registry` field of `server_info`; the public registry is the default, but the server may be configured to use the `dev` registry or a local registry.

Never guess service, resource, method or column names within a provider. Use the discovery tools to get service, resource and method names and the I/O contract before the first query, so it runs right the first time.
