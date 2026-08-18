package io.stackql.mcp

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

/**
 * The pinned stackql release, download base URL and per-platform sha256 pins.
 *
 * Nothing is hand-written here: the values come from the `platforms.json`
 * classpath resource, rendered by `packaging/mcpb/scripts/render-platforms.sh`
 * from the published `.mcpb.sha256` release assets - the same manifest every
 * other StackQL wrapper (npm, PyPI, the other SDKs) ships. Render it before
 * building: `make kotlin-manifest VERSION=X.Y.Z` (from packaging/mcpb).
 */
object Pins {
    private const val RESOURCE = "/platforms.json"

    @Serializable
    private data class Manifest(
        val version: String,
        val baseUrl: String,
        val platforms: Map<String, PlatformPin>,
    )

    @Serializable
    data class PlatformPin(val bundle: String, val sha256: String)

    private val manifest: Manifest by lazy {
        val stream = Pins::class.java.getResourceAsStream(RESOURCE)
            ?: throw StackqlMcpException(
                "classpath resource $RESOURCE not found - render it with " +
                    "'make kotlin-manifest VERSION=X.Y.Z' from packaging/mcpb",
            )
        val text = stream.use { it.readBytes().toString(Charsets.UTF_8) }
        Json { ignoreUnknownKeys = true }.decodeFromString<Manifest>(text).also {
            require(it.version.isNotBlank() && it.baseUrl.isNotBlank() && it.platforms.isNotEmpty()) {
                "platforms.json is missing version, baseUrl or platforms"
            }
        }
    }

    /** The stackql release this library is version-locked to (leading `v` stripped). */
    val STACKQL_VERSION: String get() = manifest.version

    /** Front door bundles are downloaded from (`https://releases.stackql.io/stackql/<version>`). */
    val BASE_URL: String get() = manifest.baseUrl

    /** User-Agent sent to the download proxy, per vector and version. */
    val USER_AGENT: String get() = "stackql-mcp-server-kotlin/${manifest.version}"

    /** Lowercase hex sha256 of each platform's `.mcpb` bundle for [STACKQL_VERSION]. */
    val pins: Map<Platform, String> by lazy {
        Platform.entries.associateWith { platform ->
            manifest.platforms[platform.key]?.sha256
                ?: throw StackqlMcpException("platforms.json has no pin for ${platform.key}")
        }
    }

    /** The published pin for [platform]; every platform has one. */
    fun pinFor(platform: Platform): String =
        pins[platform] ?: throw StackqlMcpException("no pin for platform ${platform.key}")

    /** Download URL for [platform]'s pinned `.mcpb` bundle: `<baseUrl>/<bundle>`. */
    fun bundleUrl(platform: Platform): String {
        val bundle = manifest.platforms[platform.key]?.bundle ?: platform.bundleName
        return "$BASE_URL/$bundle"
    }
}
