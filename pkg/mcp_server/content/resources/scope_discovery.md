---
name: stackql_scope_discovery
description: Tested StackQL queries for enumerating audit scope - AWS regions and buckets, GCP org/folder/project descent, Azure management groups and subscriptions.
---
# Scope discovery queries

Tested enumeration queries for establishing cloud scope before an audit-style sweep. Substitute placeholder values; every `WHERE` predicate shown maps to a request input and is exact-match. If a query fails against your provider version, fall back to the discovery tools (`list_methods` / `describe_method`) rather than guessing variants.

## AWS

Enabled regions (needs one seed region for routing; exclude regions with `optInStatus = 'not-opted-in'` from subsequent sweeps):

```sql
SELECT regionName, optInStatus FROM aws.ec2_native.regions WHERE region = 'us-east-1';
```

S3 buckets, cheap enumeration then per-bucket detail (the list_only/keyed pairing):

```sql
SELECT bucket_name, region FROM aws.s3.buckets_list_only WHERE region = 'us-east-1';

SELECT bucket_name, region, public_access_block_configuration, bucket_encryption,
       versioning_configuration, ownership_controls
FROM aws.s3.buckets WHERE region = 'us-east-1' AND data__Identifier = 'my-bucket';
```

## GCP

Organization descent - list folders and projects under a parent, recursing into each `ACTIVE` folder. `parent` takes the form `organizations/<org_id>` or `folders/<folder_id>`:

```sql
SELECT name, parent, state FROM google.cloudresourcemanager.folders WHERE parent = 'organizations/123456789012';

SELECT projectId, parent, state FROM google.cloudresourcemanager.projects WHERE parent = 'organizations/123456789012';
```

Audit only projects with `state = 'ACTIVE'`.

## Azure

Tenant-wide subscriptions (audit only `state = 'Enabled'`):

```sql
SELECT subscriptionId, state FROM azure.subscription.subscriptions;
```

Management-group descent, when scoping to a management group:

```sql
SELECT id, name, type FROM azure.management_groups.descendants WHERE groupId = 'my-mgmt-group';
```

## Other providers

No canned queries are maintained for GitHub, Okta or Entra scope discovery; use the discovery workflow (`list_services` / `list_resources` / `list_methods`) to find the membership or tenant enumeration resource for the provider version in use.
