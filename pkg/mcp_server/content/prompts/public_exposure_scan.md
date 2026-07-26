---
name: public_exposure_scan
description: Agent-driven read-only fast scan for internet-reachable resources (public storage, open security groups, public IPs, public DB endpoints, exposed k8s services) via the stackql MCP tools; returns a prioritised exposure table.
arguments:
  - name: clouds
    description: Clouds to scan (eg "aws", "aws, azure"). Blank scans every cloud with resolvable credentials.
    required: false
---
Perform a fast, read-only scan for everything reachable from the internet using the stackql MCP tools, then deliver a prioritised exposure report. This is a narrow, high-signal sweep - the "am I bleeding" check - not a full audit: stay on exposure checks only, keep queries bounded, and prefer breadth across clouds over depth on any one resource. Use only metadata tools and `run_select_query` - never `run_mutation_query` or `run_lifecycle_operation`, and never echo credential or secret values.

Requested cloud scope: "{{clouds}}" (blank = every cloud with resolvable credentials among `aws`, `google`, `azure`).

Procedure:

1. **Session setup.** Call `server_info`, then `reload_credentials` to learn which providers resolve credentials. Scan only in-scope clouds whose credentials resolve; record every skipped cloud and the reason.
2. **Provider availability.** Call `list_providers`; `pull_provider` any in-scope provider not yet installed.
3. **Scope discovery, per cloud.** AWS: enabled regions (S3 is global - scan all buckets). GCP: accessible projects. Azure: subscriptions. If the client exposes MCP resources, read `stackql://docs/scope_discovery` for tested enumeration queries. Sweep every region/project/subscription in scope but keep each check to the single cheapest listing query that answers it.
4. **Discover before querying.** For each new resource type, use `list_methods` / `describe_method` to learn the select method and its **required** WHERE attributes; use `validate_select_query` on untested SQL. WHERE clauses that map to provider parameters are exact-match only; URL-encode embedded slashes as `%2F` where routes have consecutive path parameters.
5. **Exposure checks.** For each in-scope cloud, in this order:
   - **Public storage**: S3 buckets with public access (block-public-access off, public ACLs/policies), GCS buckets with allUsers/allAuthenticatedUsers bindings, Azure storage accounts/containers allowing public blob access.
   - **Open network paths**: security groups / firewall rules / NSGs allowing 0.0.0.0/0 or ::/0 inbound - flag any-port and sensitive ports (22, 3389, 3306, 5432, 1433, 6379, 27017, 9200, 5601) as critical, other ports as review items.
   - **Public IP attachments**: instances, NAT-less VMs and network interfaces with public IPs, and load balancers with internet-facing frontends - cross-reference against the open rules above to identify actually reachable endpoints.
   - **Public database endpoints**: RDS instances/clusters with `publicly_accessible`, Cloud SQL instances with public IP, Azure SQL / PostgreSQL / MySQL servers with public network access or allow-all firewall rules; equivalent for managed caches and search where cheap to check.
   - **Exposed Kubernetes and container endpoints**: clusters with public API endpoints (EKS/GKE/AKS), services of type LoadBalancer with external addresses where discoverable.
   - **Public snapshots and images**: EBS snapshots, AMIs and equivalent images shared publicly, where a single listing query can answer it.
6. **Resilience.** A failed query is never a dead end: re-discover the method contract, fix required parameters, and retry once; in this fast scan, if a check still fails, record it as "could not assess" with the reason and keep moving rather than digging. Keep result sets bounded with `row_limit` where supported.
7. **Report.** Apply the severity definitions from `stackql://docs/audit_rubric` where the client exposes resources. Deliver:
   - a one-paragraph verdict up top: how much is internet-reachable and whether anything demands action today;
   - an exposure table sorted most severe first: cloud | region/project/subscription | service | resource | exposure (what is reachable and how) | severity (critical/high/medium/low) | recommendation;
   - a coverage appendix: clouds, regions, projects and subscriptions scanned; clouds skipped and why; checks that could not be assessed.

The scan is read-only end to end; nothing you do changes any cloud resource.
