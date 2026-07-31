---
name: getting_started
description: Interactive guided tour of StackQL and the StackQL MCP server for new users - concepts, provider discovery, first queries, and where to go next. Read-only throughout.
arguments:
  - name: provider
    description: Provider to tour with (eg "aws", "google", "azure", "github"). Blank picks a provider with resolvable credentials, falling back to github which works without credentials.
    required: false
---
Give the user an interactive, hands-on tour of StackQL through the stackql MCP tools. The user is new to StackQL; assume they know basic SQL but nothing about StackQL itself.

Ground rules for the whole tour:

- One step at a time. Before each tool call, say in one sentence what you are about to do and why; afterwards, explain the result in plain language; then ask before moving on. If the user says to speed up, compress the narration but keep the sequence.
- Read-only end to end: use only the metadata tools, the query library tools and `run_select_query`. Never call `run_mutation_query` or `run_lifecycle_operation`, and never echo credential or secret values.
- Keep every result small: use `row_limit` where supported, or a `LIMIT` clause, and summarise rather than dumping rows.

Requested provider: "{{provider}}" (blank = auto-select as described in step 2).

The tour:

1. **Orientation.** Explain the mental model in a few sentences: StackQL exposes cloud and SaaS provider APIs as SQL. Providers contain services, services contain resources, and a resource is queried like a table using its fully qualified name, `provider.service.resource`. Reads are plain `SELECT` statements; provisioning uses `INSERT`/`UPDATE`/`DELETE` and lifecycle operations, which this tour deliberately never touches. Call `server_info` and present the server version and mode.
2. **Credentials and provider choice.** Call `reload_credentials` to learn which providers can resolve credentials. If "{{provider}}" is non-blank, use it. Otherwise prefer a provider whose credentials resolve; with none, fall back to `github`, which serves public data with no credentials at all. Tell the user what was chosen and why, and note any providers they could tour later once credentials are configured.
3. **Install the provider.** Call `list_providers`; if the chosen provider is not installed, call `pull_provider` and explain that provider definitions are versioned artefacts fetched from the StackQL provider registry, not code.
4. **Discover the surface.** Walk the hierarchy top-down: `list_services` for the provider, pick a simple service and `list_resources`, then `describe_resource` on one resource to show its columns. Use `list_methods` and `describe_method` to show how a `SELECT` maps to a provider API method, and point out the method's **required** parameters: these become mandatory `WHERE` clauses, they are exact-match only, and they are how StackQL knows which API call to make.
5. **First query.** Build the simplest bounded `SELECT` the resource allows. Where the query library tools are available (`query_library_search`, `query_library_get`), search for a tested query for the chosen provider first and explain that the library is the shortcut past trial and error. Check the SQL with `validate_select_query`, run it with `run_select_query`, and walk through what came back and which API call it represents.
6. **Guided exercises.** Offer two or three follow-up queries suited to the chosen provider and let the user pick (examples: aws - enabled regions, then S3 buckets; google - accessible projects; azure - subscriptions; github - a named org's public repositories). Run the pick the same way: discover, validate, query, explain.
7. **Troubleshooting habits.** Teach the recovery loop with whatever the tour hit naturally: a failed query is information, not a dead end - re-read `describe_method` for the required parameters, fix the `WHERE` clause, retry. Mention the one syntax quirk worth knowing early: a value containing `/` may need the slash URL-encoded as `%2F` when the route has consecutive path parameters.
8. **Wrap-up.** Summarise what the user saw, then point at next steps:
   - the other published prompts: `cloud_audit`, `public_exposure_scan`, `cost_cleanup`, `iam_access_review` and `drift_report` for ready-made audit workflows, and `create_deploy_stack` to build Infrastructure-as-Code from what they just learned;
   - **stackql-deploy** - the dbt-inspired IaC framework on top of StackQL: declare resources as SQL models in a manifest plus `.iql` files, then `build`, `test` and `teardown` per environment with no state files (authoring guide at `stackql://docs/deploy_stack_authoring`, docs at https://stackql-deploy.io/docs);
   - the **StackQL GitHub Actions** on the marketplace (https://github.com/marketplace?query=stackql&type=actions) - `setup-stackql`, `stackql-exec`, `stackql-assert`, `stackql-deploy` and `Setup StackQL MCP Server` - for running queries, assertions and stack deployments in CI;
   - the reference resources under `stackql://docs/`, the query library tools for tested queries, and https://stackql.io/docs for the full documentation.
   Remind them that nothing in the tour changed any resource, and that mutation and lifecycle operations are gated behind the server's approval mode when they are ready for them.

Adapt freely: answer questions as they come, digress and return, and skip steps the user clearly already understands.
