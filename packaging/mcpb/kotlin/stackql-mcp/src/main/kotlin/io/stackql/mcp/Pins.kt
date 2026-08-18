package io.stackql.mcp

/**
 * Per-platform sha256 pins for the packaged stackql release.
 *
 * Rendered from the `.sha256` assets on the stackql/stackql release that the
 * packaging repo (stackql/stackql-mcpb-packaging) targets. Update [pins] and
 * [STACKQL_VERSION] together when bumping the server version. Once the
 * packaging repo publishes a consolidated platforms.json release asset,
 * prefer rendering this table from that.
 */
object Pins {
    /** The stackql release this library version pins (leading `v` stripped). */
    const val STACKQL_VERSION: String = "0.10.500"

    /** Asset download root: bundles are attached to the stackql/stackql release. */
    const val RELEASE_URL_BASE: String =
        "https://github.com/stackql/stackql/releases/download"

    /** Lowercase hex sha256 of each platform's `.mcpb` bundle for [STACKQL_VERSION]. */
    val pins: Map<Platform, String> = mapOf(
        Platform.LinuxX64 to "6615737747156b1a8413a976afb23af2e7eec29ebc98a6f0a0f65d1b153c44be",
        Platform.LinuxArm64 to "594bedbabc3096dc3563c907724e845ce0b61a67de4b3fed4158b40c0363786c",
        Platform.WindowsX64 to "d2ce895e88f9c6b557df07073158629808f56d75598f3a701164d65506b791b0",
        Platform.DarwinUniversal to "4eed70af5cfa67295ae0b42fa3a6dca71ac9acabd0d67914fd96ad1247a9b4cc",
    )

    /** The published pin for [platform]; every platform has one. */
    fun pinFor(platform: Platform): String =
        pins[platform] ?: throw StackqlMcpException("no pin for platform ${platform.key}")

    /** Download URL for [platform]'s pinned `.mcpb` bundle. */
    fun bundleUrl(platform: Platform, version: String = STACKQL_VERSION): String =
        "$RELEASE_URL_BASE/v$version/${platform.bundleName}"
}
