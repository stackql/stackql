---
name: stackql-iac
description: 'Converge, preview, apply, destroy, recover and inspect infrastructure deployments with the native IaC SQL verbs on the stackql_preview.iac relations (run, plan, ledger, recover and blueprints such as aws_vpc_subnet), from stackql shell or exec or through the stackql MCP server. Load it when asked to deploy or tear down resources from a desired-state resource specification or blueprint, preview a change set before applying it, read deployment identities and phases from the IaC ledger, resolve pending or outstanding ledger entries, apply a document_merge provider overlay to a deployment, or decide which MCP tool and server mode allow an IaC statement.'
compatibility: 'Requires stackql started with --preview ''{"unstable":true}'' on stackql mcp and on stackql shell or exec; without it the stackql_preview.iac relations do not exist. Converge, preview and destroy also need provider credentials and a provider registry that resolves the resource specification.'
license: MIT
---

# stackql IaC SQL verbs

stackql exposes Omni's desired-state engine as SQL relations under `stackql_preview.iac`. A
named collection of resources is converged, previewed, destroyed and recovered with ordinary SQL
verbs, and its state is kept in a local ledger. Without the `--preview '{"unstable":true}'` flag
these relations are not found (`could not locate table 'ledger'`, `provider relations must be
planned through omnisdk`).

See `references/walkthrough.md` for end-to-end AWS examples.

## Relations and verbs

| Statement | What it does |
|---|---|
| `INSERT INTO stackql_preview.iac.run` | Converges a collection to the desired state (creates, updates, no-ops). |
| `DELETE FROM stackql_preview.iac.run` | Destroys the collection's resources, using identities and dependencies from the ledger. Dependents go first. |
| `SELECT ... FROM stackql_preview.iac.plan` | Previews: one row per resource with `resource_key`, `action`, `reason`, `depends_on`, `change_set`. Reads the ledger and provider state; sends no mutations and writes no ledger or journal files. |
| `INSERT INTO stackql_preview.iac.plan` | Applies a saved `change_set` from a preview. |
| `UPDATE stackql_preview.iac.recover` | Records an operator-verified outcome for a `pending` ledger entry. Local ledger only; no provider calls. |
| `SELECT ... FROM stackql_preview.iac.ledger` | Reads resource identities and phases. No locks, journals or provider calls. |
| `INSERT INTO stackql_preview.iac.run (blueprint, ...)` | Converges a built-in blueprint such as `aws_vpc_subnet`. |

Any other verb is refused, for example `SELECT` from `run`, `recover` or a blueprint (the error
names the statement to use instead). Discovery is read-only and needs no collection:

```sql
SHOW RESOURCES IN stackql_preview.iac;            -- run, recover, ledger, plan, then blueprints
SHOW METHODS IN stackql_preview.iac.run;          -- run (INSERT), destroy (DELETE)
SHOW METHODS IN stackql_preview.iac.plan;         -- preview (SELECT), apply (INSERT)
DESCRIBE stackql_preview.iac.plan;                -- the five plan columns
DESCRIBE EXTENDED stackql_preview.iac.aws_vpc_subnet;  -- blueprint inputs
```

`DESCRIBE` on `run`, `ledger` and `recover` answers "takes a specification rather than columns".

## Statement shape rules

- INSERT needs a column list and exactly one `VALUES` row.
- Every value is a string or numeric literal (expressions such as `concat(...)` are refused). JSON
  goes in a SQL string literal. Session variables (`SET c = 'x';` then `$c`) are resolved to
  literals first and are accepted.
- WHERE accepts only `column = literal` predicates joined by `AND`. `LIKE`, `>`, `OR`, `ORDER BY`
  and similar are refused.
- Always use the fully qualified name `stackql_preview.iac.<relation>`.
- With `RETURNING *` the statement returns result rows. Without it, converge and destroy still run
  to completion, return no rows, and fail if any resource failed or is outstanding.

## Parameters

