# stackql-mcp (Kotlin/JVM)

[![mcp-packaging](https://github.com/stackql/stackql/actions/workflows/mcp-packaging.yml/badge.svg)](https://github.com/stackql/stackql/actions/workflows/mcp-packaging.yml)

The JVM member of the StackQL embedded-MCP family. StackQL exposes cloud providers (AWS, GitHub, Google, Azure, and more) as SQL tables; this library acquires the `stackql` binary, launches it as an MCP server over stdio, and hands you a connected client built on the official [MCP Kotlin SDK](https://github.com/modelcontextprotocol/kotlin-sdk).

Source of truth is [packaging/mcpb/kotlin](https://github.com/stackql/stackql/tree/main/packaging/mcpb/kotlin) in stackql/stackql. The library version equals the stackql release it embeds (pin the minor, e.g. `io.stackql:stackql-mcp:0.10.+`), and its per-platform sha256 pins come from `platforms.json`, rendered by `packaging/mcpb/scripts/render-platforms.sh` from the published `.mcpb.sha256` release assets - the same manifest the npm, PyPI and other SDK wrappers use.

`costgate`, the demo that makes cloud cost a build check (core, CLI, and the `io.stackql.costgate` Gradle plugin), lives in [stackql-labs](https://github.com/stackql-labs) and depends on the published library.

## Modules

| Module | Coordinate | What it is |
|---|---|---|
| `stackql-mcp` | `io.stackql:stackql-mcp` (Maven Central) | The library: acquire, launch, connect |
| `launcher` | - | Conformance launcher `smoke-test.py --cmd` drives (not published) |

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
6. Verified download - fetch the pinned `.mcpb` from `https://releases.stackql.io` (User-Agent `stackql-mcp-server-kotlin/<version>`), check its sha256 against the `platforms.json` pin on the classpath, extract, cache. Subsequent starts are offline.

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

## Build and test

JDK 17, Kotlin 2.x, Gradle (wrapper checked in). The integration/conformance tests are excluded by default to keep the build hermetic; opt in with `-PrunIntegration=true`.

```sh
make kotlin-manifest VERSION=X.Y.Z           # from packaging/mcpb: render platforms.json (classpath resource, gitignored)
./gradlew build                              # all modules, unit tests
./gradlew :stackql-mcp:test -PrunIntegration=true \
    --tests "io.stackql.mcp.ConformanceTest" # downloads + pin-verifies the server
./gradlew :launcher:installDist              # the conformance launcher
python scripts/smoke-test.py --cmd "kotlin/launcher/build/install/stackql-mcp-launch/bin/stackql-mcp-launch"  # from packaging/mcpb (unix)
python scripts/smoke-test.py --cmd "java -cp kotlin/launcher/build/install/stackql-mcp-launch/lib/* io.stackql.mcp.launcher.MainKt"  # any OS
```

The `mcp-packaging` workflow in stackql/stackql runs the unit tests, the conformance sequence, and the smoke test on Linux, macOS, and Windows.

## Publishing

Maven Central via the Central Portal, from CI (`mcp-packaging` dispatch) or a local clone: `./gradlew :stackql-mcp:publishToMavenCentral`, signed. Credentials come from the environment: `ORG_GRADLE_PROJECT_mavenCentralUsername`, `ORG_GRADLE_PROJECT_mavenCentralPassword`, `ORG_GRADLE_PROJECT_signingInMemoryKey`, `ORG_GRADLE_PROJECT_signingInMemoryKeyPassword`.

## License

MIT. mcp-name reference: `io.github.stackql/stackql-mcp`.
