# CLAUDE.md - packaging/mcpb/kotlin (Maven Central `io.stackql:stackql-mcp`)

The Kotlin/JVM member of the StackQL embedded-MCP family: `stackql-mcp` acquires the `stackql` binary (`Acquire`: overrides, local bundle, shared cache, pin-verified download via the JDK `HttpClient`), spawns it over stdio (Windows quoting fix: `jdk.lang.Process.allowAmbiguousCommands=false`) and returns a connected MCP Kotlin SDK client. Gradle wrapper 8.14, Kotlin 2.3, JDK 17 toolchain. `gradle.properties` `version=` is the stamp (`allprojects.version` reads it). Publishing uses `com.vanniktech.maven.publish` (`publishToMavenCentral`, signed when `signingInMemoryKey` is present). The `launcher` module (application plugin, not published) is the conformance launcher.

## Tier: preview

This vector is in tree, on the shared contract and renderer, and CI-validated by the `sdk` matrix on every packaging PR and dispatch - but it is NOT published and is not part of the release train (the published tier is cargo, go, dotnet). `make kotlin-publish` exists for a deliberate out-of-band publish; promoting this vector means adding a publish job to `mcp-packaging.yml`, its secrets, and its row in `docs/stackql release process.md`.

## The contract (do not deviate)

Owned by [packaging/mcpb](../CLAUDE.md) in stackql/stackql - read "The nine wrapper vectors and the ordering rules" there first. In short:

- Package version == stackql release version; stamped by `scripts/render-platforms.sh` (`make kotlin-manifest VERSION=X.Y.Z` from `packaging/mcpb`), never hand-edited.
- Pins are data: `platforms.json` `{version, baseUrl, platforms{<key>:{bundle, sha256}}}` rendered from the published `.mcpb.sha256` release assets. `stackql-mcp/src/main/resources/platforms.json` on the classpath, parsed lazily by `Pins`. No hand-written pin table anywhere. The rendered files are gitignored - render before building.
- Runtime download from `baseUrl` (`https://releases.stackql.io/stackql/<version>`) with `User-Agent: stackql-mcp-server-kotlin/<version>`. No GitHub API calls, no `latest`, no version override.
- Shared cache `~/.stackql/mcp-server-bin/<version>/<platform-key>/`; keys `linux-x64 | linux-arm64 | windows-x64 | darwin-universal`.
- Overrides `STACKQL_MCP_BIN` and `STACKQL_MCP_BUNDLE` (local `.mcpb`, no pin check); nothing else.
- Canonical argv `mcp --mcp.server.type=stdio --approot <home>/.stackql --mcp.config {"server":{"mode":"<mode>","audit":{"disabled":true}}} [--auth=<json>]`; default mode `read_only`.
- Launcher for `scripts/smoke-test.py --cmd`: `./gradlew -q :launcher:installDist` then `launcher/build/install/stackql-mcp-launch/bin/stackql-mcp-launch` (or `java -cp .../lib/* io.stackql.mcp.launcher.MainKt`); `ConformanceTest` is the in-repo port (`-PrunIntegration=true`).

## Build, test, publish

```
cd packaging/mcpb
make kotlin-manifest VERSION=X.Y.Z   # render pins + version stamp (after the .mcpb assets are on the release)
make kotlin-build    VERSION=X.Y.Z   # lint + unit tests + package
make kotlin-smoke    VERSION=X.Y.Z   # smoke-test.py --cmd against the launcher
make kotlin-publish  VERSION=X.Y.Z   # `./gradlew :stackql-mcp:publishToMavenCentral` (ORG_GRADLE_PROJECT_mavenCentral*/signingInMemory* env); skips if the version is on Maven Central
```

CI: the `sdk` matrix and `kotlin-publish` job in `.github/workflows/mcp-packaging.yml`.

## Out of scope here

Demo apps (costgate, costgate-cli, costgate-gradle, examples/) live in stackql-labs and depend on the published package. No new features beyond the contract; parity items go on the backlog as separate PRs.

## Writing style

Plain hyphens, ASCII arrows (`->`), QWERTY characters only, matter-of-fact tone, no stacked headings.
