Source: github.com/stackql/stackql-deploy-rs, tag v2.4.0, website/docs/resource-query-files.md (MIT, StackQL Studios). Condensed. The build and teardown order notes are checked against src/commands/build.rs and src/commands/teardown.rs at the same tag.

# Resource query files (.iql)

Each resource in `stackql_manifest.yml` has a query file in the stack's `resources/` directory, named `<resource name>.iql` unless the manifest sets `file`. The file holds several StackQL queries, each introduced by an anchor comment. The manifest supplies the template values for the target environment at deploy time.

## Anchor syntax

An anchor must start at the beginning of a line, and its query follows on the next lines. Options go after the anchor name, separated by commas:

```sql
/*+ statecheck, retries=5, retry_delay=10 */
SELECT COUNT(*) as count FROM google.compute.networks
WHERE name = '{{ vpc_name }}'
AND project = '{{ project }}'
```

Templates use Tera (Jinja2-compatible) syntax. Variables come from manifest `globals`, the resource's `props`, exports of earlier resources, and the built-in variables listed below.

| Anchor | Statement | Purpose |
|---|---|---|
| `exists` | `SELECT` | Does the resource exist (by identifier, not desired state)? Decides create vs update. Alias `preflight` (deprecated). |
| `create` | `INSERT` | Create the resource when `exists` finds nothing. |
| `update` | `UPDATE` | Bring an existing resource that fails `statecheck` into the desired state. |
| `createorupdate` | `INSERT` or `UPDATE` | Single idempotent mutation, used in place of `create` and `update`. |
| `statecheck` | `SELECT` | Does the resource match the desired state? Alias `postdeploy` (deprecated). |
| `exports` | `SELECT` | Return values for later resources and for stack exports. |
| `delete` | `DELETE` | Remove the resource during `teardown`. |
| `callback`, `callback:<op>` | `SELECT` | Poll an async operation after a DML statement that used `RETURNING *`. |
| `troubleshoot`, `troubleshoot:<op>` | `SELECT` | Diagnostic query run once when verification fails. |
| `command` | any | Body of a `command` type resource when the manifest has no inline `sql`. |

## How build uses the anchors

For each resource, in manifest order:

1. `exists` runs. Not found -> `create` (or `createorupdate`) runs.
2. Found -> `statecheck` runs. Returns 0 -> `update` (or `createorupdate`) runs. Returns 1 -> no change.
3. After a create or update, any matching `callback` runs, then `statecheck` runs with its `retries`. A failed statecheck runs `troubleshoot` and fails the build.
4. `exports` runs and its columns are added to the context.

When a `createorupdate` anchor is present, build skips the existence and state pre-checks and runs it. `test` runs no DML. `teardown` walks the resources in reverse order, runs `delete` and then re-runs `exists` until the resource is gone (see `postdelete_retries` below).

## exists

`exists` tests whether the resource is there, by its identifier. It can return either of two shapes.

Count-based: one row with one column named `count`. `1` means the resource exists, `0` means it does not.

```sql
/*+ exists */
SELECT COUNT(*) as count FROM google.compute.networks
WHERE name = '{{ vpc_name }}'
AND project = '{{ project }}'
```

Identifier-based: return a column other than `count` (for example `vpc_id`). Any returned row means the resource exists, no rows means it does not. Every non-`count` column is captured as a resource-scoped variable available to the resource's later queries (`statecheck`, `exports`, `delete`) as `{{ this.<field> }}`, which expands to `{{ <resource_name>.<field> }}`. Use this when the identifier is assigned by the provider and has to be discovered, for example by tag:

```sql
/*+ exists */
WITH tagged_resources AS
(
    SELECT split_part(ResourceARN, '/', 2) as vpc_id
    FROM awscc.tagging.tagged_resources
    WHERE region = '{{ region }}'
    AND TagFilters = '{{ global_tags | to_aws_tag_filters }}'
    AND ResourceTypeFilters = '["ec2:vpc"]'
),
vpcs AS
(
    SELECT vpc_id
    FROM awscc.ec2.vpcs_list_only
    WHERE region = '{{ region }}'
)
SELECT r.vpc_id
FROM vpcs r
INNER JOIN tagged_resources tr
ON r.vpc_id = tr.vpc_id;

/*+ statecheck, retries=5, retry_delay=5 */
SELECT COUNT(*) as count FROM
(
SELECT
AWS_POLICY_EQUAL(tags, '{{ vpc_tags }}') as test_tags
FROM awscc.ec2.vpcs
WHERE Identifier = '{{ this.vpc_id }}'
AND region = '{{ region }}'
AND cidr_block = '{{ vpc_cidr_block }}'
) t
WHERE test_tags = 1;
```

The `INNER JOIN` with the `*_list_only` resource makes the check require both the expected tags and a resource that still exists, which drops stale tag records for deleted resources.

`exists` also takes the teardown options `postdelete_retries` and `postdelete_retry_delay` (see Query options).

## create

An `INSERT ... SELECT` that creates the resource:

```sql
/*+ create */
INSERT INTO google.compute.networks
(
 project,
 name,
 autoCreateSubnetworks,
 routingConfig
)
SELECT
'{{ project }}',
'{{ vpc_name }}',
false,
'{"routingMode": "REGIONAL"}'
```

## update

An `UPDATE ... SET ... WHERE` that changes an existing resource. It runs when `exists` finds the resource and `statecheck` returns 0. The worked example in `assets/example-github-labels/resources/label.iql` has one.

## createorupdate

For resources whose provider method is idempotent. `SHOW METHODS IN <resource>` (or the MCP `list_methods` tool) showing a `create_or_update` method mapped to `INSERT` is the usual sign:

```sql
/*+ createorupdate */
INSERT INTO azure.network.virtual_networks(
   virtualNetworkName,
   resourceGroupName,
   subscriptionId,
   location,
   properties,
   tags
)
SELECT
   '{{ vnet_name }}',
   '{{ resource_group_name }}',
   '{{ subscription_id }}',
   '{{ location }}',
   '{"addressSpace": {"addressPrefixes":["{{ vnet_cidr }}"]}}',
   '{{ global_tags }}'
```

It can also be an `UPDATE` that sets state on a pre-existing resource, for example an AWS Cloud Control patch built with the `generate_patch_document` filter.

## delete

A `DELETE` used by `teardown`. Parameter names can differ from the `SELECT` columns of the same resource; in the source example `google.compute.networks` is selected by `name` but deleted by `network`:

```sql
/*+ delete */
DELETE FROM google.compute.networks
WHERE network = '{{ vpc_name }}' AND project = '{{ project }}'
```

## statecheck

A `SELECT` that returns one row with one column named `count`: `1` when the resource is in the desired state, `0` when it is not. It usually runs right after a create or update, so add `retries` and `retry_delay` for providers that apply changes asynchronously.

```sql
/*+ statecheck, retries=5, retry_delay=10 */
SELECT COUNT(*) as count FROM google.compute.networks
WHERE name = '{{ vpc_name }}'
AND project = '{{ project }}'
AND autoCreateSubnetworks = false
AND JSON_EXTRACT(routingConfig, '$.routingMode') = 'REGIONAL'
```

Useful functions for desired-state comparisons: `JSON_EQUAL`, `AWS_POLICY_EQUAL`, `JSON_EXTRACT`, `JSON_EACH`.

## exports

A `SELECT` whose columns are the variables listed under the resource's `exports` in the manifest. Columns are matched by name, so alias them:

```sql
/*+ exports */
SELECT
'{{ vpc_name }}' as vpc_name,
selfLink as vpc_link
FROM google.compute.networks
WHERE name = '{{ vpc_name }}'
AND project = '{{ project }}'
```

`exports` accepts `retries` and `retry_delay`. A `query` type resource is usually just an `exports` anchor:

```sql
/*+ exports, retries=5, retry_delay=5 */
SELECT
target_key_id as logging_kms_key_id
FROM aws.kms.aliases
WHERE region = '{{ region }}'
AND Identifier = 'alias/{{ stack_name }}/{{ stack_env }}/logging';
```

## callback