| Parameter | Used by | Rule |
|---|---|---|
| `collection` | every IaC statement | Required. It is the ledger key prefix, the lease scope and how later runs address the same resources. |
| `resources` | `run` INSERT, `plan` SELECT | Inline JSON resource specification. The only source `plan` SELECT accepts. |
| `resources_file` | `run` INSERT | Path to a JSON specification file, read by the stackql process. |
| `blueprint` | `run` INSERT | Blueprint handle, `aws_vpc_subnet` or `aws-vpc-subnet`. Its inputs are further columns. |
| `inputs_file` | `run` INSERT, `plan` | Path to a file holding one JSON object of binding inputs. |
| `inputs` | `run` INSERT, `plan` | The same as one inline JSON object. Supply `inputs` or `inputs_file`, not both. |
| `state` | every IaC statement | Optional directory name, relative and inside the IaC state root. Default: the root itself. |
| `region` and any other column | `run`, `plan` | Not consumed by stackql: passed to Omni as the run's execution scope (provider parameters). Rejected by `ledger` SELECT and `recover` UPDATE. |
| `run_id` | generated | stackql stamps each execution with a unique `<UTC timestamp>-<random>` id. Do not supply it. |
| `change_set` | `plan` INSERT | Required. The `change_set` string from a preview row. |
| `keys` | `run` DELETE | Optional. Nonempty JSON array of resource keys for a partial destroy, for example `'["subnet"]'`. |
| `resource_key` | `ledger`, `recover` | Optional filter on `ledger`; required on `recover`. |
| `version`, `outcome`, `identity` | `recover` | See Recovery. |

`INSERT INTO stackql_preview.iac.run` needs exactly one of `blueprint`, `resources` or
`resources_file`; none, or more than one, is refused. `SELECT FROM stackql_preview.iac.plan`
requires `collection` and `resources`; it does not read `resources_file` or `blueprint`. `ledger`
accepts only `collection`, `state` and `resource_key`. `recover` accepts `collection`, `state`,
`resource_key` and `version` in WHERE, and only `outcome` and `identity` in SET.

## Resource specification

A JSON list of resources, or an object whose `resources` member is that list. Keys must be unique.
The fields used in stackql's own tests are `key`, `provider`, `address` (service.resource),
`identity` (the response field holding the resource ID), `addressed_by` (the request parameter that
takes the ID), `desired` (the desired properties) and `inbound` (dependencies, as
`[{"from": "<key>", "as": "<parameter>"}]`). The full format is Omni's; this skill does not cover it.

```json
[{"key": "widget", "provider": "widgets", "address": "things.iac_widget",
  "identity": "id", "addressed_by": "id", "desired": {"color": "blue"}}]
```

## Converge

```sql
INSERT INTO stackql_preview.iac.run (resources_file, collection, state, region)
VALUES ('<path-to>/resources.json', 'my-collection', 'demo', 'ap-southeast-2')
RETURNING *;
```

Result rows have `key` (`<collection>/<resource_key>`), `status` and `identity`. Run the same
statement again to check idempotence: the identities must not change and nothing is duplicated.
`status = 'applied'` is also reported for a no-op. Keep converging with the latest desired-state
document; reapplying an older document is a change, not an unchanged rerun.

Blueprint form. `aws_vpc_subnet` is a VPC and a subnet in it; its CIDRs are immutable, so a change
is refused rather than replaced, and tags are stamped at create and not converged. Inputs come from
`DESCRIBE stackql_preview.iac.aws_vpc_subnet`: `region`, `vpc_cidr` and `subnet_cidr` are
required, `vpc_tags` and `subnet_tags` are JSON objects. A missing input fails with
`omnisdk: <input> is required`.

```sql
INSERT INTO stackql_preview.iac.run (blueprint, collection, region, vpc_cidr, subnet_cidr)
VALUES ('aws_vpc_subnet', 'my-vpc', 'ap-southeast-2', '10.42.0.0/16', '10.42.1.0/24')
RETURNING *;
```

## Preview and apply

```sql
SELECT resource_key, action, reason, depends_on, change_set
FROM stackql_preview.iac.plan
WHERE collection = 'my-collection' AND resources = '<inline JSON specification>';
```

