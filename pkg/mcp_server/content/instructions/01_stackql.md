StackQL exposes cloud and SaaS providers as SQL. Objects follow the hierarchy `<provider>.<service>.<resource>`.

Session guidance:

- Call `server_info` once at session start to learn the server mode and read-only flag.
- Discover with `list_providers`, `list_services` and `list_resources`, then call `list_methods` (or `describe_method`) before writing any query: it names the access method for the SQL "select" verb and the **required** WHERE clause attributes.
- WHERE clauses that map to provider parameters support exact matches only; wildcards and inequalities are not pushed down.
- If a parameter value embeds a slash (eg `refs/tags/v1`) and the route has consecutive path parameters, URL-encode the slash as `%2F`.
- Prefer `validate_select_query` before `run_select_query` for untested SQL.
