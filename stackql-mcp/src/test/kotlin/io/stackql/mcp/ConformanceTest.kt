package io.stackql.mcp

import io.modelcontextprotocol.kotlin.sdk.types.CallToolResult
import io.modelcontextprotocol.kotlin.sdk.types.TextContent
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.withTimeout
import org.junit.jupiter.api.Assertions.assertNotNull
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Assertions.fail
import org.junit.jupiter.api.Test
import java.nio.file.Path
import java.nio.file.Paths

/**
 * Conformance test mirroring stackql-mcpb-packaging scripts/smoke-test.py:
 * acquire the server (sidecar download, pin-verified), spawn it over stdio,
 * complete the MCP handshake, then exercise the github provider with
 * null_auth (no credentials):
 * initialize -> tools/list -> pull_provider -> list_services.
 *
 * Excluded from the default build (see root build.gradle.kts); run with
 * `./gradlew :stackql-mcp:test -PrunIntegration=true`. It reaches the network
 * (registry.stackql.app and the release download) and spawns the server.
 *
 * The binary cache is a persistent dir under the build folder rather than a
 * per-test @TempDir: on Windows the extracted server executable stays locked
 * for a moment after the process exits, and a @TempDir teardown that races
 * that lock fails. A shared, un-deleted cache also means the two tests share
 * one download. The approot stays per-test hermetic (data files, no locks).
 */
class ConformanceTest {

    private fun buildDir(): Path = Paths.get(
        System.getProperty("io.stackql.test.buildDir")
            ?: Paths.get("build").toAbsolutePath().toString(),
    )

    private fun sharedCacheRoot(): Path = buildDir().resolve("conformance-bin-cache")

    // The approot also lives under the build dir, not a @TempDir: the server
    // writes a log file there and keeps the handle a moment after exit, which
    // races @TempDir teardown on Windows. Each test gets its own subdir so they
    // stay isolated; the build dir is cleaned by `gradle clean`.
    private fun approot(name: String): Path = buildDir().resolve("conformance-approot").resolve(name)

    private fun textOf(result: CallToolResult): String =
        result.content.filterIsInstance<TextContent>().joinToString("\n") { it.text }

    @Test
    fun githubNullAuth() = runBlocking {
        // Hermetic approot and cache, as the packaging smoke test does with
        // its substituted ${HOME}: proves cwd-independence and no writes
        // outside the configured roots.
        val server = withTimeout(5 * 60_000) {
            StackqlMcp.builder()
                .mode(Mode.ReadOnly)
                .appRoot(approot("github-null-auth"))
                .cacheRoot(sharedCacheRoot())
                .auth(LaunchArgs.authFor("github", "null_auth"))
                .start()
        }

        server.use {
            assertNotNull(server.client.serverVersion, "initialize returned no server info")

            val tools = withTimeout(60_000) { server.client.listTools() }.tools
            val names = tools.map { it.name }.toSet()
            for (required in listOf("pull_provider", "list_services", "list_providers")) {
                assertTrue(required in names, "missing required tool $required (got $names)")
            }

            val pull = withTimeout(2 * 60_000) {
                server.client.callTool("pull_provider", mapOf("provider" to "github"))
            }
            if (pull.isError == true) fail<Unit>("pull_provider failed: ${textOf(pull)}")

            val services = withTimeout(60_000) {
                server.client.callTool(
                    "list_services",
                    mapOf("provider" to "github", "row_limit" to 5),
                )
            }
            if (services.isError == true) fail<Unit>("list_services failed: ${textOf(services)}")
            val text = textOf(services)
            assertTrue(
                text.contains("actions") || text.contains("apps"),
                "list_services did not include expected github services: $text",
            )
        }
    }

    @Test
    fun readOnlyModeRefusesMutations() = runBlocking {
        val server = withTimeout(5 * 60_000) {
            StackqlMcp.builder()
                .appRoot(approot("read-only-refusal"))
                .cacheRoot(sharedCacheRoot())
                .start()
        }
        server.use {
            val res = withTimeout(2 * 60_000) {
                server.client.callTool(
                    "run_mutation_query",
                    mapOf(
                        "sql" to "INSERT INTO github.repos.repos(data__name) SELECT 'should-never-run'",
                    ),
                )
            }
            val text = textOf(res)
            assertTrue(
                res.isError == true && text.contains("read_only"),
                "read_only server did not refuse mutation: isError=${res.isError} text=$text",
            )
        }
    }
}
