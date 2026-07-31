---
name: create_deploy_stack
description: Guide the user through generating a stackql-deploy stack (manifest + .iql resource files) with live-tested queries via the stackql MCP tools. Read-only - mutations are only ever rendered for stackql-deploy, never executed.
arguments:
  - name: requirements
    description: What the stack should provision (eg "a VPC with a public subnet and a web server on AWS"). Blank means elicit the requirements first.
    required: false
  - name: provider
    description: Target provider (eg "aws", "awscc", "azure", "google"). Blank means infer from the requirements or ask.
    required: false
  - name: environments
    description: Comma-separated environment names the stack should support (eg "dev, prd"). Blank defaults to dev/sit/prd.
    required: false
---
Build a complete, working stackql-deploy stack with the user, producing `stackql_manifest.yml` and one `resources/<name>.iql` file per resource, with every read query proven against the live provider before it is baked in. The authoring reference is the `stackql://docs/deploy_stack_authoring` resource - follow its layout, anchor and workflow rules; consult it now if the client exposes resources.

Requirements: "{{requirements}}" (blank = elicit). Provider: "{{provider}}" (blank = infer or ask). Environments: "{{environments}}" (blank = dev/sit/prd).

Hard rules:

- Read-only session: use only the metadata tools, the query library tools and `run_select_query`. Never call `run_mutation_query` or `run_lifecycle_operation` - the generated `create`/`update`/`delete` queries are executed by `stackql-deploy`, not by you. Never echo credential or secret values.
- Live-test what can be tested: every `exists`, `statecheck` and `exports` query must be validated (`validate_select_query`) and, where credentials resolve, executed (`run_select_query`, small `row_limit`) with real substituted values before the templated form goes into a file. Mutation anchors are checked only via `stackql-deploy build <stack> <env> --dry-run`, run by the user.
- Work incrementally: one resource at a time, showing the user each file as it lands and asking before moving to the next.

Procedure:

1. **Scope.** Establish what to provision, the provider, the environments, and a short stack name. Call `server_info`, then `reload_credentials` to confirm the target provider's credentials resolve (without them you can still author, but flag that live testing is degraded to `validate_select_query` only).
2. **Map requirements to resources.** Call `list_providers` (and `pull_provider` if needed), then use `list_services` / `list_resources` to translate each requirement into concrete `provider.service.resource` types. Propose the resource list in dependency order (network before compute, etc) with the exports each later resource needs; get the user's agreement.
3. **Per resource - discover the contract.** `describe_resource` for columns; `list_methods` + `describe_method` for the insert/select/update/delete methods and their **required** parameters. State plainly which parameters the create query must supply and which WHERE clauses the checks must carry.
4. **Per resource - draft and live-test the read anchors.** Check the query library (`query_library_search` / `query_library_get`) for tested shapes first. Draft `exists` (returning `count`), `statecheck` (presence plus desired configuration; add `retries` for async resources) and `exports` (the columns downstream resources need). Substitute real values, validate, and run them live where credentials allow; iterate until the shapes are proven, then convert the literals back to `{{ template }}` placeholders.
5. **Per resource - write the mutation anchors.** Derive `create` (and `update`/`delete` as applicable, or a single `createorupdate` when the provider method is idempotent) strictly from the discovered contract. These are never executed here.
6. **Assemble the manifest.** `globals` for stack-wide values (interpolating environment variables for secrets and account ids; use the `stack_name`/`stack_env` built-ins in names and tags), per-env `props` where values differ by environment, `exports` wired to downstream `{{ placeholders }}`, resources in dependency order.
7. **Hand over.** Present the final tree, then give the user the exact commands: `stackql-deploy build <stack> <env> --dry-run` to inspect every rendered query, `build` to deploy, `test` to verify, `teardown` to remove. Mention the StackQL GitHub Actions (`stackql-deploy`, `setup-stackql`, `stackql-exec`, `stackql-assert`) for running the same stack in CI, with `--dry-run` on pull requests as the review gate.

If a discovery or live test contradicts an assumption (missing method, extra required parameter, differently named column), say so, adjust the design and re-test - the point of this workflow is that nothing unproven reaches the stack files.
