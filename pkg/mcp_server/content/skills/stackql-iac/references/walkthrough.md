# IaC walkthroughs on AWS

Source: condensed from `docs/iac_walkthrough.md` and `cicd/examples/aws-nginx/README.md` in the
stackql repository. Machine-specific paths are replaced with placeholders such as
`<omnisdk-checkout>`. Notes about older Omni releases are omitted: the ledger read and recovery
statements used below are available in the build this skill ships with.

Both flows create real, billable AWS resources in `ap-southeast-2` (Sydney) using the AWS
credentials in the stackql process environment. The aws-nginx README states that its stackql steps
have not been verified against live AWS.

## VPC and subnet from an Omni example specification

This flow uses the example specification files from an Omni checkout and a prepared registry
overlay that supplies the update routes the example needs.

### Start

Run from a fixed working directory:

```sh
stackql \
  --preview '{"unstable":true}' \
  --registry '{"localDocRoot":"<omnisdk-checkout>/cicd/out/aws-update-registry","srcPrefix":"","verifyConfig":{"nopVerify":true}}' \
  shell
```

Keep the same working directory, collection, state and region throughout. The ledger is kept under
`.stackql/iac/demo`; do not remove that directory before teardown. The statements omit `run_id`,
so stackql generates a unique one per execution. They create a new deployment; they do not adopt
resources from an earlier Omni CLI deployment.

### Create and rerun

```sql
INSERT INTO stackql_preview.iac.run (resources_file, collection, state, region)
VALUES ('<omnisdk-checkout>/cicd/examples/aws-vpc-subnet/resources.json', 'stackql-vpc-demo', 'demo', 'ap-southeast-2')
RETURNING *;
```

Check for errors before continuing and record the VPC and subnet IDs. In the AWS console (Sydney)
the VPC CIDR is `10.42.0.0/16` and the subnet CIDR is `10.42.1.0/24`. The initial file assigns no
VPC Name tag.

Run the same SQL again: both IDs must stay the same, with no duplicate resources.

### Update and rerun

```sql
INSERT INTO stackql_preview.iac.run (resources_file, collection, state, region)
VALUES ('<omnisdk-checkout>/cicd/examples/aws-vpc-subnet/update-public-ip.json', 'stackql-vpc-demo', 'demo', 'ap-southeast-2')
RETURNING *;
```

The same VPC now has Name `omni-demo-vpc`, and the same subnet has automatic public IPv4
assignment enabled. No instances or internet routing are created.

Run this update SQL again to check unchanged convergence. An `applied` status can mean no-op;
verify the IDs are unchanged. Continue with this file from now on: reapplying `resources.json` is
not an unchanged rerun, because it drops fields maintained by the update document.

### Reverse the subnet setting

```sql
INSERT INTO stackql_preview.iac.run (resources_file, collection, state, region)
VALUES ('<omnisdk-checkout>/cicd/examples/aws-vpc-subnet/update-private-ip.json', 'stackql-vpc-demo', 'demo', 'ap-southeast-2')
RETURNING *;
```

Automatic public IPv4 assignment is now disabled and the VPC Name remains `omni-demo-vpc`. This
file is now the latest desired-state document and can be repeated.

### Destroy

```sql
DELETE FROM stackql_preview.iac.run
WHERE collection = 'stackql-vpc-demo'
  AND state = 'demo'
  AND region = 'ap-southeast-2'
RETURNING *;
```

Inspect the results for errors or outstanding resources, and confirm both recorded IDs are gone
from AWS. The subnet is deleted before its VPC. If an operation fails, keep the ledger and inspect
the returned diagnostics; do not switch collection names or erase state to get past the error.

### After a failed run that compensated

A failed run that reports its resources compensated and none outstanding leaves tombstones for
those keys in the ledger. Verify the old IDs are absent in AWS before running the create step
again. Do not delete the ledger or its journals.

## nginx VM with a provider overlay and an inputs file

The spec files live in `cicd/examples/aws-nginx/` of a stackql repository checkout: the resource
specification (`resources.json`), a provider overlay (`ec2.merge.json`), a stopped-state
specification (`stopped.json`) and a bootstrap script (`user-data.sh`). No Omni checkout is
needed. The deployment is a VPC, two subnets, routing, a security group, an SSH key pair and an
nginx VM. Run every command from the stackql repository root; paths are relative to it.

Use your existing pinned canonical AWS registry. The session merge augments that registry; do not
point stackql at Omni's test corpus or its generated effective registry.

### Prepare inputs

```sh
export CASE="$PWD/.stackql/aws-nginx"
export SSH_CIDR="$(curl --fail --silent --show-error https://checkip.amazonaws.com)/32"
python3 cicd/examples/aws-nginx/prepare.py \
  --case "$CASE" --prefix my-nginx-demo --region ap-southeast-2 \
  --zone-a ap-southeast-2a --zone-b ap-southeast-2b \
  --ssh-cidr "$SSH_CIDR"
```

- Choose availability zones available to your account.
- The default image is Amazon Linux 2023 x86_64, resolved once through AWS's public SSM parameter
  (needs the AWS CLI and `ssm:GetParameter`). `--image-id <ami-id>` skips the lookup. The concrete
  AMI ID is stored in `inputs.json`, so launches never use a moving image reference.
