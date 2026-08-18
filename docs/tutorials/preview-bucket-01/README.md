# Multi-cloud bucket audit (preview)

A one-command audit of storage buckets across AWS, GCP, and Azure using StackQL.
Reports encryption class, public flag, HTTPS enforcement, and other configuration
in a single normalised table.

This is preview material under `stackql_preview.*`. The view schema is still
evolving, so pin the Docker image (as this tutorial does) for reproducibility.

## What you'll need

- Docker installed
- Read-only credentials for whichever clouds you want to audit. All three are
  optional; blank credentials mean that cloud is skipped.

For the specific IAM permissions per provider, see
[required-auth.md](https://github.com/stackql/stackql-audit-action/blob/main/docs/required-auth.md)
in the `stackql-audit-action` repo.

## Run it

**1. Copy the env template**

```bash
cp ./audit/.env.audit.example ./audit/.env.audit
```

Open `./audit/.env.audit` and fill in credentials for whichever clouds you want.
The variables are named for what they are: `AWS_ACCESS_KEY_ID`,
`AWS_SECRET_ACCESS_KEY`, `AWS_REGION`, `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`,
`AZURE_CLIENT_SECRET`, `GOOGLE_CREDENTIALS` (the service account JSON, on one
line), and `GOOGLE_ORG_ID`.

**2. Pull the pinned StackQL Docker image**

```bash
docker compose -f docker-compose.bucket.audit.yaml pull
```

**3. Run the audit**

```bash
docker compose -f docker-compose.bucket.audit.yaml run --rm stackql
```

On a modest account this takes about 30 seconds. Output goes straight to your
terminal as a table.

## What the output tells you

A single normalised table with a row per bucket, columns for provider, name,
encryption class, public flag, HTTPS enforcement, region, and versioning. Three
different underlying resource types (S3 buckets, GCS buckets, Azure Storage
Accounts) unified into one view with the same columns for each.

## What this covers today

Storage buckets across AWS, GCP, and Azure. One AWS region at a time, one GCP
organization at a time. Entitlements, IAM, and other resource types are on the
roadmap as separate audits under `stackql_preview.*`.

## Where to go next

- Move it into CI: [stackql-audit-action](https://github.com/stackql/stackql-audit-action) has ready-to-use GitHub Actions workflows for all-clouds, single-cloud, deep-audit, and OIDC-authenticated variants.
- Ask questions: [StackQL community Slack](https://join.slack.com/t/stackqlcommunity/shared_invite/zt-46ndqydvn-X8ip8b9xgkT__IOTFbMlVg).
- Read the full tutorial: [Auditing three clouds without writing three scripts](https://stackql.io/blog/) *(link updates once published)*.
