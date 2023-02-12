
# StackQL Developer Guide

## Contribution 

Please see [the contributing document](/CONTRIBUTING.md).

## Quick walkthrough

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

Examples are present [here](/examples/examples.md).


## Server mode

**Note that this feature is in alpha**.  We will update timelines for General Availability after a period of analysis and testing.  At the time of writing, server mode is most useful for R&D purposes: 
- experimentation.
- tooling / system integrations and design thereof.
- development of `stackql` itself.
- development of use cases for the product.

The `stackql` server leverages the `postgres` wire protocol and can be used with the `psql` client, including mTLS auth / encryption in transit.  Please see [the relevant examples](/examples/examples.md#running-in-server-mode) for further details.

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
  - Paral;elisation of "atomic" DML ops.


## Tests

Really, the github action files are the source of truth for build and test and we do encourage perusal of them.  However, to keep things brief, here below is the developers' brief for testing.

Requirements are [detailed in the root README](/README.md#system-requirements). 

Local testing of the application:

1. Run `go test --tags "json1 sqleanall" ./...` tests.
2. Build the executable [as per the root README](/README.md#build)
3. Perform registry rewrites as needed for mocking `python3 test/python/registry-rewrite.py`.
3. Run robot tests:
    - Functional tests, mocked as needed `robot -d test/robot/functional test/robot/functional`.
    - Integration tests `robot -d test/robot/integration test/robot/integration`.  For these, you will need to set various envirnonment variables as per the github actions.
4. Run the deprecated manual python tests:
    - Prepare with `cp test/db/db.sqlite test/db/tmp/python-tests-tmp-db.sqlite`.
    - Run with `python3 test/python/main.py`.

[This article](https://medium.com/cbi-engineering/mocking-techniques-for-go-805c10f1676b) gives a nice overview of mocking in golang.

### go test

Test coverage is sparse.  Regressions are mitigated by `go test` integration testing in the [driver](/internal/stackql/driver/driver_integration_test.go) and [stackql](/stackql/main_integration_test.go) packages.  Some testing functionality is supported through convenience functionality inside the [test](/internal/test) packages.

#### Point in time gotest coverage

If not already done, then install 'cover' with `go get golang.org/x/tools/cmd/cover`.  
Then: `go test --tags "json1 sqleanall" -cover ../...`.

### Functional and Integration testing

Automated functional and integration testing are done largely through robot framework.  Please see [the robot test readme](/test/robot/README.md).

There is some legacy, deprecated [manual python testing](/test/python/main.py) which will be migrated to robot and decommissioned.

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
~/Downloads/stackql --credentialsfilepath=$HOME/stackql/stackql-devel/keys/sa-key.json exec "select group_concat(substr(name, 0, 5)) || ' lalala' as cc from google.compute.disks where project = 'lab-kr-network-01' and zone = 'australia-southeast1-b';" -o text
```

## Profiling


```
time ./stackql exec --cpuprofile=./select-disks-improved-05.profile --auth='{ "google": { "credentialsfilepath": "'${HOME}'/stackql/stackql-devel/keys/sa-key.json" }, "okta": { "credentialsfilepath": "'${HOME}'/stackql/stackql-devel/keys/okta-token.txt", "type": "api_key" } } ' "select name from google.compute.disks where project = 'lab-kr-network-01' and zone = 'australia-southeast1-a';"
```

## Postgres Server Implementation

1. Heavy duty option as per cockroachdb:
    - https://github.com/cockroachdb/cockroach/tree/e6a0d23d516203bf5e8d1c8b3c3c26ddfaddc388/pkg/sql/pgwire
2. Light touch option can be based upon:
    - https://github.com/jeroenrinzema/psql-wire


## AWS HTTP request signing

https://docs.aws.amazon.com/sdk-for-go/api/aws/signer/v4/
