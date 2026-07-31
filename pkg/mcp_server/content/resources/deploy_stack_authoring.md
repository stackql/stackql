---
name: stackql_deploy_stack_authoring
description: How to author a stackql-deploy stack - project layout, manifest anatomy, .iql query anchors, execution strategy, and the live-query-testing workflow using the stackql MCP tools. Referenced by the create_deploy_stack prompt.
---
# Authoring a stackql-deploy stack

`stackql-deploy` is a multi-cloud Infrastructure-as-Code framework built on StackQL, inspired by dbt: resources are declared as SQL-like models, then provisioned with `build`, verified with `test` and deprovisioned with `teardown` - no state files. The current edition is the Rust rewrite (v2.x), installed via `cargo install stackql-deploy` or the binaries on the GitHub releases page. Full documentation: https://stackql-deploy.io/docs

## Project layout

```
my_stack/
  stackql_manifest.yml
  resources/
    <resource_name>.iql     # one file per manifest resource entry
```

Scaffold with `stackql-deploy init my_stack --provider <provider>`.

## Manifest anatomy (stackql_manifest.yml)

- `version: 1`, `name`, `description`.
- `providers`: list of providers the stack uses; a version can be pinned with `provider::vX.Y.Z` (eg `awscc::v26.03.00379`).
- `globals`: named values available to every resource template. Values interpolate environment variables (`"{{ AWS_REGION }}"`) and the built-ins `stack_name`, `stack_env` and `resource_name`.
- `resources`: ordered list - deployment order top to bottom, teardown in reverse. Each entry has `name` (matching `resources/<name>.iql`), optional `description`, `props` and `exports`.
- `props`: per-resource template variables. A prop has either a single `value`, or per-environment `values` keyed by environment name (eg `dev`/`sit`/`prd`), and can `merge` in list-valued globals (the common pattern for tags).
- `exports`: column names captured from the resource's `exports` query, available to all later resources as `{{ <export_name> }}` - this is how ids flow between resources (vpc -> subnet -> route table).

## Query anchors (resources/<name>.iql)

One file holds all lifecycle queries for a resource, separated by SQL hint comments:

- `/*+ create */` - `INSERT INTO provider.service.resource(...) SELECT ...` provisioning query.
- `/*+ update */` - `UPDATE ... SET ... WHERE ...` convergence query (optional).
- `/*+ createorupdate */` - single idempotent mutation; when present, all checks are skipped.
- `/*+ delete */` - `DELETE FROM ... WHERE ...` teardown query.
- `/*+ exists */` - `SELECT COUNT(*) as count ...` presence probe; must return a `count` column.
- `/*+ statecheck, retries=5, retry_delay=5 */` - desired-state probe (presence AND correct configuration); retries accommodate async provisioning.
- `/*+ exports */` - `SELECT` returning the columns named in the manifest `exports`.

All queries are templates: `{{ prop_or_global_or_export }}` placeholders are rendered before execution.

## Execution strategy per resource

- `createorupdate` present: run it, skip all checks.
- `statecheck` present: `exists` -> absent ? `create` : `statecheck` -> wrong state ? `update` : up to date; then `exports`.
- No `statecheck` but `exports` present: try `exports` first - data back means validated and captured in one call; empty means `exists`/`create`-or-`update`, then `exports`.
- `test` runs the check queries only; `build --dry-run` prints rendered queries without executing; `--show-queries` prints them during a real run.

## Authoring workflow with live-tested queries (MCP)

Author stacks against the live provider surface instead of guessing, using the stackql MCP tools; the split matters:

1. **Discover the contract.** For each resource type: `list_methods` / `describe_method` to learn the insert/select/update/delete methods and their **required** parameters - these dictate the columns in the `create` query and the mandatory `WHERE` clauses everywhere else. `describe_resource` gives the selectable columns for `statecheck` and `exports`.
2. **Live-test the read anchors.** `exists`, `statecheck` and `exports` are plain SELECTs: substitute real values for the templates, `validate_select_query`, then `run_select_query` against the live provider to prove the shape (correct columns, `count` present, JSON extraction paths right) before baking the templated form into the `.iql` file. Where the query library tools are available, search them first for tested shapes.
3. **Never execute mutation anchors via MCP.** `create`/`update`/`delete` queries are written from the discovered contract but executed only by `stackql-deploy` itself - validate them with `stackql-deploy build <stack> <env> --dry-run`, which renders and prints every query without touching the provider.
4. **Wire the data flow.** Order resources by dependency, export the ids later resources need, and reference them as template variables; put environment-varying values in per-env `props`, secrets and account ids in environment variables interpolated by `globals`.

## Commands

```
stackql-deploy init <stack> --provider <provider>
stackql-deploy build <stack> <env> --dry-run          # preview, no mutations
stackql-deploy build <stack> <env> -e KEY=value       # deploy (also --env-file=.env)
stackql-deploy test <stack> <env>                     # checks only
stackql-deploy teardown <stack> <env>                 # reverse-order deprovision
```

Other flags: `--on-failure=rollback|ignore|error`, `--log-level`, `--show-queries`. `stackql-deploy info` shows environment and provider versions.

## CI/CD - StackQL GitHub Actions

The StackQL actions on the GitHub Marketplace (https://github.com/marketplace?query=stackql&type=actions) cover the pipeline: `stackql-deploy` (build/test stacks in a workflow), `setup-stackql` (install the CLI), `stackql-exec` (run a single query), `stackql-assert` (test/audit infrastructure state), and `Setup StackQL MCP Server` (MCP server config for agentic CI workflows). A typical pipeline runs `build <stack> <env> --dry-run` on pull requests and `build` + `test` on merge.

Worked examples for AWS, Azure, Google, Databricks and Snowflake live in the stackql-deploy repository's `examples/` directory.
