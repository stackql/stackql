---
name: stackql-deploy
description: Author, extend, review or fix stackql-deploy v2 stacks (a stackql_manifest.yml plus resources/*.iql query files with exists, create, update, createorupdate, statecheck, exports, delete, callback and troubleshoot anchors). Use when asked to write an infrastructure-as-code stack with StackQL, add a resource to a stack, wire exports between resources, or debug a stack that fails in stackql-deploy build, test or teardown. Read anchors (exists, statecheck, exports) are proven live through the stackql MCP server tools list_methods, describe_method, validate_select_query and run_select_query; create, update and delete anchors are never executed through MCP and are checked with stackql-deploy build --dry-run.
license: MIT
metadata:
  source: github.com/stackql/stackql-deploy-rs
  source_tag: v2.4.0
  source_copyright: StackQL Studios
---

# stackql-deploy stacks

stackql-deploy (v2, the Rust release; this skill follows tag v2.4.0) provisions, tests and tears down resources for any StackQL provider from a stack directory:

```text
my-stack/
  stackql_manifest.yml   globals, resources in processing order, props per environment, exports
  resources/
    network.iql          one query file per resource, split into anchors such as /*+ exists */
    subnetwork.iql
```

Every command takes the stack directory and an environment name:

```text
stackql-deploy build    <stack_dir> <stack_env> [-e KEY=VALUE ...] [--env-file FILE] [--dry-run] [--show-queries]
stackql-deploy test     <stack_dir> <stack_env> [same options]
stackql-deploy teardown <stack_dir> <stack_env> [same options]
```

`build` creates or updates resources in manifest order, `test` checks them without running any DML, and `teardown` deletes them in reverse order. `-e` can be repeated; it supplies the variables that `globals` read with `{{ VAR }}` (account ids, secrets, run-specific inputs).

## When to use this skill

- Writing a new stack, or adding a resource to an existing one.
- Turning a working StackQL query into the `exists`, `statecheck` or `exports` anchor of a resource.
- Passing a value produced by one resource (an id, a URL, an ARN) to another.
- Diagnosing a stack whose `build`, `test` or `teardown` fails: a statecheck that never passes, a resource updated on every run, a delete that is never confirmed, or a template that does not render.

## How build decides what to run

For each resource, in order: `exists` runs; if nothing is found, `create` runs. If the resource exists, `statecheck` runs; a result of 0 runs `update`, a result of 1 means no change. After a create or update, any `callback` runs, then `statecheck` runs with its `retries`; if it fails, `troubleshoot` runs and the build stops. Finally `exports` runs and its columns join the context for later resources. A `createorupdate` anchor replaces `create` and `update` for idempotent provider methods and runs without the pre-checks.

## Workflow

Work one resource at a time, in dependency order (network before subnet before instance).

### 1. Discover each method's contract

Call `list_methods` with `provider`, `service` and `resource` (for example `github`, `issues`, `labels`). It returns each method with its SQL verb (`SELECT`, `INSERT`, `UPDATE`, `DELETE`, `EXEC`) and required params. Then call `describe_method` with the same arguments plus `method` for every method the resource file will use:

- the `SELECT` method behind `exists`, `statecheck` and `exports`: its required params become exact-match `WHERE` predicates, and its output fields are the only columns those queries can reference;
- the `INSERT` method behind `create` (its inputs are the column list), the `UPDATE` method behind `update`, and the `DELETE` method behind `delete`.

Describe each method separately: param names can differ between methods of the same resource (the source example `google.compute.networks` filters on `name` in its `SELECT` and on `network` in its `DELETE`). A `create_or_update` method mapped to `INSERT` means `createorupdate` is the right anchor. Params that map to the request path accept exact matches only, never wildcards, `LIKE` or inequalities.

### 2. Prove exists, statecheck and exports live, then template them

Write each read query with real literal values and run it through the MCP server:

1. `validate_select_query` (argument `sql`) catches parser, planner and `no such column` errors. The provider's credentials must resolve, as for execution.
2. `run_select_query` (argument `sql`, small `row_limit`) runs it against the provider.

For example, the `exists` query of the worked example, proven with literals:

```sql
SELECT COUNT(*) AS count
FROM github.issues.labels
WHERE owner = 'my-org'
AND repo = 'my-repo'
AND name = 'bug'
```

What each anchor has to return:

- `exists`: either one row with one column named `count` (1 = exists, 0 = not), or an identifier column such as `vpc_id` (any row = exists, no rows = not). Identifier columns are captured as `{{ this.<field> }}` for the resource's later anchors. Run it for an object that exists and for one that does not; an empty result with a 404 message is the normal "not found" answer.
- `statecheck`: one row with a `count` column, 1 when every desired property matches and 0 otherwise. Prove both outcomes: 1 for the current object, and 0 after changing one literal to a value the object does not have.
- `exports`: one row whose column names equal the export names declared for the resource in the manifest (alias with `AS`).

Once a query is proven, replace each literal with a template variable: `'{{ label_name }}'` for props and globals, `'{{ this.vpc_id }}'` for values captured by `exists`, and the built-ins `{{ stack_name }}`, `{{ stack_env }}` and `{{ resource_name }}` for names and tags. Props whose values are lists or maps render as compact JSON text, so quote them like strings (`'{{ subnet_tags }}'`). Add `retries` and `retry_delay` to `statecheck` (and `exports` if needed) when the provider applies changes asynchronously: `/*+ statecheck, retries=5, retry_delay=10 */`.

If the provider's credentials do not resolve, say so. Do not write read anchors you could not run without flagging them as unproven.

### 3. Write create, update and delete; never run them through MCP

Write `create` as `INSERT INTO ... SELECT ...`, `update` as `UPDATE ... SET ... WHERE ...` and `delete` as `DELETE FROM ... WHERE ...`, using only the inputs and params that `describe_method` reported. For providers whose mutations return an asynchronous operation handle (for example `awscc`), end the statement with `RETURNING *` and add a `callback:create` / `callback:delete` anchor that polls the handle.

Do not execute `create`, `update`, `createorupdate`, `delete` or `command` queries through the MCP server. Do not call `run_mutation_query` or `run_lifecycle_operation` for them, in any server mode, even with literal values. stackql-deploy runs them. Check them instead with a dry run, run locally by you or the user:

```text
stackql-deploy build <stack_dir> <stack_env> --dry-run -e KEY=VALUE
```

A dry run renders every query the build would run and logs it without executing any of them, read or write. Exports become the placeholder `<evaluated>` so later resources still render. Read the output for leftover `{{ }}`, wrong quoting, and columns or params that do not match the method contract. `teardown` accepts `--dry-run` too.

### 4. Wire exports between resources

1. List the names under the producing resource's `exports:` in the manifest.
2. Return them as columns from that resource's `exports` anchor.
3. Use them in later resources as `{{ name }}` (the latest export with that name) or `{{ resource_name.name }}` (fixed once that resource has run). Use the qualified form when a name is not unique across the stack.
4. List values that the caller or CI pipeline needs under the top-level `exports:`. After `build` or `test` they are printed and written to `.stackql-deploy-exports` as `STACKQL_DEPLOY__<stack>__<env>__<name>`.

Resource names must be unique. Put sensitive exports under the resource's `protected:` list so they are masked in logs. Fields returned by `RETURNING *` are available as `{{ callback.<field> }}` in the same resource and `{{ <resource_name>.callback.<field> }}` downstream; `return_vals` captures named fields as `{{ this.<field> }}`.

### 5. Assemble the manifest

- `version: 1`, `name` (becomes `stack_name`; no environment in it), `description`, `providers`.
- `globals` for stack-wide values. Read secrets and account ids from variables (`value: "{{ AWS_REGION }}"`) and mark secrets `protected: true`. Never put secret values in the manifest.
- One entry per resource under `resources`, in dependency order, with `props` (`value`, or `values` keyed by environment) and `exports`. Set `file` when several resources share one `.iql`, and `type: query` for read-only lookups.
- Optional fields: `if`, `merge`, `protected`, `skip_validation`, `skip_on_delete`, `auth`, `sql`, `return_vals`; see `references/manifest.md`.

Hand over with the commands in order: `build --dry-run`, `build`, `test`, and `teardown` when finished.

## Common mistakes

- `tojson` does not exist in v2. Use Tera's `json_encode`, and only on structured values (output of `from_json`, `merge_lists`, `merge_objects`, loop variables), not on JSON-text props.
- Filter arguments must be named: `a | from_json | merge_lists(other=b | from_json)`, not `merge_lists(b)`.
- `exists` with a count column, and every `statecheck`, must name the column `count`.
- Unquoted string values in SQL: write `'{{ var }}'`.
- An `if` condition with a bare variable name. Write `if: "'{{ stack_env }}' == 'prd'"`; a bare word is compared as literal text.
- A name listed in `exports` that the `exports` query does not return as a column.
- No `retries` on `statecheck` for a provider that creates resources asynchronously, so the build fails before the resource is ready.
- `preflight` and `postdeploy` are deprecated aliases for `exists` and `statecheck`.

## References and worked example

Load these files when you need the detail:

- `references/resource-query-files.md` - every anchor and option: count and identifier-based `exists`, `{{ this.<field> }}`, `statecheck`, `exports`, `callback`, `troubleshoot`, `retries`, `postdelete_retries`, `idempotency_token`, plus complete examples.
- `references/manifest.md` - every manifest field: resource types, `props` and `values`, `file()`, `merge`, `protected`, `if`, `skip_on_delete`, `skip_validation`, `auth`, `sql`, `return_vals`, export scoping, stack exports.
- `references/template-filters.md` - Tera filters (`from_json`, `json_encode`, `generate_patch_document`, `base64_encode`, `sql_list`, `sql_escape`, `merge_lists`, `merge_objects`, `to_aws_tag_filters`), built-in variables and `uuid()`.

The worked example in `assets/example-github-labels/` is a two-resource stack on GitHub repository labels. It costs nothing to run, and teardown removes the label it creates.

- `assets/example-github-labels/stackql_manifest.yml` - globals read from `-e` inputs, per-environment `values` (`ci` and `dev` only), a `query` type resource with inline `sql`, resource and stack `exports`.
- `assets/example-github-labels/resources/label.iql` - count-based `exists`, `create`, `statecheck` with retries, `update`, `exports`, `delete` against a REST provider without `RETURNING`.

To try it, set `STACKQL_GITHUB_USERNAME` and `STACKQL_GITHUB_PASSWORD` (a token with issues:write on the repository), copy the directory, then run:

```text
stackql-deploy build example-github-labels dev --dry-run -e GITHUB_OWNER=<owner> -e GITHUB_REPO=<repo> -e RUN_ID=<suffix> -e LABEL_COLOR=<color>
```

Change `LABEL_COLOR` between two real builds to exercise the `statecheck` -> `update` path.
