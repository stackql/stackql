Source: github.com/stackql/stackql-deploy-rs, tag v2.4.0 (MIT, StackQL Studios). Condensed from website/docs/manifest-file.md, website/docs/file-directive.md and website/docs/manifest_fields/: name.mdx, description.mdx, providers.mdx, version.mdx, globals.mdx, globals/name.mdx, globals/value.mdx, globals/description.mdx, globals/protected.mdx, resources.mdx, resources/name.mdx, resources/description.mdx, resources/type.mdx, resources/file.mdx, resources/props.mdx, resources/props/name.mdx, resources/props/description.mdx, resources/props/value.mdx, resources/props/values.mdx, resources/props/merge.mdx, resources/props/protected.mdx, resources/exports.mdx, resources/protected.mdx, resources/if.mdx, resources/auth.mdx, resources/sql.mdx, resources/skipvalidation.mdx, resources/skipondelete.mdx, exports.mdx. Field spellings checked against src/resource/manifest.rs; `if` evaluation and renamed exports checked against src/commands/base.rs; rendering of list and map values checked against src/core/config.rs.

# stackql_manifest.yml

The manifest sits in the root of the stack directory and declares every resource and its properties for every target environment. Resources are processed in the order they are declared, and a resource can export variables that later resources use (for example a `vpc_id` needed by a subnet). Do not put secrets in the manifest: declare them as `globals` sourced from environment variables, passed at deploy time with `-e KEY=VALUE` or `--env-file`.

## Top-level fields

| Field | Type | Required | Notes |
|---|---|---|---|
| `version` | integer | no | Document version, for example `1`. |
| `name` | string | yes | Stack name, exposed as `stack_name`. Must suit the naming rules of the resources it is used in. Do not put environment designators in it. |
| `description` | string | no | Documentation only. |
| `providers` | string[] | yes | StackQL providers used by the stack, pulled from the registry if not present. The worked example pins a version: `github::v26.05.00393`. |
| `globals` | global[] | yes | Stack-wide variables; can be an empty list. |
| `resources` | resource[] | yes | The resources, in processing order. |
| `exports` | string[] | no | Variables to output after a successful `build` or `test`. |

Built-in variables, available in every property value and query template without declaring them:

| Variable | Value |
|---|---|
| `stack_name` | the manifest `name` |
| `stack_env` | the environment argument to `build`, `test` or `teardown` |
| `resource_name` | the `name` of the resource currently being processed |

```yaml
resources:
  - name: example_vpc
    props:
      - name: vpc_name
        value: "{{ stack_name }}-{{ stack_env }}-{{ resource_name }}"
```

With stack `my-stack` and environment `dev` this renders `my-stack-dev-example_vpc`.

## globals

Each entry has `name` (required), `value` (required), `description` (optional) and `protected` (optional bool). Reference a global as `{{ name }}` in props and queries. A value is a literal, or `{{ ENV_VAR }}` to read an environment variable or secret supplied at deploy time:

```yaml
globals:
  - name: project
    description: google project name
    value: "{{ GOOGLE_PROJECT }}"
  - name: region
    value: australia-southeast1
  - name: postgres_master_password
    value: "{{ POSTGRES_MASTER_PASSWORD }}"
    protected: true
```

`protected: true` masks the rendered value as `********` in all log output, including `--dry-run`, `--show-queries` and `DEBUG` logs. The real value is still sent to the provider. Values can also be lists or maps (for example a `global_tags` list of `Key`/`Value` pairs) or a `file()` directive.

## resources

| Field | Type | Notes |
|---|---|---|
| `name` | string, required | Unique within the stack (duplicates are rejected before any command runs). Exposed as `resource_name`. |
| `type` | string | `resource` (default), `query`, `script`, `multi`, `command`. |
| `file` | string | Query file in `resources/`. Defaults to `<name>.iql`. Lets several resources share one `.iql`. Subdirectories are allowed, for example `aws/s3/buckets.iql`. |
| `description` | string | Documentation only. |
| `props` | prop[] | Property values for the query templates. |
| `exports` | list | Variables exported by the resource's `exports` query. |
| `protected` | string[] | Subset of `exports` to mask in logs. |
| `if` | string | Condition; the resource is skipped when it is false. |
| `auth` | object | Per-resource provider authentication override. |
| `sql` | string | Inline SQL for `query` and `command` types. |
| `skip_validation` | bool | Skip validation checks for this resource in `build` and `test`. |
| `skip_on_delete` | bool | Do not process this resource in `teardown`. |
| `return_vals` | map | Fields to capture from `RETURNING *` as `this.*` variables. |

There is no `callback` field. Callback polling is configured on the `/*+ callback */` anchor in the `.iql` file (see `references/resource-query-files.md`).

