# AWS VPC and subnet through StackQL

Run from the StackQL repository root with your AWS credentials already configured. This walkthrough
creates real resources in `ap-southeast-2` (Sydney). It uses the existing Omni example files and prepared
registry on this machine; the registry overlay supplies the update routes required by the example.

## Start

```sh
./build/stackql \
  --preview '{"unstable":true}' \
  --registry '{"localDocRoot":"/Users/generalkroll/stackql-labs/omnisdk/cicd/out/aws-update-registry","srcPrefix":"","verifyConfig":{"nopVerify":true}}' \
  shell
```

Keep the same working directory, collection, state and region throughout. StackQL retains the ledger
under `.stackql/iac/demo`. Do not remove that directory before teardown. These commands omit `run_id`:
StackQL generates a unique one for each execution. They create a new deployment rather than adopting
resources from a previous Omni CLI deployment.

## Create and rerun

Paste into the StackQL shell:

```sql
SELECT * FROM stackql_preview.iac.run
WHERE resources_file = '/Users/generalkroll/stackql-labs/omnisdk/cicd/examples/aws-vpc-subnet/resources.json'
  AND collection = 'stackql-vpc-demo'
  AND state = 'demo'
  AND region = 'ap-southeast-2';
```

Check for errors before continuing. Record the VPC and subnet IDs. In the AWS console, select Sydney
and find those IDs: the VPC CIDR is `10.42.0.0/16`, and the subnet CIDR is `10.42.1.0/24`. The initial
file does not assign a VPC Name tag.

Run that exact SQL again now: both IDs must stay the same, with no duplicate resources.

## Update and rerun

```sql
SELECT * FROM stackql_preview.iac.run
WHERE resources_file = '/Users/generalkroll/stackql-labs/omnisdk/cicd/examples/aws-vpc-subnet/update-public-ip.json'
  AND collection = 'stackql-vpc-demo'
  AND state = 'demo'
  AND region = 'ap-southeast-2';
```

Verify the same VPC now has Name `omni-demo-vpc`, and the same subnet has automatic public IPv4
assignment enabled. This does not create instances or internet routing.

Run this update SQL again to check unchanged convergence. A successful `applied` status can mean
no-op; verify the IDs remain unchanged. **Continue using this file after the update.** Reapplying
`resources.json` is not an unchanged rerun: it drops fields maintained by the update document.

## Reverse the subnet setting

```sql
SELECT * FROM stackql_preview.iac.run
WHERE resources_file = '/Users/generalkroll/stackql-labs/omnisdk/cicd/examples/aws-vpc-subnet/update-private-ip.json'
  AND collection = 'stackql-vpc-demo'
  AND state = 'demo'
  AND region = 'ap-southeast-2';
```

Verify automatic public IPv4 assignment is disabled. The VPC Name remains `omni-demo-vpc`.
You can repeat this same SQL; it is now the latest desired-state document.

## Destroy

```sql
SELECT * FROM stackql_preview.iac.destroy
WHERE collection = 'stackql-vpc-demo'
  AND state = 'demo'
  AND region = 'ap-southeast-2';
```

Inspect the results for errors or outstanding resources, and confirm both recorded IDs disappear
from AWS. The subnet is deleted before its VPC. If an operation fails, retain the ledger and inspect
the returned diagnostics; do not switch collection names or erase state to bypass the error.

## After the earlier reused-run failure

That failure reported both resources compensated and no outstanding resources. The local ledger
contains tombstones for both keys. Verify the old IDs are absent in AWS before starting the create
step again with the fixed binary. Do not delete the ledger or its old journals.
