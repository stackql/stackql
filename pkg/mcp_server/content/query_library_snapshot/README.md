# Query library snapshot - NOT the master

This directory is a vendored, point-in-time copy of a handful of query library
entries, compiled into the binary via `go:embed`. It is NOT where the query
library is mastered and it is NOT the ground truth.

- **Mastered in**: the `stackql/query-library.stackql.io` repository, under
  `query-library/` (published to `https://stackql.io/docs/query-library`,
  served from its own site behind a proxy rewrite). Author, fix and extend
  entries there, never here.
- **What this copy is for**: the last tier of the fetch fallback chain in
  `pkg/mcp_server/query_library.go` - served only when the server runs with
  `STACKQL_QUERY_LIBRARY_OFFLINE=true` (air-gapped installs) or when both the
  primary site and the raw GitHub fallback are unreachable. Responses served
  from here carry `source_tier: snapshot` and an `embedded://` citation, and
  are flagged `stale` when reached via fetch failure. The robot and unit
  suites also exercise this tier because it needs no network.
- **Refreshing**: copy the generated artifacts (`manifest.json`, `index.json`,
  `queries/**/*.json`) from the query-library.stackql.io repo's
  `static/docs/query-library/` output. Keep the set small - this ships in every binary; it is a survival
  ration, not a mirror. Preserve the `embedded-snapshot-*` build_id convention
  so snapshot serves are distinguishable in citations and telemetry.