Optional polling query that runs after a `create`, `update` or `delete` to follow a long-running asynchronous operation. It is only used when that DML statement ends with `RETURNING *` and the provider returns a tracking handle. The main use is the AWS Cloud Control API (`awscc`), whose mutations return a `ProgressEvent` with `OperationStatus` and `RequestToken`.

```sql
/*+ callback:create, retries=10, retry_delay=15, short_circuit_field=ProgressEvent.OperationStatus, short_circuit_value=SUCCESS */
SELECT OperationStatus = 'SUCCESS' as success
FROM awscc.cloudcontrol.resource_request_statuses
WHERE region = '{{ region }}'
AND RequestToken = '{{ callback.ProgressEvent.RequestToken }}'
```

The qualifier (`:create`, `:update`, `:delete`) ties the callback to that DML anchor. A plain `/*+ callback */` runs after any DML on the resource that used `RETURNING *`. There is no `callback` section in the manifest; everything is configured on the anchor.

| Option | Default | Meaning |
|---|---|---|
| `retries` | 3 | Maximum poll attempts |
| `retry_delay` | 5 | Seconds between attempts |
| `short_circuit_field` | none | Dot-path into the `RETURNING *` result, checked before polling |
| `short_circuit_value` | none | Value of that field that means polling can be skipped |

Success condition: the query returns a column named `success` (or `count`), and the operation is complete when a row has `success` equal to `1` or `true`. No rows or a falsy value -> wait `retry_delay` and retry. When retries run out the run fails with a timeout error.

`RETURNING *` capture: the first row of the response is stored as

- `callback.<field>` - shorthand inside the current resource's `.iql` file, overwritten by the next DML with `RETURNING *` on any resource;
- `<resource_name>.callback.<field>` - fully qualified, available to every later resource for the rest of the run.

`RETURNING *` without a callback is valid; the row is captured and nothing is polled. Short-circuit: when the provider already returns the final status in the `RETURNING *` row and it matches `short_circuit_value`, the callback is skipped.

Scope and limits:

- Callbacks run only in `build` and `teardown`. In dry-run mode neither the `RETURNING *` capture nor the callback runs.
- There is no way to stop early on a terminal failure (for example `OperationStatus = 'FAILED'`); polling continues until success or the retries run out.
- Only the first row of a `RETURNING *` response is captured.

## troubleshoot

Optional diagnostic query that runs when post-operation verification fails: a `build` statecheck after create or update, or a `teardown` delete that cannot be confirmed. Pair it with `return_vals` in the manifest to capture an operation handle as `{{ this.<field> }}`:

```yaml
return_vals:
  create:
    - RequestToken
```

```sql
/*+ troubleshoot:create */
SELECT OperationStatus, StatusMessage, ErrorCode
FROM awscc.cloud_control.resource_request
WHERE RequestToken = '{{ this.RequestToken }}'
AND region = '{{ region }}';
```

- The qualifier (`:create`, `:update`, `:delete`) ties it to one operation; a plain `/*+ troubleshoot */` runs after any failed operation.
- It runs once (not polled). Options `retries` (default 1) and `retry_delay` (default 0).
- If its template variables are missing it is skipped silently; if the query fails a warning is logged and the original error is still reported.
- Results are logged as pretty-printed JSON at `info` level. It does not run in dry-run mode.
- It renders with the full context at the point of failure: globals and props, `callback.*` from `RETURNING *`, `this.*` from `exists`, and upstream exports.

## Query options

`retries` and `retry_delay` (seconds) give asynchronous or slow operations time to finish or reach the desired state before the query is treated as failed:

```sql
/*+ statecheck, retries=5, retry_delay=5 */
SELECT COUNT(*) as count FROM azure.resources.resource_groups
WHERE subscriptionId = '{{ subscription_id }}'
AND resourceGroupName = '{{ resource_group_name }}'
AND location = '{{ location }}'
AND JSON_EXTRACT(properties, '$.provisioningState') = 'Succeeded'
```

