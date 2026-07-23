---
name: stackql_sql_dialect
description: Notes on the StackQL SQL dialect for provider-backed queries.
---
# StackQL SQL dialect notes

StackQL parses a PostgreSQL-flavoured dialect and pushes queries down to provider APIs.

- Projections, aliases, functions and aggregates run locally in the embedded SQL engine; WHERE attributes that map to provider parameters are pushed into the HTTP call and support exact matches only.
- Required WHERE attributes for a resource come from `SHOW METHODS IN <provider>.<service>.<resource>`; a missing required attribute fails the query rather than widening it.
- `INSERT`/`UPDATE`/`REPLACE`/`DELETE` mutate the provider and are gated by the server mode.
- Hierarchical identifiers containing slashes may need the slash encoded as `%2F` when the route has consecutive path parameters.