`action` is `create`, `update`, `replace`, `no-op` or `unknown` (`unknown` is reported, for
example, for resources that depend on something not yet created). `depends_on` is a JSON array.
Every row carries the same `change_set`. To apply it:

```sql
INSERT INTO stackql_preview.iac.plan (collection, change_set)
VALUES ('my-collection', '<change_set>')
RETURNING status;
```

- Use the same `collection`, `state`, scope columns and inputs as the preview.
- The apply rechecks the plan under the collection lease. If state changed after the preview it is
  refused as stale; preview again.
- The change set is JSON inside a SQL string literal: double every `'` and every `\`.

## Destroy

```sql
DELETE FROM stackql_preview.iac.run
WHERE collection = 'my-collection' AND state = 'demo' AND region = 'ap-southeast-2'
RETURNING *;
```

Use the same `collection`, `state` and scope as the converge. Result rows have `key`, `status`
and, when not destroyed, `reason`. A row with `status = 'outstanding'` means the resource
remains; the statement then fails with `IaC operation did not complete successfully for key <key>
(status outstanding); inspect the journal`. Add `AND keys = '["<resource_key>"]'` to destroy only
named resources.

## Ledger and state location

- State root: `<approot>/iac`. The approot defaults to `.stackql` under the stackql process's
  working directory, or the `--approot` value. `iacStateRoot` in the preview flag overrides the
  root: `--preview '{"unstable":true,"iacStateRoot":"/srv/stackql/iac"}'`.
- Layout: `<root>/ledger/<collection>/` and `<root>/journal/`; with `state = 'demo'`,
  `<root>/demo/ledger/<collection>/` and `<root>/demo/journal/`.
- When caller access management is configured, the root is further partitioned per principal or
  tenant.
- File paths (`resources_file`, `inputs_file`, `@file` overlays) are on the stackql process's
  filesystem; for an MCP server that is the server host, not the client.

```sql
SELECT resource_key, identity, phase, version
FROM stackql_preview.iac.ledger
WHERE collection = 'my-collection' AND state = 'demo';
```

Ledger columns: `address`, `addressed_by`, `collection`, `depends_on`, `identity`, `key`,
`phase` (`live` or `pending`), `provider`, `resource_key`, `version`. A `pending` row's
`identity` is the previous committed identity, not proof that the pending operation happened.

## Rules for a deployment's life

- Keep `collection`, `state`, scope columns such as `region`, the approot or `iacStateRoot`, and
  any `document_merge` overlay the same for every run. A different ledger location can create
  duplicate resources.
- For an existing deployment, point `iacStateRoot` (or `--approot`) at its original ledger before
  continuing.
- Never delete, move or edit the ledger or journals to get past an error, and never switch to a
  new collection name to bypass one.
- After any failed or outstanding result, inspect the returned rows and the journal before any
  further mutation.
- Name tags are labels. Ownership and recovery come from the ledger.

## Recovery

An interrupted or failed operation can leave a ledger entry `pending` (for example a create whose
outcome is unknown, or a delete the provider did not complete). While an entry is pending, destroy
returns `<key> has unresolved intent; recover its outcome before destroying this collection`, and
a preview that includes the key returns `<key> has unresolved intent; recover it before planning`.
There is no automatic replay.

1. Read the entry and note `version`:
   `SELECT resource_key, identity, phase, version FROM stackql_preview.iac.ledger WHERE collection = 'my-collection' AND resource_key = 'subnet_a';`
2. Establish the real outcome at the provider yourself and confirm the operation is no longer in
   flight. Verify completion and readiness, not only existence. Tags are not evidence.
3. Record it, with the version from step 1 (add `state` if the deployment uses one):

```sql
UPDATE stackql_preview.iac.recover SET outcome = 'absent'
WHERE collection = 'my-collection' AND resource_key = 'subnet_a' AND version = '1'
RETURNING *;