- A unique client token is generated once per prepared deployment and stored with the AMI. Retries
  and reapplies reuse both. Preparation refuses to overwrite an existing `inputs.json`. Do not
  rotate a token while an earlier launch has an unresolved outcome.
- HTTP is public; SSH is restricted to the supplied CIDR. `--http-cidr` restricts HTTP.

### Create

Open stackql with your usual registry configuration plus the preview flag:

```sh
stackql shell --preview '{"unstable":true}'
```

In that session, merge the overlay and converge:

```sql
SET "$.document_merge" = '@.stackql/aws-nginx/merge.json';

INSERT INTO stackql_preview.iac.run (collection, region, resources_file, inputs_file)
VALUES ('my-nginx-demo', 'ap-southeast-2', 'cicd/examples/aws-nginx/resources.json', '.stackql/aws-nginx/inputs.json')
RETURNING *;
```

Use the collection and region chosen during preparation. Run the same INSERT again to verify an
unchanged reapply; each execution gets a fresh run ID.

### Read managed identities

```sql
SELECT resource_key, identity, phase
FROM stackql_preview.iac.ledger
WHERE collection = 'my-nginx-demo' AND resource_key = 'instance';
```

Reads use the same state root as the converge, without locks, journals or cloud calls. A pending
row's identity is the previous committed identity, not confirmation of the pending operation.
Explicit `Name` tags are display labels only; ownership and recovery come from the ledger.

### Query the IP and connect over SSH

In the same merged session, set `instance_id` to the `identity` returned for the `instance` row:

```sql
SET instance_id = '<identity returned for instance>';

SELECT id, public_ip,
       CONCAT('ssh -i .stackql/aws-nginx/id_ed25519 ec2-user@', public_ip) AS ssh_command
FROM stackql_unstable_aws.ec2.web_instance
WHERE region = 'ap-southeast-2'
  AND "InstanceId.1" = $instance_id;
```

Run the returned `ssh_command` locally. Preparation generated the private key at
`.stackql/aws-nginx/id_ed25519` and imported its public key into EC2; if preparation used
`--public-key`, use the matching private key instead. `public_ip` also serves the HTTP `/healthz`
check. A running instance does not mean the nginx bootstrap has finished.

### Stop and destroy

Declare the VM stopped, then destroy the collection in dependency order:

```sql
INSERT INTO stackql_preview.iac.run (collection, region, resources_file, inputs_file)
VALUES ('my-nginx-demo', 'ap-southeast-2', 'cicd/examples/aws-nginx/stopped.json', '.stackql/aws-nginx/inputs.json')
RETURNING *;

-- Only after the stopped-state result succeeds:
DELETE FROM stackql_preview.iac.run
WHERE collection = 'my-nginx-demo'
  AND region = 'ap-southeast-2'
RETURNING *;

-- After teardown, restore the session's canonical document view:
UNSET "$.document_merge";
```

- The ledger and journals default to `.stackql/iac` under the working directory, or under
  `--approot`. The merge cache normally defaults to `.stackql/cache/document-views`; the prepared
  `merge.json` selects its own case-local `registry-cache`.
- Keep the same app root, collection and overlay for every run. A reopened session needs the
  `SET` again before continuing. Paths refer to the stackql process's filesystem.
- For an existing deployment, set `iacStateRoot` to its original ledger directory before
  continuing; a different ledger can create duplicates.
- Inspect failed or outstanding results before any further mutation.

### Recover a pending create, then resume

First establish the original operation's actual outcome. For a create independently verified as
absent and no longer in flight:

```sql
SELECT resource_key, phase, version FROM stackql_preview.iac.ledger
WHERE collection = 'my-nginx-demo' AND resource_key = 'subnet_a';

UPDATE stackql_preview.iac.recover
SET outcome = 'absent'
WHERE collection = 'my-nginx-demo'
  AND resource_key = 'subnet_a'
  AND version = '1'
RETURNING *;
```

Use the version returned by the first query; `1` is only the first-create case. Then repeat the
original converge INSERT with the same collection, state, inputs and resource specification.
Existing resources are retained. Recovery changes local ledger state only, records the operator
assertion durably and checks the pending version under an exclusive lock. Absent recovery is
refused when there is prior committed state or a dependent. If the create completed, use
`outcome = 'applied'` with the verified `identity`; verify completion and readiness, not only
existence. Tags are not evidence. There is no automatic replay of an uncertain mutation.

### Offline preparation checks

```sh
python3 cicd/examples/aws-nginx/test_prepare.py
```

## Running these flows through the stackql MCP server

Start the server with `--preview '{"unstable":true}'` (plus the same `--registry` or `--approot`
as above) and send the statements with `run_mutation_query`; send ledger reads with
`run_select_query`. Each tool call is a fresh session, so put `SET "$.document_merge" = ...;` and
the statement that needs it in one `sql` batch, and repeat the `SET` in every later call that
needs the overlay. File paths are resolved on the server host. See `SKILL.md` for how each
statement is gated by the server mode.
