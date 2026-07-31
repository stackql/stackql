# Overview

StackQL exposes cloud and SaaS providers as SQL. Objects follow the hierarchy `<provider>.<service>.<resource>`. StackQL providers are versioned interface documents pulled from the provider registry; queries execute as live API calls against the provider, not against stored data.

Session guidance:

- Call `server_info` once at session start to learn the `sql_backend` and the server `mode`.
- Credentials are operator-configured on the server, resolved from the server's process environment; the client never supplies or sees credential values.
- If a provider reports unresolved credentials, ask the operator to set the named env vars (or update the configured env file) and call `reload_credentials`; it re-sources the file and reports per-provider status, names only, never values.
- StackQL abstracts the underlying API surface. Frame answers in terms of providers, resources and SQL - not HTTP calls, endpoints, wire params, pagination or encoding workarounds - except when diagnosing a failure or when the user asks for that level of detail.
