---
name: cloud_audit
description: Agent-driven read-only cross-cloud (AWS/GCP/Azure + Entra) security and FinOps audit using the stackql MCP tools; returns a findings report.
arguments:
  - name: clouds
    description: Clouds to audit (eg "aws", "aws, azure"). Blank audits every cloud with resolvable credentials.
    required: false
  - name: focus
    description: Audit focus - "security", "finops", or blank for both.
    required: false
---
Perform a read-only cross-cloud security and FinOps audit using the stackql MCP tools, then deliver a findings report. The operator has configured least-privilege read-only credentials; you do the rest. Use only metadata tools and `run_select_query` - never `run_mutation_query` or `run_lifecycle_operation`, and never echo credential or secret values.

Requested cloud scope: "{{clouds}}" (blank = every cloud with resolvable credentials). Focus: "{{focus}}" (blank = both security and FinOps).

Procedure:

1. **Session setup.** Call `server_info`, then `reload_credentials` to learn which providers resolve credentials (relevant providers: `aws`, `google`, `azure`, `entra_id`). Audit only in-scope clouds whose credentials resolve; record every skipped cloud and the reason.
2. **Provider availability.** Call `list_providers`; `pull_provider` any in-scope provider not yet installed.
3. **Scope discovery, per cloud.**
   - AWS: enumerate enabled regions and sweep per region; scan all S3 buckets.
   - GCP: enumerate accessible projects (descend from the organization when one is reachable) and audit per project.
   - Azure: enumerate subscriptions (descend management groups when configured) and audit per subscription; identity checks via `entra_id`.
4. **Discover before querying.** For each new resource type, use `list_methods` / `describe_method` to learn the select method and its **required** WHERE attributes; use `validate_select_query` on untested SQL. WHERE clauses that map to provider parameters are exact-match only; URL-encode embedded slashes as `%2F` where routes have consecutive path parameters.
5. **Security checks** (when in focus): publicly accessible storage (S3 buckets, GCS buckets, Azure blob containers); network exposure (security groups / firewall rules open to 0.0.0.0/0 on sensitive ports, public IP attachments); encryption at rest disabled (volumes, disks, storage accounts, snapshots); IAM hygiene (users without MFA, stale access keys, over-broad role bindings, Entra guest accounts and apps with expiring/expired secrets); audit-logging gaps where discoverable.
6. **FinOps checks** (when in focus): unattached volumes/disks; unassociated static/elastic IPs; stopped instances still accruing storage; aged snapshots; obviously oversized or idle resources where metadata allows an inference.
7. **Resilience.** A failed query is never a dead end: re-discover the method contract, fix required parameters, and retry; if a check remains unassessable (typically missing permissions), record it as "could not assess" with the reason and move on. Keep result sets bounded with `row_limit` where supported.
8. **Report.** Deliver:
   - an executive summary: overall posture, top risks, top savings opportunities, in plain business language;
   - a findings table: cloud | service | resource | finding | severity (critical/high/medium/low) | evidence | recommendation;
   - a coverage appendix: clouds, regions, projects and subscriptions audited; clouds skipped and why; checks that could not be assessed with the granted permissions.

The audit is read-only end to end; nothing you do changes any cloud resource.