UPDATE stackql_preview.iac.recover SET outcome = 'applied', identity = '<verified id>'
WHERE collection = 'my-collection' AND resource_key = 'subnet_a' AND version = '3'
RETURNING *;
```

4. Re-run the original converge with the same collection, state, inputs and specification.

`absent` only clears an unresolved first create with no prior identity and no dependents;
otherwise it is refused (`absent recovery only clears an unresolved first create; prior state must
be preserved`). `applied` needs `identity` and cannot replace a prior identity. Result rows have
`key`, `identity` and `status` (`recovered_absent` or `recovered_applied`). A stale version is
refused (`ledger: version conflict: recovery observed 2, current 1`), as is a non-pending entry
(`<collection>/<key> is not pending`). Each recovery writes a durable assertion to the journal.

## Provider overlays with document_merge

```sql
SET "$.document_merge" = '@/absolute/path/merge.json';
-- or inline: SET "$.document_merge" = '{"patches":[...]}';
INSERT INTO stackql_preview.iac.run (collection, region, resources_file, inputs_file)
VALUES ('my-collection', 'ap-southeast-2', '<path-to>/resources.json', '<path-to>/inputs.json')
RETURNING *;
UNSET "$.document_merge";
```

The overlay changes the session's effective provider documents for later queries and IaC; the
base registry is unchanged. Each `SET` replaces the whole patch set; `UNSET` restores the base.
It is session scoped: a reopened shell needs the `SET` again before continuing the deployment.

## Through the stackql MCP server

Start the server with the preview flag, for example
`stackql mcp --mcp.server.type=stdio --preview '{"unstable":true}'`.

- `run_select_query`: `SELECT` on `plan` and `ledger`, `SHOW`, `DESCRIBE`. It refuses INSERT,
  UPDATE and DELETE in every mode (`mutation_create statements run through run_mutation_query or
  run_lifecycle_operation`).
- `run_mutation_query`: `INSERT INTO run`, `DELETE FROM run`, `INSERT INTO plan`,
  `UPDATE recover`.
- Each MCP tool call runs in a fresh session: a `SET` in one call is gone in the next. Put
  `SET "$.document_merge" = ...;` and the IaC statement in the same `sql` batch.

The server mode (`server_info` reports it) decides each call:

| Statement | Class | `read_only` | `safe` (default) | `delete_safe` | `full_access` |
|---|---|---|---|---|---|
| SELECT on `plan` or `ledger`, SHOW, DESCRIBE | select | allow | allow | allow | allow |
| INSERT INTO `run` or `plan`, UPDATE `recover` | mutation_create | refuse | approval | allow | allow |
| DELETE FROM `run` | mutation_delete | refuse | approval | approval | allow |
| Batch containing `SET`/`UNSET`; SELECT on a bare name such as `iac.ledger` | unknown | refuse | approval | approval | allow |

- Approval uses MCP elicitation: the client shows the tool, query class and SQL, and the call
  proceeds only on accept. Decline or cancel returns a refusal. A client without elicitation is
  refused with a message telling the operator to use an elicitation-capable client or
  `full_access`.
- In `read_only` the IaC writes are refused before anything runs; no ledger state is written.
- State the intended change in the conversation before a gated call. A refusal is enforcement; do
  not reword the statement to get around it.

Converge and destroy are long-running, so prefer a tasks-capable client. When the server has the
tasks extension enabled (`server.extensions.tasks.enabled: true`) and the client declares the
`io.modelcontextprotocol/tasks` extension, a `run_mutation_query` call whose statement is an
`INSERT` or `DELETE` on `stackql_preview.iac.run`, or an `INSERT` on `stackql_preview.iac.plan`,
returns a task that the client polls; approval still happens first. Only a single statement is
eligible: a batch that also carries `SET` runs synchronously. On the task path
`server.extensions.tasks.timeout` (default 30m) replaces `server.request_timeout`. Other clients
get the synchronous result and the call is subject to `server.request_timeout`. A converge or
destroy that timed out may already have changed resources: read the ledger before retrying.

## Troubleshooting

- `'state' must name a directory inside the iac state root`: use a relative name without `..`.
- `ledger accepts collection, state and optional resource_key predicates`: drop `region` and
  other scope columns from ledger reads.
- Credential errors on destroy: stackql passes its `--auth` configuration only for the provider
  of the first resource in a converge or preview. Destroy, and providers after the first, resolve
  credentials from the provider document's own settings and environment variables.
