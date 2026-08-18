package io.stackql.mcp

import java.nio.file.Files
import java.nio.file.Path

/**
 * Resolve a runnable stackql binary. Resolution order (first match wins):
 *
 *  1. `STACKQL_MCP_BIN` env: run that binary directly, no acquisition.
 *  2. Builder `binary` override: same.
 *  3. `STACKQL_MCP_BUNDLE` env: extract that local `.mcpb`. No pin check - an
 *     explicit override is operator intent and may be a custom build.
 *  4. Builder `bundlePath` override: same.
 *  5. Shared cache: an already-extracted binary for the pinned version.
 *  6. Verified download: fetch the pinned `.mcpb`, check its sha256 against the
 *     baked-in pin, extract, cache. Subsequent starts are offline.
 */
internal object Acquire {

    /** Inputs collected from the builder; env overrides are read here and win. */
    class Inputs(
        val binary: Path? = null,
        val bundlePath: Path? = null,
        val cacheRoot: Path = Cache.defaultRoot(),
        val version: String = Pins.STACKQL_VERSION,
    )

    fun resolveBinary(inputs: Inputs): Path {
        envPath(Cache.ENV_BIN)?.let { return existing(it, Cache.ENV_BIN) }
        inputs.binary?.let { return existing(it, "Builder.binary") }
        envPath(Cache.ENV_BUNDLE)?.let { return extractLocalBundle(it, Cache.ENV_BUNDLE, inputs) }
        inputs.bundlePath?.let { return extractLocalBundle(it, "Builder.bundlePath", inputs) }
        return sidecar(inputs)
    }

    private fun envPath(name: String): Path? =
        System.getenv(name)?.takeIf { it.isNotEmpty() }?.let { Path.of(it) }

    private fun existing(path: Path, what: String): Path {
        if (!Files.isRegularFile(path)) {
            throw StackqlMcpException("$what points at $path, which is not a file")
        }
        return path
    }

    /** Extract a caller-supplied `.mcpb` into a slot keyed by its content hash. */
    private fun extractLocalBundle(bundlePath: Path, what: String, inputs: Inputs): Path {
        if (!Files.isRegularFile(bundlePath)) {
            throw StackqlMcpException("$what points at $bundlePath, which is not a file")
        }
        val digest = Cache.sha256(bundlePath)
        val dest = inputs.cacheRoot.resolve("custom").resolve(digest.substring(0, 16))
        return BundleExtractor.extract(bundlePath, dest)
    }

    private fun sidecar(inputs: Inputs): Path {
        val platform = Platform.current()
        val dest = Cache.binaryDir(inputs.cacheRoot, inputs.version, platform)
        BundleExtractor.cachedBinary(dest)?.let { return it }

        val pin = Pins.pinFor(platform)
        val url = Pins.bundleUrl(platform, inputs.version)
        val mcpb = inputs.cacheRoot.resolve(inputs.version).resolve(platform.bundleName)
        System.err.println("stackql-mcp: downloading $url (first run, cached at $dest)")
        Download.verified(url, pin, mcpb)
        val binary = BundleExtractor.extract(mcpb, dest)
        // The extracted dir is the cache; drop the archive to halve disk use.
        Files.deleteIfExists(mcpb)
        return binary
    }
}
