---
name: cost_cleanup
description: Agent-driven read-only FinOps cleanup pass via the stackql MCP tools; finds idle and orphaned resources, estimates monthly savings, and returns a prioritised safe-to-delete list with a dollar figure.
arguments:
  - name: clouds
    description: Clouds to review (eg "aws", "aws, azure"). Blank reviews every cloud with resolvable credentials.
    required: false
  - name: min_monthly_savings
    description: Minimum estimated monthly saving (in USD) for an item to appear in the main list. Blank includes everything and rolls small items into an "other" line.
    required: false
---
Perform a read-only FinOps cleanup review using the stackql MCP tools, then deliver a prioritised savings report with an estimated monthly dollar figure. Use only metadata tools and `run_select_query` - never `run_mutation_query` or `run_lifecycle_operation`, and never echo credential or secret values. You identify candidates and estimate savings; you never delete anything.

Requested cloud scope: "{{clouds}}" (blank = every cloud with resolvable credentials among `aws`, `google`, `azure`). Minimum monthly saving to itemise: "{{min_monthly_savings}}" USD (blank = itemise everything, rolling small items into an aggregate line).

Procedure:

1. **Session setup.** Call `server_info`, then `reload_credentials` to learn which providers resolve credentials. Review only in-scope clouds whose credentials resolve; record every skipped cloud and the reason.
2. **Provider availability.** Call `list_providers`; `pull_provider` any in-scope provider not yet installed.
3. **Scope discovery, per cloud.** AWS: enabled regions, swept per region. GCP: accessible projects. Azure: subscriptions. Fetch enumeration queries from the query library (`query_library_search` / `query_library_get`); the `stackql://docs/scope_discovery` resource carries the per-cloud strategy where the client exposes resources.
4. **Discover before querying.** For each new resource type, use `list_methods` / `describe_method` to learn the select method and its **required** WHERE attributes; use `validate_select_query` on untested SQL. WHERE clauses that map to provider parameters are exact-match only; URL-encode embedded slashes as `%2F` where routes have consecutive path parameters.
5. **Waste checks.** For each in-scope cloud:
   - **Unattached block storage**: EBS volumes, persistent disks and managed disks with no attachment - capture size and type for pricing.
   - **Unassociated addresses**: elastic IPs, static external IPs and public IP resources not bound to anything.
   - **Stopped-but-billing compute**: stopped/deallocated instances still accruing storage and reserved addresses; note how long stopped where discoverable.
   - **Aged snapshots and images**: snapshots and custom images older than 90 days, especially those whose source volume/disk no longer exists.
   - **Idle load balancers and gateways**: load balancers with no healthy targets or no backends, NAT gateways in networks with no instances, where metadata allows the inference.
   - **Empty or oversized services**: empty clusters and node pools, obviously oversized instances or databases where the metadata alone supports the inference - mark these lower confidence.
6. **Estimate savings.** For each candidate, estimate the monthly cost from its size, type and region using published list prices you know; state assumptions (eg gp3 at ~$0.08/GB-month) and round sensibly. Label every figure an estimate; where you cannot price a resource, list it with "unpriced" rather than inventing a number.
7. **Rank by safety and value.** Assign each item a deletion confidence:
   - **Safe** - orphaned by definition (unattached volume, unassociated IP, snapshot of a deleted volume);
   - **Probable** - idle by strong signal (long-stopped instance, empty target group) but worth an owner check;
   - **Investigate** - inference from metadata only (oversizing, low-traffic guesses).
   Order the report by confidence first, then estimated monthly saving.
8. **Resilience.** A failed query is never a dead end: re-discover the method contract, fix required parameters, and retry; if a check remains unassessable (typically missing permissions), record it as "could not assess" with the reason and move on. Keep result sets bounded with `row_limit` where supported.
9. **Report.** Apply the deletion-confidence definitions from `stackql://docs/audit_rubric` where the client exposes resources. Deliver:
   - an executive summary leading with the total estimated monthly saving (and the safe-to-delete subtotal), in plain business language;
   - a cleanup table: cloud | region/project/subscription | service | resource | why it is waste | deletion confidence (safe/probable/investigate) | estimated monthly saving (USD, marked as estimate);
   - suggested next steps: the safe items as a candidate deletion checklist for a human to action, and the probable/investigate items as owner-confirmation questions;
   - a coverage appendix: clouds, regions, projects and subscriptions reviewed; clouds skipped and why; checks that could not be assessed; pricing assumptions used.

The review is read-only end to end; nothing you do changes or deletes any cloud resource - the deliverable is a list for humans to act on.
