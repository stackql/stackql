package io.stackql.mcp

import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import java.nio.file.Path
import java.nio.file.Paths

/**
 * The canonical stackql MCP launch arguments. This shape is the embedding
 * contract from stackql/stackql-mcpb-packaging and must stay cwd-independent:
 * MCP hosts may launch with cwd "/" (read-only on macOS), so every path is
 * absolute.
 *
 * ```text
 * mcp --mcp.server.type=stdio --approot <approot>
 *     --mcp.config {"server":{"mode":"<mode>","audit":{"disabled":true}}}
 *     [--auth=<json>]
 * ```
 */
object LaunchArgs {

    /** `~/.stackql`, the application root shared with every stackql distribution. */
    fun defaultAppRoot(): Path =
        Paths.get(System.getProperty("user.home"), ".stackql")

    /**
     * Build the argument vector. [mode] defaults to [Mode.ReadOnly]; [appRoot]
     * defaults to [defaultAppRoot] and must be absolute. [auth], if present, is
     * serialized into a single `--auth=<json>` flag (for example the github
     * `null_auth` fixture used by the conformance tests).
     */
    fun build(
        mode: Mode = Mode.ReadOnly,
        appRoot: Path = defaultAppRoot(),
        auth: JsonElement? = null,
    ): List<String> {
        require(appRoot.isAbsolute) { "approot must be absolute, got $appRoot" }
        val config = buildJsonObject {
            put("server", buildJsonObject {
                put("mode", mode.value)
                put("audit", buildJsonObject { put("disabled", true) })
            })
        }
        val args = mutableListOf(
            "mcp",
            "--mcp.server.type=stdio",
            "--approot", appRoot.toAbsolutePath().toString(),
            "--mcp.config", config.toString(),
        )
        if (auth != null) {
            args += "--auth=$auth"
        }
        return args
    }

    /**
     * Convenience for the common single-provider auth override, e.g.
     * `authFor("github", "null_auth")` produces `{"github":{"type":"null_auth"}}`.
     */
    fun authFor(provider: String, type: String): JsonObject = buildJsonObject {
        put(provider, buildJsonObject { put("type", type) })
    }
}
