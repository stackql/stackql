# Preview functionality

> 🚨 **Preview functionality is not stable.** Behaviour, syntax and output may change or break between releases without notice. Do not rely on it in production.

Bleeding edge `stackql` functionality can be exposed using the `--preview` CLI argument.


```bash
./build/stackql shell --preview='{"unstable":true}'
```

For any provider you wish to consume, you will need to pull it, for example in support of what follows:

```sql
registry pull aws v26.08.00444;
registry pull azure v26.07.00420;
registry pull google v26.08.00446;
```

## Relevant preview functionality

Streaming high volume queries at low latency has relevance for audit and related use cases.

The examples below use `--output jsonl`; replace it with `--output otel` to emit one OpenTelemetry log record per row instead.

Every relation named `stackql_unstable_<provider>.<service>.<resource>` is resolved and run by `omnisdk` straight from the provider documents: which method each relation runs, which conditions become request parameters, which become edges between relations, and which become row filters. Rows are streamed to the output as they are produced; nothing is staged in the SQL backend.

```bash
_googleProject="stackql-demo" && \
./build/stackql exec --preview='{"unstable":true}' --output jsonl \
"select name, location, storageClass, timeCreated
   from stackql_unstable_google.storage.buckets
   where project = '${_googleProject}';"
```

### What streams, and what does not

- Use `--output jsonl` or `--output otel`. Each row is written and flushed as it arrives. The default table output holds every row until the query completes.
- `ORDER BY`, `GROUP BY`, `HAVING`, `DISTINCT` and aggregates are refused on these relations, since each needs every row before it can emit one.
- `LIMIT` without `ORDER BY` is pushed down and stops the requests early.
- Supported joins are `INNER JOIN` and `LEFT JOIN` with `ON`. A condition in `ON` that feeds a required input of the joined relation becomes a request per row of the left side.

### Credentials

`omnisdk` takes one credential per query, so each query below stays within one cloud.

| Provider | Credential |
|----------|------------|
| `aws` | `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` (and `AWS_SESSION_TOKEN` if temporary) |
| `azure` | service principal: `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET` |
| `google` | service account key JSON in `GOOGLE_CREDENTIALS`, or its path in `GOOGLE_APPLICATION_CREDENTIALS` |

## Audit queries

Each query below has been run against live accounts with the provider versions pinned above. Each is a single statement, so any number of them can be run concurrently, each streaming into its own sink.

### AWS

IAM users and their access keys (a user without one comes back without `AccessKeyId`):

```bash
_awsRegion="us-east-1" && \
./build/stackql exec --preview='{"unstable":true}' --output jsonl \
"select u.UserName, k.AccessKeyId, k.Status, k.CreateDate
from stackql_unstable_aws.iam.users u
left join stackql_unstable_aws.iam.access_keys k on k.UserName = u.UserName
where region = '${_awsRegion}';"
```

IAM users and their MFA devices (a user without one comes back without `SerialNumber`):

> **Under investigation:** against a live account every user has come back without `SerialNumber`; not yet confirmed whether that reflects the account or the join not passing `UserName`.

```bash
_awsRegion="us-east-1" && \
./build/stackql exec --preview='{"unstable":true}' --output jsonl \
"select u.UserName, m.SerialNumber, m.EnableDate
from stackql_unstable_aws.iam.users u
left join stackql_unstable_aws.iam.mfa_devices m on m.UserName = u.UserName
where region = '${_awsRegion}';"
```

Managed policies attached to each IAM user, and to each IAM role:

```bash
_awsRegion="us-east-1" && \
./build/stackql exec --preview='{"unstable":true}' --output jsonl \
"select u.UserName, u.PasswordLastUsed, p.PolicyName, p.PolicyArn
from stackql_unstable_aws.iam.users u
inner join stackql_unstable_aws.iam.attached_user_policies p on p.UserName = u.UserName
where region = '${_awsRegion}';"
```

```bash
_awsRegion="us-east-1" && \
./build/stackql exec --preview='{"unstable":true}' --output jsonl \
"select r.RoleName, r.AssumeRolePolicyDocument, p.PolicyName
from stackql_unstable_aws.iam.roles r
inner join stackql_unstable_aws.iam.attached_role_policies p on p.RoleName = r.RoleName
where region = '${_awsRegion}';"
```

### Azure

Role assignments in a subscription:

```bash
_azureSubscriptionId="$(az account show --query id -o tsv)" && \
./build/stackql exec --preview='{"unstable":true}' --output jsonl \
"select principalId, principalType, roleDefinitionId, scope, createdOn
from stackql_unstable_azure.authorization.role_assignments
where subscription_id = '${_azureSubscriptionId}';"
```

### Google

Bucket hardening:

```bash
_googleProject="stackql-demo" && \
./build/stackql exec --preview='{"unstable":true}' --output jsonl \
"select name, location, iamConfiguration, encryption, logging, versioning, retentionPolicy
from stackql_unstable_google.storage.buckets
where project = '${_googleProject}';"
```

IAM bindings on every bucket, one request per bucket:

```bash
_googleProject="stackql-demo" && \
./build/stackql exec --preview='{"unstable":true}' --output jsonl \
"select b.name as bucket, i.role, i.members
from stackql_unstable_google.storage.buckets b
inner join stackql_unstable_google.storage.buckets_iam_policies i on i.bucket = b.name
where b.project = '${_googleProject}';"
```

Project IAM bindings:

```bash
_googleProject="stackql-demo" && \
./build/stackql exec --preview='{"unstable":true}' --output jsonl \
"select role, members, condition
from stackql_unstable_google.cloudresourcemanager.projects_iam_policies
where projectsId = '${_googleProject}';"
```

Service account keys and their validity windows:

```bash
_googleProject="stackql-demo" && \
./build/stackql exec --preview='{"unstable":true}' --output jsonl \
"select s.email, s.disabled, k.name as key_name, k.keyType, k.validAfterTime, k.validBeforeTime
from stackql_unstable_google.iam.service_accounts s
inner join stackql_unstable_google.iam.service_account_keys k on k.serviceAccountsId = s.email
where s.projectsId = '${_googleProject}' and k.projectsId = '${_googleProject}';"
```
