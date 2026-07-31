---
name: iam_access_review
description: Agent-driven read-only identity and access review across AWS IAM, Entra ID, Okta and GitHub via the stackql MCP tools; returns an auditor-ready access review (SOC2/ISO style).
arguments:
  - name: providers
    description: Identity providers to review (eg "aws", "aws, entra_id, github"). Blank reviews every supported provider with resolvable credentials (aws, entra_id, okta, github).
    required: false
  - name: stale_days
    description: Age in days beyond which credentials and inactivity count as stale. Blank defaults to 90.
    required: false
---
Perform a read-only identity and access review using the stackql MCP tools, then deliver an auditor-ready report of the kind requested in SOC2 and ISO 27001 access reviews. Use only metadata tools and `run_select_query` - never `run_mutation_query` or `run_lifecycle_operation`, and never echo credential or secret values (key IDs and ages are fine; key material never).

Requested provider scope: "{{providers}}" (blank = every supported identity provider with resolvable credentials: `aws`, `entra_id`, `okta`, `github`). Staleness threshold: "{{stale_days}}" days (blank = 90).

Procedure:

1. **Session setup.** Call `server_info`, then `reload_credentials` to learn which of the in-scope identity providers resolve credentials. Review only those that resolve; record every skipped provider and the reason.
2. **Provider availability.** Call `list_providers`; `pull_provider` any in-scope provider not yet installed.
3. **Discover before querying.** For each new resource type, use `list_methods` / `describe_method` to learn the select method and its **required** WHERE attributes; use `validate_select_query` on untested SQL. WHERE clauses that map to provider parameters are exact-match only; URL-encode embedded slashes as `%2F` where routes have consecutive path parameters.
4. **Inventory principals, per provider in scope.**
   - AWS: IAM users, roles, groups and their attached/inline policies.
   - Entra ID: users (member vs guest), directory role assignments, app registrations and service principals.
   - Okta: users and their status, group memberships, administrator role assignments.
   - GitHub: organisation members and their role, outside collaborators, teams; enumerate the orgs the token can see.
5. **Over-privilege checks.** Flag principals with more access than their apparent function needs: AWS policies granting `Action: *` on `Resource: *` or attached AdministratorAccess, broad wildcard actions on sensitive services (iam, kms, s3); Entra users holding Global Administrator or other privileged directory roles; Okta super admins; GitHub org owners. Report privileged-role counts against the population (eg 9 owners out of 40 members).
6. **Credential hygiene checks.** Stale access keys and keys never rotated past the threshold (AWS `access_keys` create date, last-used where discoverable); users without MFA where the provider exposes it; Entra app credentials expired or expiring; principals inactive past the threshold (last sign-in / password-last-used); GitHub members without 2FA where the token can see enforcement data.
7. **External and dormant access checks.** Entra guest accounts and their role assignments; GitHub outside collaborators and the repos they reach; AWS roles never used or with trust policies allowing foreign accounts; Okta deactivated-but-not-deleted users; service accounts or app principals with no discernible owner.
8. **Resilience.** A failed query is never a dead end: re-discover the method contract, fix required parameters, and retry; if a check remains unassessable (typically missing permissions or API scope), record it as "could not assess" with the reason and move on. Keep result sets bounded with `row_limit` where supported.
9. **Report.** Apply the severity definitions from `stackql://docs/audit_rubric` where the client exposes resources. Deliver:
   - an executive summary: overall access posture, headline numbers (privileged principals, stale credentials, guests/outside collaborators, MFA gaps), and the top actions, in plain business language;
   - a findings table: provider | principal | principal type (user/role/service/guest/collaborator) | finding | severity (critical/high/medium/low) | evidence | recommendation;
   - a privileged-access register: every principal holding an admin-equivalent role, per provider - this is the artifact auditors ask for by name;
   - a coverage appendix: providers and orgs/tenants/accounts reviewed, providers skipped and why, and checks that could not be assessed with the granted permissions.

The review is read-only end to end; nothing you do changes any account, role, key or membership.
