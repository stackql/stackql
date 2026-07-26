# Discovery workflow

Drill down in order - provider -> service -> resource -> methods - then describe before querying. The following discovery tools are available:

- `list_providers` - providers installed on the server
- `list_services` - available services within the provider
- `list_resources` - available resources ("things" you query and/or operate on)
- `list_methods` - available methods, their mapped SQL verbs and required params
- `describe_resource` - for the output fields of the primary read method
- `describe_method` - the full I/O contract including optional input params

If a provider is not installed, check availability with `list_registry` then install with `pull_provider`. The registry in use is reported in the `provider_registry` field of `server_info`; the public registry is the default, but the server may be configured to use the `dev` registry or a local registry.

Never guess service, resource, table or column names within a provider. Use the discovery tools available to get service, resource, field names and I/O contracts.