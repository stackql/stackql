# Dialect rules

The working SQL dialect is reported in the `sql_backend` field of `server_info`: `sqlite3` (the default) or `postgres`. Statement-level discovery commands (`SHOW`, `DESCRIBE`, `REGISTRY LIST`, `REGISTRY PULL`) exist in the dialect, but prefer the dedicated discovery tools in this context. Validate non-trivial queries with `validate_select_query` before executing, particularly wide fan-out queries.

## Request inputs in the WHERE clause

- Required params (shown in `RequiredParams` from `list_methods`) must appear in the `WHERE` clause as equality predicates. They are request inputs, which are normally path parameters or server variables - but can include required query, header or body parameters - not post-filters. Omitting one fails the query; it does not widen the result.
- Optional input params can also be supplied in the `WHERE` clause and are passed to the provider request.
- Predicates that map to input params are exact-match equality only: no wildcards, `LIKE`, or inequalities. If a param value with an embedded slash fails, URL-encode the slash as `%2F` and retry.
- Fields in the `describe_resource` response can be referenced in the column list or the `WHERE` clause; projection and filtering on these happen locally after the provider response, unlike params, which shape the request itself.
- Input params containing hyphens must be double-quoted: `WHERE "max-buckets" = 1000`.

## Lexical rules

- String literals use single quotes. Identifiers use double quotes.
- Aggregate aliases do not resolve in `ORDER BY`. Use `ORDER BY COUNT(*)`, not the alias name.

## Statement support

StackQL is largely ANSI SQL compliant. Supported:

- CTEs (`WITH ... AS`), window functions (`OVER`, `PARTITION BY`, frames), derived tables (`FROM (SELECT ...)`), views and materialized views, `UNION` / `UNION ALL`, `GROUP BY` / `HAVING`, and `LIKE` / JSON and string functions on response fields.

Not supported - assume a construct not listed as supported does not work until `validate_select_query` proves otherwise:

- Recursive CTEs
- Transaction control (`BEGIN` / `COMMIT` / `ROLLBACK`)
- Set operations other than `UNION` / `UNION ALL` (no `EXCEPT`, no `INTERSECT`)
- `RIGHT`, `FULL` or `CROSS` joins
- Subqueries in the `WHERE` clause

Rewrites for the gaps: `WHERE x IN (SELECT ...)` -> inner join on a derived table; `EXCEPT` -> `LEFT OUTER JOIN ... WHERE rhs.col IS NULL`.

## Mutation shapes

Always run `describe_method` before a mutation or lifecycle op. It returns the full I/O contract: every input field with its exact column name, `param_type` (`input_required` / `input_optional` / `output`), and a `shape` carrying the nested JSON schema for object and array inputs. Use the reported names verbatim: body field names are normally the provider's raw field names; a minority of methods (eg the generic `aws.cloud_control` surface) prefix body fields with `data__` - the `describe_method` output already reflects whichever applies, so never add or strip a prefix yourself. Whether an input travels in the path, query, header or body is deliberately not exposed; every input is just a column.

- `INSERT INTO <provider>.<service>.<resource> (param, Field) SELECT 'p', 'v'` (or `VALUES (...)`). `INSERT ... SELECT` from a scalar row or another query is supported, as is `RETURNING` (functions allowed in the returning list) where the provider response carries the resource.
- `UPDATE <provider>.<service>.<resource> SET Field = 'v', param = 'p' WHERE <required params>` - `SET` carries body fields and routing params; `WHERE` carries the identifying params.
- `REPLACE` (maps to HTTP PUT) has the same statement shape as `UPDATE` (which typically maps to PATCH).
- `DELETE FROM <provider>.<service>.<resource> WHERE <required params>`.
- JSON-valued inputs are passed as single-quoted JSON strings: `Protocols = '[ "SFTP" ]'`.
- The `Field` names in the examples above are placeholders; substitute the names `describe_method` reports.
- `RETURNING` projects fields of the method's response: exactly the rows `describe_method` reports with `param_type` = `output`. A method with an empty response (eg HTTP 204) has no output rows and nothing to return.

## Functions

