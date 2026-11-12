# AWS nginx deployment through stackql

The [resource spec](resources.json), [provider overlay](ec2.merge.json), [stopped-state spec](stopped.json)
and [bootstrap script](user-data.sh) are included here. No Omni checkout or generated SQL is required.
The spec provisions a VPC, two subnets, routing, security group, SSH key pair and nginx VM.


Start with **your existing pinned canonical AWS registry** in stackql. The session merge augments
that registry; do not point stackql at Omni's test corpus or its generated effective registry.
Requires stackql's `document_merge` and `inputs_file` support, consuming Omni `v0.1.11-alpha01`.
These stackql steps have not been verified against live AWS.

Run from the **stackql repository root**. Prepare local SSH keys and nginx bootstrap inputs
(the default AMI lookup is read-only):

```sh
export CASE="$PWD/.stackql/aws-nginx"
export SSH_CIDR="$(curl --fail --silent --show-error https://checkip.amazonaws.com)/32"
python3 cicd/examples/aws-nginx/prepare.py \
  --case "$CASE" --prefix my-nginx-demo --region ap-southeast-2 \
  --zone-a ap-southeast-2a --zone-b ap-southeast-2b \
  --ssh-cidr "$SSH_CIDR"
```

Choose availability zones available to your account. The default image is Amazon Linux 2023 x86_64,
resolved once during preparation through AWS's public SSM parameter. The concrete AMI ID is
persisted in `inputs.json`; launches never use a moving image reference. This default lookup requires
the AWS CLI and `ssm:GetParameter`. `--image-id` with a concrete AMI skips it. Deployment requires
the EC2 permissions. Creates billable AWS resources. HTTP is public; SSH is
restricted to the supplied CIDR. Use `--http-cidr` to restrict HTTP.

A unique client token is generated once per prepared deployment and persisted alongside the AMI.
Retries/reapply reuse both; preparation refuses to overwrite an existing `inputs.json`. A new case
gets a new token even with the same display prefix. Do not rotate a token while an earlier launch
has an unresolved outcome.

Open stackql using your usual pinned registry configuration, adding the preview flag:

```sh
./build/stackql shell --preview '{"unstable":true}'
```

Inside that session, merge the overlay and create the deployment:

```sql
SET "$.document_merge" = '@.stackql/aws-nginx/merge.json';

SELECT * FROM stackql_preview.iac.run
WHERE collection = 'my-nginx-demo'
  AND region = 'ap-southeast-2'
  AND resources_file = 'cicd/examples/aws-nginx/resources.json'
  AND inputs_file = '.stackql/aws-nginx/inputs.json';
```

Use the collection and region chosen during preparation. AWS credentials use the normal environment.
Run the same SELECT again to verify unchanged reapply. Each execution gets a fresh run ID automatically.

## Read managed identities

```sql
SELECT resource_key, identity, phase
FROM stackql_preview.iac.ledger
WHERE collection = 'my-nginx-demo' AND resource_key = 'instance';
```

Requires an Omni release with the new read-only ledger API (not yet in `v0.1.11-alpha01`). Reads
use the same app/state root as apply, without locks, journals or cloud calls. A pending row's
identity is the previous committed identity, not confirmation of the pending operation.
Explicit `Name` tags are display labels only; ownership and recovery come from the ledger.

## Query the IP and connect over SSH

Set the SQL variable `instance_id` to the `identity` returned for the collection's `/instance` row
(`SET instance_id = 'the actual returned identity';`). Then, in the same merged session:

```sql
SELECT id, public_ip,
       CONCAT('ssh -i .stackql/aws-nginx/id_ed25519 ec2-user@', public_ip) AS ssh_command
FROM stackql_unstable_aws.ec2.web_instance
WHERE region = 'ap-southeast-2'
  AND "InstanceId.1" = $instance_id;
```

Run the returned `ssh_command` in your local terminal. Preparation generated the private key at
`.stackql/aws-nginx/id_ed25519`; its public key was imported into EC2. The private key is local,
not fetched from AWS. If preparation used `--public-key`, substitute its corresponding private key
path. The query returns `public_ip` for the HTTP `/healthz` check too. EC2 running does not guarantee
nginx bootstrap has completed.

## Stop and destroy

Declare the VM stopped, then destroy the collection in dependency order:

```sql
SELECT * FROM stackql_preview.iac.run
WHERE collection = 'my-nginx-demo'
  AND region = 'ap-southeast-2'
  AND resources_file = 'cicd/examples/aws-nginx/stopped.json'
  AND inputs_file = '.stackql/aws-nginx/inputs.json';

-- Execute only after the stopped-state result succeeds.
SELECT * FROM stackql_preview.iac.destroy
WHERE collection = 'my-nginx-demo'
  AND region = 'ap-southeast-2';

-- After teardown, restore the session's canonical document view.
UNSET "$.document_merge";
```

Ledger and journals default to `.stackql/iac` under the working directory (or `--approot`). Merge cache
normally defaults to `.stackql/cache/document-views`; preparation's `merge.json` explicitly selects
its case-local `registry-cache`. Keep the same app root, collection and overlay for every run.
Reopening a session requires repeating SET before continuing. Paths refer to the stackql process.

For an existing deployment, configure `iacStateRoot` to its original ledger directory before
continuing. A different ledger can create duplicates. Inspect failed/outstanding results before
any further mutation.

## Recover a pending create, then resume

First establish the original operation's actual outcome. For an independently verified absent
create that is no longer in flight:

```sql
SELECT resource_key, phase, version FROM stackql_preview.iac.ledger
WHERE collection = 'my-nginx-demo' AND resource_key = 'subnet_a';

SELECT * FROM stackql_preview.iac.recover
WHERE collection = 'my-nginx-demo'
  AND resource_key = 'subnet_a'
  AND outcome = 'absent'
  AND version = '1';
```

Use the version returned by the first query; `1` is only the first-create example. Then repeat the
original apply SELECT with the same collection, state, inputs and resource spec. Existing resources
are retained. Recovery updates local ledger state only, records the operator assertion durably and
checks the pending version under an exclusive lock. It refuses prior committed state or dependents
for absent recovery. If the create completed, use `outcome = 'applied'` and the verified `identity`;
verify completion/readiness, not merely existence. Tags are not ownership or recovery evidence.
Requires an Omni release after `v0.1.11-alpha03` containing recovery. Older releases return an explicit
upgrade error. There is no automatic replay of an uncertain mutation.

## Offline preparation checks

```sh
python3 cicd/examples/aws-nginx/test_prepare.py
```
