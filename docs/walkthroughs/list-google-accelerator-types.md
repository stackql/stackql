
## Setup

First, create a google service account key using the GCP Console, per [the GCP documentation](https://cloud.google.com/iam/docs/keys-create-delete).  Grant the service account at least `Viewer` role equivalent privileges, per [the GCP documentation](https://cloud.google.com/iam/docs/create-service-agents#grant-roles).

Then, do this in bash:

```bash setup stackql-shell credentials_path=cicd/keys/testing/google-ro-credentials.json app_root_path=./test/tmp/.get-google-accel.stackql

export GOOGLE_CREDENTIALS="$(cat <<credentials_path>>)";

stackql shell --approot=<<app_root_path>>
```

## Method

Do this in the `stackql` shell, replacing `<<project>>` with your GCP project name, and `<<zone>>` as desired, eg: `australia-southeast1-a`:

```sql stackql-shell input required project=stackql-demo zone=australia-southeast1-a

registry pull google;

select 
  name, 
  kind 
FROM google.compute.accelerator_types 
WHERE 
  project = '<<project>>' 
  AND zone = '<<zone>>'
ORDER BY
  name desc
;

```

## Result


You will see exactly this included in the output:

```sql expectation stdout-contains-all
|---------------------|-------------------------|
|        name         |          kind           |
|---------------------|-------------------------|
| nvidia-tesla-t4-vws | compute#acceleratorType |
|---------------------|-------------------------|
| nvidia-tesla-t4     | compute#acceleratorType |
|---------------------|-------------------------|
| nvidia-tesla-p4-vws | compute#acceleratorType |
|---------------------|-------------------------|
| nvidia-tesla-p4     | compute#acceleratorType |
|---------------------|-------------------------|
```

## Cleanup

```bash teardown best-effort app_root_path=./test/tmp/.get-google-accel.stackql

rm -rf <<app_root_path>>

```