### type

- `resource` - provisions or configures something; typically has `create`, `update`, `delete`, `exists`, `statecheck` and `exports` anchors.
- `query` - returns data through an `exports` query (from the `.iql` file or inline `sql`).
- `script` - runs an external script (not a StackQL query).
- `multi` - resources created in a loop (for example three VMs).
- `command` - runs a statement such as an `UPDATE` or `INSERT` during `build`; exports nothing.

### props

Each prop has `name` (required), `description`, exactly one of `value` or `values`, and optionally `merge` and `protected`.

`values` selects by `stack_env`:

```yaml
props:
  - name: ip_cidr_range
    values:
      prd:
        value: 192.168.0.0/16
      sit:
        value: 10.10.0.0/16
      dev:
        value: 10.240.0.0/24
```

Values can be scalars, lists or maps, and can contain template expressions. Lists and maps reach the query templates as compact JSON text, which is why the source examples quote them in SQL like strings (`'{{ subnet_tags }}'`).

`merge` lists context variables (globals or exports of earlier resources) to merge into the prop's value. The value and every merged variable must be the same kind: all lists or all objects.

```yaml
globals:
  - name: global_tags
    value:
      - Key: Provisioner
        Value: stackql
      - Key: StackEnv
        Value: "{{ stack_env }}"
resources:
  - name: example_vpc
    props:
      - name: vpc_tags
        value:
          - Key: Name
            Value: "{{ stack_name }}-{{ stack_env }}-vpc"
        merge:
          - global_tags
```

`protected: true` on a prop masks its rendered value in all log output (same rules as `global.protected`).

### file() directive

`file(relative/path)` inside a prop or global value is replaced by the parsed contents of that file. Paths are relative to the stack's `resources/` directory.

- `.json` is parsed as JSON, `.yml` / `.yaml` as YAML; anything else tries JSON first, then YAML.
- Directives are resolved when the manifest is loaded, before template rendering, so included files should hold static data. Templates in the surrounding value still render.
- Included files can contain further `file()` directives, resolved recursively.
- A missing file or invalid JSON/YAML fails manifest loading with an error naming the file.

```yaml
props:
  - name: policies
    value:
      - PolicyDocument:
          Statement:
            - file(aws/iam/policies/ec2_permissions.json)
            - file(aws/iam/policies/iam_service_linked_role.json)
          Version: '2012-10-17'
        PolicyName: "{{ stack_name }}-{{ stack_env }}-policy"
```

A directive can also replace a whole value (`Statement: file(aws/iam/policies/cross_account_statements.json)` where the file is a JSON array), or a global value (`value: file(common/default_tags.json)`).

### exports and export scoping

Every name in a resource's `exports` must come back as a column of its `exports` query:

```yaml
- name: network
  exports:
    - vpc_name
    - vpc_link
```

```sql
/*+ exports */
SELECT
'{{ vpc_name }}' as vpc_name,
selfLink as vpc_link
FROM google.compute.networks
WHERE name = '{{ vpc_name }}'
AND project = '{{ project }}'
```

An entry can also be a map `column: new_name`, which exports the query column under a new name (the source shows `- aws_iam_role_arn: metastore_access_role_arn`). In v2.4.0 renaming applies only when every entry in that resource's `exports` list is a map.

Each exported variable is stored under two keys:

| Reference | Behaviour |
|---|---|
| `{{ var }}` | Global short name. A later resource exporting the same name overwrites it. |
| `{{ resource_name.var }}` | Resource-scoped. Written once and never overwritten for the rest of the run. |

Use the qualified form when the name is not unique across the stack or must not change.

### protected (resource)

A list of export names whose values are masked wherever they appear in logs, including `--dry-run`, `--show-queries` and `DEBUG` output. To mask inputs instead, use `global.protected` or `prop.protected`.

```yaml
- name: container_registry
  exports:
    - acr_url
    - acr_username
    - acr_password
  protected:
    - acr_password
```

### if

A condition deciding whether the resource is tested, provisioned or deprovisioned. It can reference literals and template variables with `{{ }}`. In v2.4.0 the string is rendered as a template and the result is evaluated as a simple comparison: `'a' == 'b'`, `'a' != 'b'`, `'a' in ['a', 'b']`, or `true` / `false`. Anything else is an error that stops the run. Put every variable inside `{{ }}` and quote both sides:

```yaml
resources:
  - name: get_transfer_kms_key_id
    if: "'{{ stack_env }}' == 'prd'"
```

The source docs also show `if: "environment == 'production'"`; in v2.4.0 the bare word `environment` is not substituted, so that compares the literal text `environment` and is always false.

### auth

