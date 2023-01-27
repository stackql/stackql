
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

First, go to the root of the vitess repository.

```bash
make -C go/vt/sqlparser
```

If you need to add new AST node types, make sure to add them to [go/vt/sqlparser/ast.go](https://github.com/stackql/vitess/blob/feature/stackql-develop/go/vt/sqlparser/ast.go) and then regenerate the file [go/vt/sqlparser/rewriter.go](https://github.com/stackql/vitess/blob/feature/stackql-develop/go/vt/sqlparser/rewriter.go) as follows:

```
cd go/vt/sqlparser

go run ./visitorgen/main -input=ast.go -output=rewriter.go
```

## Outstading required Uplifts

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

## Notes on vitess

Vitess implements mysql client and sql driver interfaces.  The server backend listens over HTTP and RPC and implements methods for:

  - "Execute"; execute a simple, single query.
  - "StreamExecute"; tailored to execute a query returning a large result set.
  - "ExecuteBatch"; execution of multiple queries inside a txn.

Vitess maintains an LRU cache of query plans, searchable by query plaintext.  This model will likely work better for vitess thatn stackql; in the former routing is the main concern, in the latter "hotspots" in finer granularity is indicated.

If we do choose to leverage vitess' server implementation, we may implement the vitess vtgate interface [as per vtgate/vtgateservice/interface.go](https://github.com/stackql/vitess/blob/feature/stackql-develop/go/vt/vtgate/vtgateservice/interface.go).

### Low level vitess notes

The various `main()` functions:

  - [line 34 cmd/vtctld/main.go](https://github.com/stackql/vitess/blob/feature/stackql-develop/go/cmd/vtctld/main.go)
  - [line 52 cmd/vtgate/vtgate.go](https://github.com/stackql/vitess/blob/feature/stackql-develop/go/cmd/vtgate/vtgate.go) 
  - [line 106 cmd/vtcombo/main.go](https://github.com/stackql/vitess/blob/feature/stackql-develop/go/cmd/vtcombo/main.go)

...aggregate all the requisite setup for the server.

[Run(); line 33 run.go](https://github.com/stackql/vitess/blob/feature/stackql-develop/go/vt/servenv/run.go) sets up RPC and HTTP servers.

[Init(); line 133 in vtgate.go](https://github.com/stackql/vitess/blob/feature/stackql-develop/go/vt/vtgate/vtgate.go) initialises the VT server singleton.

Init() calls [NewExecutor(); line 108 in executor.go](https://github.com/stackql/vitess/blob/feature/stackql-develop/go/vt/vtgate/executor.go), a one-per-server object which includes an LRU cache of plans.

In terms of handling individual queries:

  - VTGate sessions [vtgateconn.go line 46](https://github.com/stackql/vitess/blob/feature/stackql-develop/go/vt/servenv/run.go) are passed in per request.
  - On the client side, [conn.Query(); line 284 in driver.go](https://github.com/stackql/vitess/blob/feature/stackql-develop/go/vt/vitessdriver/driver.go) calls (for example) `conn.session.StreamExecute()`.
  - Server side, [Conn.handleNextCommand(); line 759 mysql/conn.go](https://github.com/stackql/vitess/blob/feature/stackql-develop/go/mysql/conn.go)
  - Server side, vt software; [VTGate.StreamExecute(); line 301 in vtgate.go](https://github.com/stackql/vitess/blob/feature/stackql-develop/go/vt/vtgate/vtgate.go).
  - Then, (either directly or indirectly) [Executor.StreamExecute(); line 1128 in executor.go](https://github.com/stackql/vitess/blob/feature/stackql-develop/go/vt/vtgate/executor.go) handles synchronous `streaming` queries, and calls `Executor.getPlan()`.
  - [Executor.getPlan(); in particular line 1352 in executor.go](https://github.com/stackql/vitess/blob/feature/stackql-develop/go/vt/vtgate/executor.go)
is the guts of query processing.
  - [Build(); line 265 in builder.go](https://github.com/stackql/vitess/blob/feature/stackql-develop/go/vt/vtgate/planbuilder/builder.go) is the driver for plan building.


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
