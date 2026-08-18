package io.stackql.mcp

import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.Paths
import java.security.MessageDigest

/**
 * The shared on-disk binary cache, `~/.stackql/mcp-server-bin`. Same location
 * the npm and pypi wrappers use, so multiple embedders on one machine share a
 * single extraction, and so do polyglot toolchains.
 *
 * Layout: `<root>/<version>/<platform-key>/stackql[.exe]`.
 */
object Cache {
    /** Env override: path to a stackql binary to run directly, skipping acquisition. */
    const val ENV_BIN: String = "STACKQL_MCP_BIN"

    /** Env override: path to a local `.mcpb` to extract instead of downloading. */
    const val ENV_BUNDLE: String = "STACKQL_MCP_BUNDLE"

    /** `~/.stackql/mcp-server-bin`. */
    fun defaultRoot(): Path =
        Paths.get(System.getProperty("user.home"), ".stackql", "mcp-server-bin")

    /** The extraction directory for a version/platform: `<root>/<version>/<key>`. */
    fun binaryDir(root: Path, version: String, platform: Platform): Path =
        root.resolve(version).resolve(platform.key)

    /** The canonical binary path inside [binaryDir]. */
    fun binaryPath(root: Path, version: String, platform: Platform): Path =
        binaryDir(root, version, platform).resolve(platform.exeName)

    /** Lowercase hex sha256 of a byte array. */
    fun sha256(data: ByteArray): String =
        MessageDigest.getInstance("SHA-256").digest(data).toHex()

    /** Lowercase hex sha256 of a file, streamed so large binaries do not load whole. */
    fun sha256(path: Path): String {
        val digest = MessageDigest.getInstance("SHA-256")
        Files.newInputStream(path).use { input ->
            val buf = ByteArray(64 * 1024)
            while (true) {
                val n = input.read(buf)
                if (n < 0) break
                digest.update(buf, 0, n)
            }
        }
        return digest.digest().toHex()
    }

    private fun ByteArray.toHex(): String =
        joinToString("") { "%02x".format(it) }
}