Overrides provider authentication for one resource, for example to switch between control plane and data plane credentials in one stack. The object's shape depends on the provider. Variable references can point at exports of earlier resources:

```yaml
- name: users_topic
  auth:
    confluent:
      type: basic
      username_var: app_manager_api_key_id
      password_var: app_manager_api_key_secret
  props:
    - name: topic_name
      value: "users"
```

### sql

Inline SQL for `query` and `command` resources, instead of a separate `.iql` file. A `command` resource needs either `sql` or an `.iql` file with a `/*+ command */` anchor; when both exist, `sql` is used. Templates render as usual.

```yaml
- name: repo_label_count
  type: query
  props: []
  sql: |
    SELECT COUNT(*) AS label_count
    FROM github.issues.labels
    WHERE owner = '{{ owner }}'
    AND repo = '{{ repo }}'
  exports:
    - label_count
```

### skip_validation

`true` makes `build` and `test` skip validation checks for this resource. Use it for resources first created with placeholder values that a later step in the same stack updates (for example an IAM role with a placeholder external ID, or circular dependencies).

### skip_on_delete

`true` means `teardown` does not process the resource; no effect on `build` or `test`.

- `query` type: the query is not run. Each declared export is set to `<unknown>`, and downstream queries that would interpolate it are skipped.
- `resource` and `multi` types: exports are still collected (a downstream `delete` may need them) but the `delete` query is not run, so the resource is kept.

During teardown any `exists`, `statecheck`, `exports`, `delete` or inline `sql` query whose rendered text contains `<unknown>` is skipped automatically, so `skip_on_delete` is never required for teardown to succeed. Use it for build-only gates (readiness polls, assertions) and for resources that must survive teardown (a bucket holding data, a shared resource only asserted by the stack).

### return_vals

Captures fields from a `RETURNING *` response as `this.*` variables, per operation (`create`, `update`, `delete`):

```yaml
- name: example_vpc
  return_vals:
    create:
      - Identifier: identifier
      - ErrorCode
  exports:
    - vpc_id
```

- `SourceField: target_name` makes `SourceField` available as `{{ this.target_name }}`.
- `FieldName` makes it available as `{{ this.FieldName }}`.
- Captured values are mutable: a later `update` can overwrite what `create` set.
- If a listed field is missing from the response (or the DML has no `RETURNING *`), the build fails.

Without `return_vals`, `RETURNING *` rows are still logged and available to callbacks as `callback.*`, but nothing is added as `this.*`.

## Stack exports

The top-level `exports` list names context variables (globals or resource exports) to output after a successful `build` or `test`. A listed name that does not exist fails the operation. `stack_name`, `stack_env` and `elapsed_time` are added automatically.

Exports are printed as a table and written to `.stackql-deploy-exports` in the current directory as `export KEY='value'` lines, with keys `STACKQL_DEPLOY__<stack_name>__<stack_env>__<variable>` (hyphens in the stack name or environment become underscores). With `--output-file <path>` they are also written as JSON. Lists and objects are kept as JSON.

```bash
source .stackql-deploy-exports
echo $STACKQL_DEPLOY__my_stack__dev__workspace_url
```

```powershell
Get-Content .stackql-deploy-exports | ForEach-Object { Invoke-Expression $_ }
```

## Example excerpt

From the source's Google example (the full file has more resources):

```yaml
version: 1
name: kubernetes-the-hard-way
description: stackql-deploy example for kubernetes-the-hard-way
providers:
  - google
globals:
- name: project
  description: google project name
  value: "{{ GOOGLE_PROJECT }}"
- name: region
  value: australia-southeast1
resources:
- name: network
  description: vpc network for k8s-the-hard-way sample app
  props:
  - name: vpc_name
    description: name for the vpc
    value: "{{ stack_name }}-{{ stack_env }}-vpc"
  exports:
  - vpc_name
  - vpc_link
- name: subnetwork
  props:
  - name: subnet_name
    value: "{{ stack_name }}-{{ stack_env }}-{{ region }}-subnet"
  - name: ip_cidr_range
    values:
      prd:
        value: 192.168.0.0/16
      dev:
        value: 10.240.0.0/24
  exports:
    - subnet_name
    - subnet_link
- name: internal_firewall
  file: firewalls.iql
  props:
  - name: fw_name
    value: "{{ stack_name }}-{{ stack_env }}-allow-internal-fw"
  - name: fw_allowed
    value: [{IPProtocol: tcp}, {IPProtocol: udp}, {IPProtocol: icmp}]
- name: get_controller_instances
  type: query
  exports:
    - controller_instances
exports:
  - vpc_name
  - vpc_link
  - subnet_name
```

The complete, runnable stack used by this skill is `assets/example-github-labels/stackql_manifest.yml`.
