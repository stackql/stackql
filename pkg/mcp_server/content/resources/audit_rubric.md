---
name: stackql_audit_rubric
description: Shared severity, drift-class and deletion-confidence definitions used by the built-in audit, drift, exposure and cost prompts.
---
# Audit rubric

Shared definitions so reports from different prompts and runs stay comparable. Every finding cites evidence: the query used and the key output values that support the claim.

## Severity (security and audit findings)

- **critical** - exploitable now or exposing data publicly: public storage with sensitive content, admin credentials at risk, sensitive ports open to the internet on reachable hosts. Act today.
- **high** - clear misconfiguration with significant risk but no confirmed exposure: encryption disabled, MFA absent on privileged principals, over-broad IAM on sensitive services. Remediate this cycle.
- **medium** - hardening gap or policy violation with mitigating context: stale credentials still constrained by other controls, audit-logging gaps.
- **low** - hygiene and informational: naming, tagging, minor deviations from best practice.

## Drift classes (IaC comparison)

- **missing** - declared in the intent, absent in the cloud.
- **unmanaged** - live in the cloud, absent from the intent.
- **divergent** - present in both with attribute differences on declared attributes only; provider defaults are not drift.
- **unverifiable** - declared but not checkable with granted permissions or available methods; always record the reason.

## Deletion confidence (cost cleanup)

- **safe** - orphaned by definition: unattached volume, unassociated address, snapshot of a deleted source.
- **probable** - idle by strong signal (long-stopped instance, empty target group) but worth an owner check.
- **investigate** - inference from metadata only (oversizing, low-traffic guesses).

## Reporting conventions

- A check that cannot run is reported as "could not assess" with the reason (typically missing permissions), never silently dropped.
- Monetary figures are always labelled as estimates with their pricing assumptions stated; unpriceable items are listed as "unpriced", never invented.
- Completeness is never asserted over a scope that was skipped; skipped clouds, regions, projects and subscriptions are enumerated in a coverage appendix.
