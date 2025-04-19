
# Testing approach

## Contrived local integration testing

Offline invocations of `stackql` are assessed against expected responses, through:

1. the functionality of [/test/deprecated/python/main.py](/test/deprecated/python/main.py).
2. [robot tests in /test/functional](/test/functional)  

(1) is deprecated and will be entirely migrated to (2).

These tests are run during the build process:
  - locally as per [/README.md#build](/README.md#build)
  - in github actions based CICD as per [/.github/workflows/go.yml](/.github/workflows/go.yml).

## Unit tests using standard golang approaches

Proliferation is a fair way behind development.

These are also run inside build processes: local and remote.

## E2E integration tests

TBA.


## Sundry opinions about testing in golang

  - [Simple approach and dot import.](https://medium.com/@benbjohnson/structuring-tests-in-go-46ddee7a25c)
  - [Making use of containers, make and docker compose for integration testing.](https://blog.gojekengineering.com/golang-integration-testing-made-easy-a834e754fa4c)
  - [HTTP client testing.](http://hassansin.github.io/Unit-Testing-http-client-in-Go)
  - [Mocking HTTPS in unit tests.](https://stackoverflow.com/questions/27880930/mocking-https-responses-in-go)

## Benchmarks

### Benchmarks with `go test`

```bash
go test -run='^$' -bench . -count=3 ./... > cicd/log/current-bench.txt

```

### Manual benchmarks

First, you will need to start aws mocks as per [the mock server example](/test/mockserver/README.md).

```bash
export workspaceFolder="/path/to/repository/root"

cd $workspaceFolder/build

## experiment
time ./stackql exec --registry="{ \"url\": \"file://${workspaceFolder}/test/registry-mocked\", \"localDocRoot\": \"${workspaceFolder}/test/registry-mocked\", \"verifyConfig\": { \"nopVerify\": true } }" --auth="{ \"google\": { \"credentialsfilepath\": \"${workspaceFolder}/test/assets/credentials/dummy/google/functional-test-dummy-sa-key.json\" } }" --tls.allowInsecure "select  instanceId,  ipAddress  from aws.ec2.instances  where  instanceId not in ('some-silly-id')   and region in (   'ap-southeast-2',    'ap-southeast-1',   'ap-northeast-1',   'ap-northeast-2',   'ap-south-1',   'ap-east-1',   'ap-northeast-3',   'eu-central-1',   'eu-west-1',   'eu-west-2',   'eu-west-3',   'eu-north-1',   'sa-east-1',   'us-east-1',   'us-east-2',   'us-west-1',   'us-west-2'  ) ;"

## control
time ./stackql exec --registry="{ \"url\": \"file://${workspaceFolder}/test/registry-mocked\", \"localDocRoot\": \"${workspaceFolder}/test/registry-mocked\", \"verifyConfig\": { \"nopVerify\": true } }" --auth="{ \"google\": { \"credentialsfilepath\": \"${workspaceFolder}/test/assets/credentials/dummy/google/functional-test-dummy-sa-key.json\" } }" --tls.allowInsecure "select  instanceId,  ipAddress  from aws.ec2.instances  where  instanceId not in ('some-silly-id')   and region =  'ap-southeast-2' ;"

```