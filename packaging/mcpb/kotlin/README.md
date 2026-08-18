# stackql-mcp-kotlin

[![ci](https://github.com/stackql/stackql-mcp-kotlin/actions/workflows/ci.yml/badge.svg)](https://github.com/stackql/stackql-mcp-kotlin/actions/workflows/ci.yml)

The JVM member of the StackQL embedded-MCP family. StackQL exposes cloud providers (AWS, GitHub, Google, Azure, and more) as SQL tables; this library acquires the `stackql` binary, launches it as an MCP server over stdio, and hands you a connected client built on the official [MCP Kotlin SDK](https://github.com/modelcontextprotocol/kotlin-sdk).

It ships with `costgate`, a demo that makes cloud cost a build check: a Gradle plugin and a CLI that price a declared resource intent before it exists and fail the build when it blows the budget.

## Modules

| Module | Coordinate | What it is |
|---|---|---|
| `stackql-mcp` | `io.stackql:stackql-mcp` (Maven Central) | The library: acquire, launch, connect |
| `costgate` | - | Demo core: price an intent, gate on a budget |
| `costgate-cli` | - | costgate as a standalone command |
| `costgate-gradle` | `io.stackql.costgate` (Gradle Plugin Portal) | costgate as a `costgate` Gradle task |

## Library quickstart

```kotlin
import io.stackql.mcp.LaunchArgs
import io.stackql.mcp.Mode
import io.stackql.mcp.StackqlMcp

suspend fun main() {
    val server = StackqlMcp.builder()
        .mode(Mode.ReadOnly)
        .auth(LaunchArgs.authFor("github", "null_auth")) // github needs no creds
        .start()

    server.use {
        val tools = server.client.listTools().tools
        println("${tools.size} tools available")
    }
}
```

`StackqlMcp.builder().start()` acquires the server binary (see Acquisition), spawns it as an MCP stdio server with the canonical launch arguments, completes the handshake, and returns a connected client. `close()` (here via `use`) shuts the session down and terminates the process. `stdout` belongs to the MCP protocol; the server's `stderr` is forwarded to `System.err` (override with `onStderr`).

Dependency (once published):

```kotlin
dependencies {
    implementation("io.stackql:stackql-mcp:0.1.0")
}
```

## Acquisition

The library resolves a runnable `stackql` binary in this order, first match wins:

1. `STACKQL_MCP_BIN` env - run that binary directly
2. `Builder.binary(path)` - same, from code
3. `STACKQL_MCP_BUNDLE` env - extract that local `.mcpb` (no pin check; an explicit override is operator intent)
4. `Builder.bundlePath(path)` - same, from code
5. Shared cache - an already-extracted binary for the pinned version
6. Verified download - fetch the pinned `.mcpb`, check its sha256 against the pins baked into the library, extract, cache. Subsequent starts are offline.

The binary cache is shared with the StackQL npm and PyPI MCP wrappers: `~/.stackql/mcp-server-bin/<version>/<platform>/`. Existing cache entries are used before any download, so a polyglot machine extracts once. Platforms: `linux-x64`, `linux-arm64`, `windows-x64`, `darwin-universal`.

## Safety modes

The server enforces a safety contract per session; the library defaults to the most restrictive.

| Mode | Allows |
|---|---|
| `Mode.ReadOnly` (default) | SELECT and metadata tools only |
| `Mode.Safe` | reads plus non-destructive mutations |
| `Mode.DeleteSafe` | safe plus deletes |
| `Mode.FullAccess` | everything, including lifecycle provisioning |

Escalation is an explicit caller opt-in via `.mode(...)`. The default `read_only` server refuses mutation calls server-side, so a confused agent cannot create anything in a planning phase.

## Bring your own MCP stack

`StackqlMcp.builder().commandLine()` resolves the exact command (binary path plus canonical args) without starting a session, so an external conformance harness - or your own process supervisor - can run the launcher directly. This is the same path the packaging repo's `smoke-test.py --cmd` mode exercises.

## Demo: costgate

Platform teams want deploys blocked when the infrastructure being shipped will blow the budget - before it exists. Today's answer is a post-hoc billing surprise; costgate makes cost a build check, like tests or lint.

```sh
# offline, no credentials: prices against a bundled rate-card snapshot
costgate check --intent examples/costgate.yaml --explain

# price against live provider data via an embedded stackql server (needs creds)
costgate check --intent costgate.yaml --live
```

The intent file declares what a deploy intends to create:

```yaml
provider: aws
budget: 500/month
resources:
  - name: api servers
    type: compute
    size: m5.large
    region: us-east-1
    count: 3
  - name: data volumes
    type: storage
    size: gp3
    region: us-east-1
    count: 1
    gb: 500
```

costgate prices each line, totals it, compares to the budget, and exits non-zero when over - the gate. `--explain` shows the SQL or rate-card rule behind each price and the top cost drivers. `--junit-xml <path>` writes a JUnit-style report so CI UIs render the result inline.

Pricing is pluggable: the live source runs a `read_only` SELECT against the provider's pricing surface through the embedded server; the bundled rate card is the offline, credential-free fallback (and what CI uses). The live source supersedes the card when credentials are present.

### As a Gradle task

```kotlin
plugins {
    id("io.stackql.costgate") version "0.1.0"
}

costgate {
    intent.set(layout.projectDirectory.file("costgate.yaml"))
    explain.set(true)
    // failOnOverBudget.set(false)  // soft check: report without blocking
}
```

```sh
gradle costgate   # fails the build when the intent is over budget
```

The report is written to `build/reports/costgate/costgate.xml`.

## Build and test

JDK 17, Kotlin 2.x, Gradle (wrapper checked in). The integration/conformance tests are excluded by default to keep the build hermetic; opt in with `-PrunIntegration=true`.

```sh
./gradlew build                              # all modules, unit tests
./gradlew :stackql-mcp:test -PrunIntegration=true \
    --tests "io.stackql.mcp.ConformanceTest" # downloads + pin-verifies the server
./gradlew :costgate-cli:installDist          # build the costgate command
```

CI runs the unit tests, the conformance sequence, and the offline costgate demo on Linux, macOS, and Windows for every PR to `main` and every release tag.

## Publishing

- Library: Maven Central via the Central Portal (`./gradlew :stackql-mcp:publish`, then release the staging repository in the portal - manual/2FA).
- Plugin: the Gradle Plugin Portal (`./gradlew :costgate-gradle:publishPlugins`).

Both expect signing and credentials in the environment (`SIGNING_KEY`, `SIGNING_PASSWORD`, `CENTRAL_USERNAME`, `CENTRAL_PASSWORD`, and the plugin portal key/secret).

## License

MIT. mcp-name reference: `io.github.stackql/stackql-mcp`.