The available function surface is determined by the `sql_backend`.

### sqlite3 (default)

There are some additional StackQL specific functions or extensions to existing functions in `sqlite3` dialect only, exposed to the stackql engine:

- `json_equal(json1, json2)` - semantic JSON equality; key order and formatting insensitive; returns `1` or `0`
- `aws_policy_equal(policy1, policy2)` - semantic comparison of AWS IAM policy documents; treats `Action`, `Resource`, `Principal`, `Tags` and similar arrays as unordered sets, service names in ARNs are case-insensitive; also compares two top-level JSON arrays (e.g. raw tags arrays) as unordered sets; returns `1` or `0`
- `regexp_like(string, pattern)` - returns `1` if the pattern matches, else `0`
- `regexp_replace(string, pattern, replacement)` - replaces all matches of the pattern
- `regexp_substr(string, pattern)` - returns the first match, or `NULL` if no match
- `split_part(string, delimiter, index)` - Postgres-style string split; `index` is 1-based

Notes on the `regexp_*` functions: the engine is a minimal regex implementation (tiny-regex-c), not PCRE. Supported syntax: `.` `^` `$` `*` `+` `?`, character classes `[abc]` and ranges `[a-z]`, and `\s` `\S` `\w` `\W` `\d` `\D`. Alternation (`|`), capture groups, and `{n,m}` quantifiers are not supported.

Additionally in the `sqlite3` dialect:

- SQLite math functions are enabled (they are not present in stock embedded sqlite Go distributions): `sqrt`, `pow`, `mod`, `ceil`, `floor`, `exp`, `ln`, `log10`, `pi`, `degrees`, `radians`, trigonometric functions, etc.
- SQLite JSON1 scalar and aggregate functions are available: `json_extract`, `json_array`, `json_object`, `json_patch`, `json_type`, `json_valid`, `json_array_length`, `json_group_array`, `json_group_object`, etc.

Constructs not supported on `sqlite3`, even where the underlying engine supports them (the StackQL parser and planner sit in front of the storage backend):

- `CREATE VIRTUAL TABLE ...` - not in the parser grammar; FTS3/FTS4 and R*Tree modules are compiled into the backend but cannot be created or used
- The `MATCH` operator - parser syntax error; full-text search query syntax is unavailable (use `regexp_like` / `regexp_substr` for pattern matching instead)
- Table-valued functions in `FROM` (`json_each(...)`, `json_tree(...)`) - the statement parses but the planner rejects it (`cannot process cartesian join select just yet`); use `json_extract` with explicit paths instead

### postgres

The `postgres` backend delegates function evaluation to the connected PostgreSQL server: with one exception below, function calls pass through the planner verbatim, so the available surface is what that server natively provides (`string_agg`, `strpos`, `split_part`, `regexp_replace` with full POSIX regex, `to_char`, etc.). Differences from `sqlite3`:

- The StackQL-specific functions are registered only in the embedded sqlite engine and do not exist on postgres: `json_equal`, `aws_policy_equal`, `regexp_like`, `regexp_substr`. SQLite built-ins are likewise absent: use `pg_typeof` for `typeof`, `strpos` for `instr`, `string_agg` for `group_concat`, `to_timestamp` for `datetime(..., 'unixepoch')`.
- One automatic translation exists in the planner: `json_extract(expr, '$.a.b')` is rewritten to `json_extract_path_text(expr, 'a', 'b')`. Only the two-argument form with a literal dot path translates; array indices (`$.a[0]`) do not.
- Reliability caution: the black-box regression suite currently skips most function-bearing scenarios on the postgres backend (including `json_extract`, `split_part` and `group_concat` scenarios). Prefer simple projections on postgres; when a function is needed, test it on a single-row bounded query before building on it.

Behavioral differences on postgres (engine semantics, not bugs):

- Column aliases cannot be referenced in the `WHERE` clause (sqlite3 tolerates this).
- Every non-aggregated select column must appear in `GROUP BY`; `SELECT count(*), col` without grouping is rejected (sqlite3 tolerates it).
- Mixed-case column identifiers are case-sensitive: double-quote them exactly as discovery reports them.
