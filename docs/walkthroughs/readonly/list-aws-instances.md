
## Setup

First, for whichever AWS user you would like to use, grant read only privileges on EC2 (eg: using `arn:aws:iam::aws:policy/ReadOnlyAccess`).  Then, create a set of AWS CLI credentials per [the AWS documentation](https://docs.aws.amazon.com/cli/latest/userguide/cli-authentication-user.html#cli-authentication-user-get), and store them in the appropriate environment variables `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`.

Then, do this in bash:

```bash setup stackql-shell app_root_path=./test/tmp/.list-aws-instances.stackql

stackql shell --approot=<<app_root_path>>  --registry="{ \"url\": \"file://$(pwd)/test/registry\", \"localDocRoot\": \"$(pwd)/test/registry\", \"verifyConfig\": { \"nopVerify\": true } }"
```

## Method

Do this in the `stackql` shell, replacing the tuple of regions with whichever AWS regions hold interest for you (these are not templated in the example):

```sql stackql-shell


SELECT instance_id, region
FROM aws.ec2_nextgen.instances
WHERE region IN ('us-east-1', 'ap-southeast-2', 'eu-west-1');

```

## Result


Assuming you have chosen regions wisely, you will see something like this included in the output:

```sql stackql stdout expectation stdout-table-contains-data
|---------------------|----------------|
|     instance_id     |     region     |
|---------------------|----------------|
| i-some-silly-id-011 | us-east-1      |
|---------------------|----------------|
| i-some-other-id-011 | ap-southeast-2 |
|---------------------|----------------|
```

## Cleanup

```bash teardown best-effort app_root_path=./test/tmp/.list-aws-instances.stackql

rm -rf <<app_root_path>>;

echo "teardown complete";

```