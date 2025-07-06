
# StackQL Developer Guide

## Contribution 

Please see [the contributing document](/CONTRIBUTING.md).

## CICD

See [the CICD documentation](/docs/CICD.md).

## Developer quickstart

The short of things is that for basic build and unit testing, these are needed:

- Install `golang` on your system **if you do not already have version >= 1.21**, per [the `golang` doco](https://go.dev/doc/install).
- Install `python` on your system **if you do not already have version >= 3.11**, available from [the `python` website](https://www.python.org/downloads/) and numerous package managers.
- Using a `venv` or otherwise, install the requisite python packages, eg: (system permitting) from the repository root: `pip install -r cicd/requirements.txt`.

Then, each of these should be run from the repository root:

    - Get required `go` libs: `go get ./...`.
    - Run unit tests: `python cicd/python/build.py --test`.
    - Build the application: `python cicd/python/build.py --build` (this will generate an executable at `./build/stackql`).


For serious development, simulated integration tests are essential.  So, there are more dependencies:

- Install `psql`.  On some systems, this can be done as client only and/or with various package managers; fallback is to just [install postgres manually](https://www.postgresql.org/download/).

Having installed all dependencies, the `robot` tests should be run from the repository root directory (this relies upon the executable in `./build/stackql`, built above):
    - Run mocked functional tests: `python cicd/python/build.py --robot-test`.  This will subject the executable to the automated testing regimen.

Running linting locally is also a nice-to-have, and can be done:

- Install `golangci-lint` `v1.59.1` (the current version used in CI) per [the `golangci-lint` doco](https://golangci-lint.run/welcome/install/).
- From the root directory of the repository: `golangci-lint run`.


## xml

The inherent difficulty in generically serialising `xml` is nicely expressed by the golang dev community in [the `encoding/xml` documentation](https://pkg.go.dev/encoding/xml#pkg-note-BUG):

> Mapping between XML elements and data structures is inherently flawed: an XML element is an order-dependent collection of anonymous values, while a data structure is an order-independent collection of named values. See [encoding/json](https://pkg.go.dev/encoding/json) for a textual representation more suitable to data structures.

As of now, `stackql` handles `xml` SERDE through the core, and does not route this to SDKs.  Depending on priorities, this can be revisited *with care*.

## Building locally

```bash
env CGO_ENABLED=1 PLANCACHEENABLED=false go build \
  --tags "sqlite_stackql" \
  -ldflags "-X github.com/stackql/stackql/internal/stackql/cmd.BuildMajorVersion=${BUILDMAJORVERSION:-1} \
  -X github.com/stackql/stackql/internal/stackql/cmd.BuildMinorVersion=${BUILDMINORVERSION:-1} \
  -X github.com/stackql/stackql/internal/stackql/cmd.BuildPatchVersion=${BUILDPATCHVERSION:-1} \
  -X github.com/stackql/stackql/internal/stackql/cmd.BuildCommitSHA=$BUILDCOMMITSHA \
  -X github.com/stackql/stackql/internal/stackql/cmd.BuildShortCommitSHA=$BUILDSHORTCOMMITSHA \
  -X \"github.com/stackql/stackql/internal/stackql/cmd.BuildDate=$BUILDDATE\" \
  -X \"stackql/internal/stackql/planbuilder.PlanCacheEnabled=$PLANCACHEENABLED\" \
  -X github.com/stackql/stackql/internal/stackql/cmd.BuildPlatform=$BUILDPLATFORM" -o ./build ./stackql
```

## Testing locally

### Unit tests

At this time, we are not dogmatic about how to implement unit tests.  Aspirationally, unit tests can be implemented in similar fashion to the none-too opinionated [official testing package documentation](https://pkg.go.dev/testing), and in particular [the overview section](https://pkg.go.dev/testing#pkg-overview).

To run all unit tests:

```bash
go test -timeout 1200s --tags "sqlite_stackql" ./...
```

### Robot tests

**Note**: this requires the local build (above) to have been completed successfully, which builds a binary in `./build/`.

```bash
env PYTHONPATH="$PYTHONPATH:$(pwd)/test/python" robot -d test/robot/functional test/robot/functional
```

Or better yet, if you have docker desktop and the `postgres` image cited in the docker compose files:

```bash
robot --variable SHOULD_RUN_DOCKER_EXTERNAL_TESTS:true -d test/robot/functional test/robot/functional
```

### Manually Testing

Please see [the mock testing doco](/test/python/stackql_test_tooling/flask/README.md).


## Debuggers

The `vscode` tooling configuration is mostly ready to use, as seen in the `.vscode` directory.  You will need to create a file at the `.gitignore`d location `.vscode/.env`.  Simplest thing just copy the example to get going: `cp .vscode/example.env .vscode/.env`.

The debugger config is pretty messy, and probably with time we will slim it down.  That said, it is far from useless as an example.

## Provider development

Keen to expose some new functionality though `stackql`?  We are very keen on this!  

Please see [registry_contribution.md](/docs/registry_contribution.md).

## Provider Authentication

At this stage, authentication config must be specified for each provider, even for unauthorized ones.  Supported auth types are:

- `api_key`.
- `basic`.
- `interactive` for interactive oAuth, thus far only google supported via `gcloud` command line tool.
- `service_account` for json style private keys (eg: google service accounts).
- `null_auth` for unauthenticated providers.

If you want further auth types or discover bugs, please raise an issue.

Examples are present [here](/docs/examples/examples.md).


## Server mode

**Note that this feature is in alpha**.  We will update timelines for General Availability after a period of analysis and testing.  At the time of writing, server mode is most useful for R&D purposes: 
- experimentation.
- tooling / system integrations and design thereof.
- development of `stackql` itself.
- development of use cases for the product.

The `stackql` server leverages the `postgres` wire protocol and can be used with the `psql` client, including mTLS auth / encryption in transit.  Please see [the relevant examples](/docs/examples/examples.md#running-in-server-mode) for further details.

## Concurrency considerations

In server mode, a thread pool issues one thread to handle each connection.

The following are single threaded:

  - Lexical and Syntax Analysis.
  - Semantic Analysis.
  - Execution of a single, childless primitive. 
  - Execution of primitives a, b where a > b or b < a in the partial ordering of the plan DAG.  Although it need not be the same thread executing each, they will be strictly sequential.

The following are potentially multi threaded:

  - Plan optimization.
  - Execution of sibling primitives.

## Rebuilding Parser

Please consult [the parser repository](https://github.com/stackql/stackql-parser).


## Outstanding required Uplifts

### High level Tech debt / bugs for later

Really high level stuff:

  - Cache system -> db (redis????).

### Cache

  - Cache size limitations and rotation policy.
  - Cache persistence format from simple json -> db (redis????).
  - Re-use vitess LRU Cache???

### Data Model

  - Need reasoned view of tables / joins / rows.
  - Migrate repsonses to MySQL server type *a la* Vitess.
  - DML operations to use similar response filtering to metadata ops.

### Execution model

  - Failure modes and possible multiple errors... how to communicate cause and final state to user.  Need some overall philosophy that is extensible to transactions.
  - Need reasoned view of primitives and optimisations, default extensible method map approach.
  - Parallelisation of "atomic" DML ops.


## Tests

Really, the github action files are the source of truth for build and test and we do encourage perusal of them.  However, to keep things brief, here below is the developers' brief for testing.

Requirements are [detailed in the root README](/README.md#system-requirements). 

Local testing of the application:

1. Run `go test --tags "sqlite_stackql" ./...` tests.
2. Build the executable [as per the root README](/README.md#build)
3. Perform registry rewrites as needed for mocking `python3 test/python/stackql_test_tooling/registry_rewrite.py --srcdir "$(pwd)/test/registry/src" --destdir "$(pwd)/test/registry-mocked/src"`.
3. Run robot tests:
    - Functional tests, mocked as needed `robot -d test/robot/functional test/robot/functional`.
    - Integration tests `robot -d test/robot/integration test/robot/integration`.  For these, you will need to set various envirnonment variables as per the github actions.
4. Run the deprecated manual python tests:
    - Prepare with `cp test/db/db.sqlite test/db/tmp/python-tests-tmp-db.sqlite`.
    - Run with `python3 test/deprecated/python/main.py`.

[This article](https://medium.com/cbi-engineering/mocking-techniques-for-go-805c10f1676b) gives a nice overview of mocking in golang.

### go test

Test coverage is sparse.  Regressions are mitigated by `go test` integration testing in the [driver](/internal/stackql/driver/driver_integration_test.go) and [stackql](/stackql/main_integration_test.go) packages.  Some testing functionality is supported through convenience functionality inside the [test](/internal/test) packages.

#### Point in time gotest coverage

If not already done, then install 'cover' with `go get golang.org/x/tools/cmd/cover`.  
Then: `go test --tags "sqlite_stackql" -cover ../...`.

### Functional and Integration testing

Automated functional and integration testing are done largely through robot framework.  Please see [the robot test readme](/test/robot/README.md).

There is some legacy, deprecated [manual python testing](/test/deprecated/python/main.py) which will be migrated to robot and decommissioned.

### Linting

We use `golangci-lint`.

The linting of go files (and also Actions) for CI is defined in [.github/workflows/lint.yml](/.github/workflows/lint.yml).

To run the linter locally, first ensure you have the same version of `golangci-lint` as the CI and then either:

-  `golangci-lint run` to dump everything to console, or...
-  `golangci-lint run > cicd/log/lint.log 2>&1` to send all output to `cicd/log/lint.log` (w.r.t repository root).

## Cross Compilation locally

### From mac

In order to support windows compilation:

```
brew install mingw-w64
```

In order to support linux compilation:

```
export HOMEBREW_BUILD_FROM_SOURCE=1
brew install FiloSottile/musl-cross/musl-cross
```

## Testing latest build from CI system

### On mac

Download and unzip.  For the sake of example, let us consider the executable `~/Downloads/stackql`.

First:
```
chmod +x ~/Downloads/stackql
```

Then, on OSX > 10, you will need to whitelist the executable for execution even though it was not signed by an identifie developer.  Least harmful way to do this is try and execute some command (below is one candidate), and then open `System Settings` > `Security & Privacy` and there should be some UI to allow execution of the untrusted `stackql` file.  At least this works on High Sierra `v1.2.1`.

Then, run test commands, such as:
```
~/Downloads/stackql --credentialsfilepath=$HOME/stackql/stackql-devel/cicd/keys/sa-key.json exec "select group_concat(substr(name, 0, 5)) || ' lalala' as cc from google.compute.disks where project = 'lab-kr-network-01' and zone = 'australia-southeast1-b';" -o text
```

## Profiling


```
time ./stackql exec --cpuprofile=./select-disks-improved-05.profile --auth='{ "google": { "credentialsfilepath": "'${HOME}'/stackql/stackql-devel/cicd/keys/sa-key.json" }, "okta": { "credentialsfilepath": "'${HOME}'/stackql/stackql-devel/cicd/keys/okta-token.txt", "type": "api_key" } } ' "select name from google.compute.disks where project = 'lab-kr-network-01' and zone = 'australia-southeast1-a';"
```


## AWS HTTP request signing

https://docs.aws.amazon.com/sdk-for-go/api/aws/signer/v4/

## Selection from non-select DML

`INSERT RETURNING` can function in two mechanisms:

- Synchronous responses, such as [`google.storage.buckets`](https://cloud.google.com/storage/docs/json_api/v1/buckets/insert).  The returning clause is a projection on the immediately available reponse body.
- Asynchronous responses, such as [`google.compute.instances`](https://cloud.google.com/compute/docs/reference/rest/v1/instances/insert).  The returning clause is a projection on the reponse body **after** the await flow has concluded.

Future use cases for `UPDATE RETURNING`, `REPLACE RETURNING` and `DELETE RETURNING` will function the same observable fashion.

