# Error handling

How errors surface:

- Provider HTTP errors are reported as messages (eg `http response status code: 404, response body is nil`), not SQL exceptions. A query can partially succeed: in a multi-member acquisition (eg per-identifier detail reads), members that 404 contribute zero rows and the message is reported alongside the rows that resolved. A hard failure in an `IN` fan-out, by contrast, fails the whole query (see routing).
- An empty result set accompanied by a 404 message is the normal outcome of querying an object that may or may not exist. Treat it as "not found" - a valid answer - not as a failure to be retried.
- Parser and planner errors (unknown table, `no such column`, syntax) are returned immediately; no provider call is made.

Automatic retry, owned by the engine (per-operation overrides live in the provider spec):

- GET and HEAD requests that return 408, 429, 502, 503 or 504 - or fail with a transport-level network error - are retried automatically with exponential backoff (default 3 attempts, 500ms initial delay, doubling, capped at 10s). Do not layer your own retry loop on top of these.
- Everything else is a single attempt: 400, 401, 403, 404, 409 and 500 are not auto-retried, and mutations are never auto-retried regardless of status.

Recovery by class:

- 400 or a provider validation error -> the request shape is wrong. Re-check `describe_method` for required params and exact field names; never resubmit unmodified.
- 401 -> credentials invalid or expired. Call `reload_credentials` and retry once; if it persists, report the named env vars to the operator.
- 403 -> authenticated but not permitted. Retry cannot fix this; record the check as "could not assess" or ask the operator for the missing permission.
- 429 surviving auto-retry -> reduce fan-out breadth and query frequency before trying again.
- 5xx -> provider-side trouble. 502/503/504 arrive only after auto-retry is exhausted; 500 is never auto-retried. Either way back off and report, do not hammer.
- `no such column` -> call `describe_resource` and use the exact field names returned. Do not guess variants.
- Empty result from a list operation that should plainly return rows -> suspect endpoint routing. Retry with the provider's default region or scope before concluding the result is genuinely empty.
- Mutation rejected by policy -> server-mode enforcement, not a syntax error. Never rewrite the query to evade the gate; report the server mode to the user.
