# Query Before Mutation

Companion demo for the tutorial **Query before you mutate: how agents should touch your infrastructure**.

The example demonstrates a simple infrastructure control loop using Google Cloud Storage:

1. Query live state
2. Identify resources that differ from policy
3. Apply bounded policy gates
4. Mutate only non-compliant resources
5. Query again to verify convergence
6. Skip mutation when the resource is already compliant

## Scenario

The policy for this demo requires GCS buckets to use a customer-managed Cloud KMS key (CMEK).

Buckets where `encryption` is `NULL` are using Google-managed encryption.

The demo queries the live GCS API, identifies buckets that do not satisfy the policy, checks location and resource-count gates, applies CMEK, and verifies the resulting state.

## Requirements

- Docker
- Google Cloud project
- Service account credentials with permission to read and update Cloud Storage buckets
- Existing Cloud KMS key
- stackql

## Provider

This demo uses:

```text
google.storage.buckets
```

Pull the validated Google provider version:

```sql
REGISTRY PULL google v26.07.00432;
```

## Run the demo

Follow the SQL statements in:

```text
queries.sql
```

Before running them, replace:

- `your-project-id`
- `demo-app-bucket1`
- `your-ring`
- `your-key`

with values from your Google Cloud environment.

The flow is:

```text
live query -> policy check -> bounded gate -> mutation -> verification -> repeat safely
```

The important property is that every execution begins by querying current cloud state.

A previous state snapshot can describe what was managed before. The live API describes what exists now.

## Agent usage

The same pattern can be exposed to an agent through the StackQL MCP server.

An MCP-capable client can use StackQL to query the same live cloud APIs and reason over the results before taking an allowed action.

The SQL walkthrough in this folder keeps the underlying query, gate, mutation, and verification steps explicit.


