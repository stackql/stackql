---
name: cloud_audit_quickstart
description: Quickstart for running the read-only cross-cloud (AWS/GCP/Azure + Entra) security and FinOps audit via docker compose.
---
Run a read-only cross-cloud (AWS / GCP / Azure + Entra) security & FinOps audit with one command, from the root of the stackql repository:

```bash
cp cicd/audit/.env.audit.example cicd/audit/.env.audit
# fill in read-only creds for the cloud(s) you want to audit
docker compose -f docker-compose.audit.yaml pull          # get the standard stackql image
docker compose -f docker-compose.audit.yaml run --rm audit
```

Pin the image for reproducibility by setting `STACKQL_IMAGE` (e.g. `STACKQL_IMAGE=stackql/stackql:v0.5.888`) in `cicd/audit/.env.audit`; it defaults to `stackql/stackql:latest`.

Findings land in `cicd/audit/output/findings.json`. A cloud you don't configure is simply skipped.

What it runs - two stock images (`docker-compose.audit.yaml`):

- `stackql` (`stackql/stackql`) - runs as a Postgres-wire **server** holding your read-only `--auth`.
- `audit` (`python:3.11-slim`) - downloads the audit engine and drives the queries against the server.

The audit engine - the check `queries/`, `filters.py`, and `scripts/discover.py` (S3 deep scan, AWS region sweep, GCP org descent, Azure management-group descent) - is **not** vendored in the stackql repository. `run.sh` downloads it at run time from https://github.com/stackql/stackql-audit-action (`cicd/audit/quickstart/`), pinned by `AUDIT_ENGINE_REF` in `.env.audit`. The action repo is the single source of truth; the stackql repo carries only the thin quickstart (compose + `run.sh` + docs).

Configure credentials by editing `cicd/audit/.env.audit` - fill in the plain credential values for the cloud(s) you have. A cloud left blank is skipped; no JSON to assemble.

- **AWS** - `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` (+ `AWS_SESSION_TOKEN` for temporary creds) and `AWS_REGION`.
- **GCP** - `GCP_SA_JSON` (whole SA key JSON, inline on one line); `GOOGLE_ORG_ID` (digits) to sweep the whole org's projects.
- **Azure / Entra** - `AZURE_TENANT_ID` / `AZURE_CLIENT_ID` / `AZURE_CLIENT_SECRET` (+ optional `AZURE_MGMT_GROUP` to scope; blank = whole tenant).

Required read-only roles per cloud: https://github.com/stackql/stackql-audit-action/blob/main/docs/required-auth.md