`postdelete_retries` and `postdelete_retry_delay` go on the `exists` anchor and apply only to `teardown`. After each `delete` (and its `callback:delete`, if any), `exists` runs immediately and, while the resource is still present, up to `postdelete_retries` more times with `postdelete_retry_delay` seconds between checks (defaults 10 and 5). If it is still present after the last check, the `delete` is re-issued according to the `delete` anchor's own `retries` and `retry_delay` (default `retries=1`, a single attempt); after that the resource is reported as not confirmed deleted and teardown moves on.

```sql
/*+ exists, postdelete_retries=10, postdelete_retry_delay=5 */
SELECT COUNT(*) as count FROM google.compute.instances
WHERE name = '{{ instance_name }}'
AND project = '{{ project }}'
AND zone = '{{ zone }}'
```

## Built-in variables

| Variable | Scope | Meaning |
|---|---|---|
| `stack_name` | global | `name` from the manifest |
| `stack_env` | global | environment argument given to the CLI (`dev`, `prd`, ...) |
| `resource_name` | per resource | name of the resource being processed |
| `idempotency_token` | per resource | UUID v4, stable for this resource for the whole run |
| `this.idempotency_token` | inside the resource's `.iql` | preferred form; expands to `<resource_name>.idempotency_token` |
| `<resource_name>.idempotency_token` | global | the token from any later resource |
| `this.<field>` | inside the resource's `.iql` | fields captured by an identifier-based `exists`, or by `return_vals` |

`idempotency_token` is generated once per resource per invocation and stays the same across retries in that run. Pass it where a provider accepts a client request token so that a retry is recognised as the same request. `{{ uuid() }}` generates a new value on every render, so it is not retry-safe.

```sql
/*+ create */
INSERT INTO awscc.cloudformation.stacks(
  StackName,
  TemplateURL,
  ClientRequestToken,
  region
)
SELECT
  '{{ stack_name }}-{{ stack_env }}',
  '{{ template_url }}',
  '{{ this.idempotency_token }}',
  '{{ region }}'
RETURNING *
```

## Complete example: awscc bucket with callbacks

```sql
/*+ exists */
SELECT COUNT(*) as count
FROM awscc.s3.buckets
WHERE region = '{{ region }}'
AND Identifier = '{{ bucket_name }}'

/*+ create */
INSERT INTO awscc.s3.buckets(BucketName, region)
SELECT '{{ bucket_name }}', '{{ region }}'
RETURNING *

/*+ callback:create, retries=10, retry_delay=15, short_circuit_field=ProgressEvent.OperationStatus, short_circuit_value=SUCCESS */
SELECT OperationStatus = 'SUCCESS' as success
FROM awscc.cloudcontrol.resource_request_statuses
WHERE region = '{{ region }}'
AND RequestToken = '{{ callback.ProgressEvent.RequestToken }}'

/*+ statecheck, retries=3, retry_delay=5 */
SELECT COUNT(*) as count
FROM awscc.s3.buckets
WHERE region = '{{ region }}'
AND Identifier = '{{ bucket_name }}'

/*+ exports */
SELECT '{{ bucket_name }}' as bucket_name

/*+ delete */
DELETE FROM awscc.s3.buckets
WHERE region = '{{ region }}'
AND Identifier = '{{ bucket_name }}'
RETURNING *

/*+ callback:delete, retries=10, retry_delay=15, short_circuit_field=ProgressEvent.OperationStatus, short_circuit_value=SUCCESS */
SELECT OperationStatus = 'SUCCESS' as success
FROM awscc.cloudcontrol.resource_request_statuses
WHERE region = '{{ region }}'
AND RequestToken = '{{ callback.ProgressEvent.RequestToken }}'
```

Its manifest entry needs no callback configuration:

```yaml
  - name: aws_s3_workspace_bucket
    file: aws/s3/buckets.iql
    props:
      - name: bucket_name
        value: "{{ stack_name }}-{{ stack_env }}-root-bucket"
      - name: region
        value: "ap-southeast-2"
    exports:
      - bucket_name
```

For a plain REST provider with no async handle (count-based `exists`, `create`, `statecheck` with retries, `update`, `exports`, `delete`), see `assets/example-github-labels/resources/label.iql`.
