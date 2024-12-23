
## Setup

First, create a google service account key using the GCP Console, per [the GCP documentation](https://cloud.google.com/iam/docs/keys-create-delete).  Grant the service account at least `Viewer` role equivalent privileges, per [the GCP dumentation](https://cloud.google.com/iam/docs/create-service-agents#grant-roles).

Then, do this in bash:

```bash setup stackql-shell credentials-path=cicd/keys/testing/google-ro-credentials.json app-root-path=./test/tmp/.get-google-vms.stackql

export GOOGLE_CREDENTIALS="$(cat <credentials-path>)";

stackql shell --approot=<app-root-path>
```

## Method

Do this in the `stackql` shell, replacing `<project>` with your GCP project name, and `<zone>` as desired, eg: `australia-southeast1-a`:

```sql stackql-shell input required project=stackql-demo zone=australia-southeast1-a

registry pull google;

select 
  name, 
  id 
FROM google.compute.instances 
WHERE 
  project = '<project>' 
  AND zone = '<zone>'
;

```

## Result


You will see something very much like this included in the output, presuming you have one VM (if you have zero, only the headers should appper, more VMs means more rows):

```sql stackql stdout expectation stdout-table-contains-data
|--------------------------------------------------|---------------------|
|                       name                       |         id          |
|--------------------------------------------------|---------------------|
| any-compute-cluster-1-default-abcd-00000001-0001 | 1000000000000000001 |
|--------------------------------------------------|---------------------|
```

<!---  EXPECTATION
google\ provider,\ version\ 'v24.11.00274'\ successfully\ installed
goodbye
-->

<x-expectation style="display: none;">
<stdout-contains-nonempty-table></stdout-contains-nonempty-table>
</x-expectation>

## Cleanup

```bash teardown best-effort app-root-path=./test/tmp/.get-google-vms.stackql

rm -rf <app-root-path>

```