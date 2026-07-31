# Pushdown and pagination

The engine handles these mechanics automatically. They shape how you write queries and interpret results.

Pagination:

- Paginated list APIs are traversed to exhaustion automatically. The strategy is provider-configured and includes: request/response tokens (`nextPageToken` style), page number with a page-count terminator, OData `@odata.nextLink`, HTTP `Link` headers with `rel="next"`, and GraphQL cursor strategies (Relay `after` cursor, keyset, offset, `pageInfo.hasNextPage`).
- Never reference page tokens, cursors or page-number params in SQL. They are not columns; the engine owns them end to end.
- Traversal is bounded: when a response lacks its configured terminator, the engine stops after one page rather than looping. If a result count looks implausibly low for the resource, consider truncated pagination and say so rather than asserting completeness.
- An unfiltered SELECT over a large collection can mean many sequential wire requests. Bound the work with input params in the `WHERE` clause and `LIMIT` where the intent allows.

Pushdown:

- Write plain SQL and let the planner optimise. Where the provider dialect supports it, `WHERE` predicates on response fields (input params always travel as request inputs per the dialect rules), `LIMIT`, `OFFSET`, `ORDER BY`, the projection and `count(*)` are rendered into the provider's own query language on the wire - eg OData `$filter` (`=` becomes `eq`, `LIKE 'A%'` becomes `startswith`), `$top`, `$skip`, `$orderby`, `$select`, `$count`; GraphQL `LIMIT n` becomes `first: n`.
- Never hand-assemble provider query options (`$filter=...` and kin) as param values when plain SQL expresses the same thing; the rendered form is the engine's job.
- Pushdown is an optimisation only: client-side filtering and projection remain authoritative on the returned rows, and pushdown is suppressed where it would change results (eg `LIMIT` is not pushed under a `GROUP BY` that changes grain).
- For providers without a pushdown dialect, server-side filtering is still available where the method exposes optional query params (`input_optional` rows in `describe_method`, eg `filter`, `query`, `maxResults`): supply them in the `WHERE` clause as exact-equality predicates and the value passes to the provider verbatim, in the provider's own filter syntax.
- `LIMIT` is always honoured in the final result set; whether it also reduces wire traffic depends on the provider's pushdown and pagination support.
