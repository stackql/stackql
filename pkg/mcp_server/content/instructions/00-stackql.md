# Overview

StackQL exposes cloud and SaaS providers as SQL. Objects follow the hierarchy `<provider>.<service>.<resource>`. StackQL providers are versioned interface documents pulled from the provider registry; queries execute as live API calls against the provider, not against stored data.

Session guidance:

- `server_info` reports the `sql_backend` (the dialect: `sqlite3` unless the operator configured `postgres`) and the server `mode` (the write contract: `safe` unless changed). Call it on demand - to diagnose a refused or gated operation, before relying on dialect-specific functions, or when reporting a problem - not as a session-start step, and never before queries.
- Credentials are operator-configured on the server, resolved from the server's process environment at query time; the client never supplies or sees credential values, and no credential step precedes queries.
- When a query fails with a credential resolution error, the error names the provider and the fix: ask the operator to update the configured env file (or set the named env vars), call `reload_credentials` (optionally scoped to that provider), then retry once. The report carries variable names and per-provider status, never values; `changed: false` for the provider after a failure means nothing was updated, so ask the operator rather than reloading or retrying again.
- StackQL abstracts the underlying API surface. Frame answers in terms of providers, resources and SQL - not HTTP calls, endpoints, wire params, pagination or encoding workarounds - except when diagnosing a failure or when the user asks for that level of detail.
