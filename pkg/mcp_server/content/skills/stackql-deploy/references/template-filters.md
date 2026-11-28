Source: github.com/stackql/stackql-deploy-rs, tag v2.4.0, website/docs/template-filters.md (MIT, StackQL Studios). Condensed. Corrections for v2.4.0 are checked against src/template/engine.rs and Cargo.lock (Tera 1.20.0) at the same tag, and against the CHANGELOG.md entry on the default branch after v2.4.0 that fixes the `tojson` documentation.

# Template filters

`.iql` files and manifest values are rendered with Tera, a Jinja2-like engine: `{{ expr }}` for values, `{% for %}`, `{% if %}` and `{% set %}` for logic, `|` for filters. stackql-deploy adds the custom filters below to Tera's built-in filters.

## JSON output: json_encode, not tojson

The filter that serializes a value to JSON is Tera's built-in `json_encode`. There is no `tojson` filter in v2. `tojson` was a Jinja2 built-in in the Python v1, and the v2.4.0 docs (and the k8s-the-hard-way example) still show it, but a template that uses it fails to render with `Filter 'tojson' not found`. Every `tojson` in the source docs is written as `json_encode` below.

## Values reach templates as text

Context values are passed to Tera as strings. Props and globals whose values are lists or maps arrive as compact JSON text, so:

- To insert one as a JSON literal, quote it directly: `'{{ subnet_tags }}'`. Do not add `json_encode`, which would encode the text a second time (`"{\"k\":\"v\"}"`).
- To loop over one, index into it, or pass it to `merge_lists` / `merge_objects`, parse it first with `from_json`.
- Use `json_encode` on values that are already structured: the output of `from_json`, `merge_lists` or `merge_objects`, or a loop variable.

## Filter arguments are keyword arguments

Tera only accepts named filter arguments (`filter(name=value)`). A positional call such as `merge_lists(custom_policies)`, which the source docs show, fails to parse with `expected a keyword argument`. The custom merge filters read the argument named `other`.

## Custom filters

These are registered by stackql-deploy in v2.4.0, in addition to Tera's built-ins.

### from_json

Parses a JSON string into a list or object, typically to iterate:

```sql
{% for network_interface in network_interfaces | from_json %}
INSERT INTO google.compute.instances
 (
  /* fields... */
 )
 SELECT
'{{ instance_name_prefix }}-{{ loop.index }}',
/* other values... */
'[ {{ network_interface | json_encode }} ]';
{% endfor %}
```

The v2.4.0 examples also use it with Tera's built-in `get`: `{{ root_bucket_info | from_json | get(key="bucket_name") }}`.

### generate_patch_document

Builds an RFC 6902 patch document, mainly for AWS Cloud Control API updates, which require one:

```sql
update aws.s3.buckets
set PatchDocument = string('{{ {
    "NotificationConfiguration": transfer_notification_config
    } | generate_patch_document }}')
WHERE
region = '{{ region }}'
AND Identifier = '{{ bucket_name }}';
```

### base64_encode

Base64-encodes a string, for API fields that take binary data:

```sql
INSERT INTO aws.ec2.instances (
 /* fields... */
 UserData,
 region
)
SELECT
 /* values... */
 '{{ user_data | base64_encode }}',
 '{{ region }}';
```

### sql_list

Turns a list, or a JSON array string, into a quoted SQL list for `IN` clauses:

```sql
SELECT * FROM aws.ec2.instances
WHERE region = '{{ region }}'
AND InstanceId IN {{ instance_ids | sql_list }}
```

### sql_escape

Doubles single quotes so a string can sit inside a SQL string literal, for example SQL nested inside a statement:

```sql
INSERT INTO snowflake.sqlapi.statements (
statement,
/* other fields... */
)
SELECT
'{{ statement | sql_escape }}',
/* other values... */
;
```

### merge_lists

Merges two lists into one list of unique items. Both sides must be lists, so parse JSON-text props with `from_json` first:

```sql
{% set combined_policies = default_policies | from_json | merge_lists(other=custom_policies | from_json) %}
INSERT INTO aws.iam.policies (
  /* fields... */
  PolicyDocument
)
SELECT
  /* values... */
  '{{ combined_policies | json_encode }}'
;
```

### merge_objects

Merges two objects; on duplicate keys the value from `other` wins. Same argument and parsing rules as `merge_lists`:

```sql
{% set complete_config = base_config | from_json | merge_objects(other=environment_specific_config | from_json) %}
INSERT INTO aws.lambda.functions (
  /* fields... */
  Environment
)
SELECT
  /* values... */
  '{{ complete_config | json_encode }}'
;
```

### to_aws_tag_filters

AWS-specific. Converts a list of tag pairs (as in `global_tags`) to the Resource Groups Tagging API `TagFilters` shape, for `awscc.tagging.tagged_resources` queries.

- Input: `[{"Key": "k", "Value": "v"}, ...]`
- Output: `[{"Key": "k", "Values": ["v"]}, ...]`

```sql
/*+ exists */
SELECT split_part(ResourceARN, '/', 2) as vpc_id
FROM awscc.tagging.tagged_resources
WHERE region = '{{ region }}'
AND TagFilters = '{{ global_tags | to_aws_tag_filters }}'
AND ResourceTypeFilters = '["ec2:vpc"]'
```

```yaml
globals:
  - name: global_tags
    value:
      - Key: 'stackql:stack-name'
        Value: "{{ stack_name }}"
      - Key: 'stackql:stack-env'
        Value: "{{ stack_env }}"
      - Key: 'stackql:resource-name'
        Value: "{{ resource_name }}"
```

## Built-in variables

Injected automatically, no manifest configuration needed:

- `stack_name` - the manifest `name`.
- `stack_env` - the environment argument (`dev`, `sit`, `prd`, ...).
- `resource_name` - the resource currently being processed.
- `idempotency_token` - UUID v4 generated once per resource per invocation and stable across retries. Use `{{ this.idempotency_token }}` inside the resource's own `.iql` and `{{ <resource_name>.idempotency_token }}` from a later resource. See `references/resource-query-files.md`.

## uuid()

`{{ uuid() }}` returns a new random UUID v4 on every render, for example `'{{ stack_name }}-{{ uuid() }}'` as a unique name. Because it changes on each render it is not safe in queries that may be retried; use `this.idempotency_token` there.

## Chaining

Filters apply left to right:

```sql
'{{ user_config | from_json | merge_objects(other=default_config | from_json) | json_encode | base64_encode }}'
```
