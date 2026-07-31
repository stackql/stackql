---
name: stackql_scope_discovery
description: Scope discovery strategy for audit-style sweeps - what to enumerate per cloud and which query library entries carry the tested queries.
---
# Scope discovery strategy

How to establish cloud scope before an audit-style sweep. Query shapes are deliberately
not inlined here: fetch them from the query library (`query_library_search` /
`query_library_get`), which is the single maintained home for tested enumeration queries.

## Per cloud

- **AWS** - enumerate enabled regions first and exclude `not-opted-in` regions from every
  subsequent sweep. Library entry: `aws/ec2/regions-enabled`. For storage scope, pair the
  cheap bucket enumeration (`aws/s3/buckets-list`) with per-bucket detail
  (`aws/s3/bucket-detail`) only for the buckets you need.
- **GCP** - descend the organization: list folders and projects per parent
  (`organizations/<org_id>` or `folders/<folder_id>`), recursing into `ACTIVE` folders;
  audit only `ACTIVE` projects. Library entry:
  `google/cloudresourcemanager/projects-by-parent`.
- **Azure** - enumerate subscriptions tenant-wide and audit only `Enabled` ones; when a
  management-group scope is configured, descend it instead. Library entry:
  `azure/subscription/subscriptions-list`.

## Rules

- Fetch templates by id with `query_library_get` and render with params server-side; do
  not hand-author enumeration queries while a library entry exists.
- If an entry is missing for a provider in scope, fall back to the discovery workflow
  (`list_methods` / `describe_method`) per the server instructions.
- Record every scope element skipped (region, project, subscription) and the reason;
  never assert completeness over unswept scope.
