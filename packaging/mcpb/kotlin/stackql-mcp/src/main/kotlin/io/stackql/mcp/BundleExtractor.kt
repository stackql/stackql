package io.stackql.mcp

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import java.io.InputStream
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.StandardCopyOption
import java.util.zip.ZipInputStream

/**
 * `.mcpb` bundle extraction. A `.mcpb` is a zip containing `manifest.json`
 * and the server binary at the manifest's `server.entry_point`
 * (typically `server/stackql` or `server/stackql.exe`).
 *
 * Extraction goes to a sibling temp directory first and is moved into place
 * with a single rename, so a crash never leaves a half-populated cache entry
 * and concurrent extractors race benignly.
 */
object BundleExtractor {
    private val json = Json { ignoreUnknownKeys = true }

    /** Open a bundle from disk and extract into [destDir]; returns the binary path. */
    fun extract(bundle: Path, destDir: Path): Path =
        Files.newInputStream(bundle).use { extract(it, destDir) }

    /**
     * Extract a `.mcpb` from [bundle] into [destDir] and return the path to
     * the server binary. If [destDir] already holds a valid extraction, that
     * binary is returned without re-extracting.
     */
    fun extract(bundle: InputStream, destDir: Path): Path {
        cachedBinary(destDir)?.let { return it }

        val parent = destDir.parent
            ?: throw StackqlMcpException("cache dir $destDir has no parent")
        Files.createDirectories(parent)

        val tmp = parent.resolve(".extract-${ProcessHandle.current().pid()}-${destDir.fileName}")
        if (Files.exists(tmp)) tmp.toFile().deleteRecursively()

        try {
            unzipInto(bundle, tmp)
            val entry = readEntryPoint(tmp)
            val binary = tmp.resolve(entry)
            if (!Files.isRegularFile(binary)) {
                throw StackqlMcpException("entry_point $entry not found in bundle")
            }
            makeExecutable(binary)
            try {
                Files.move(tmp, destDir, StandardCopyOption.ATOMIC_MOVE)
            } catch (_: Exception) {
                // Another process won the race, or the FS rejects atomic move
                // across an existing dir. If the destination is now valid, use
                // it; otherwise rethrow via a plain move.
                cachedBinary(destDir)?.let { return it }
                Files.move(tmp, destDir)
            }
            return destDir.resolve(entry)
        } finally {
            if (Files.exists(tmp)) tmp.toFile().deleteRecursively()
        }
    }

    /** If [destDir] holds a valid extracted bundle, return its binary path. */
    fun cachedBinary(destDir: Path): Path? {
        val manifest = destDir.resolve("manifest.json")
        if (!Files.isRegularFile(manifest)) return null
        val entry = runCatching { readEntryPoint(destDir) }.getOrNull() ?: return null
        val binary = destDir.resolve(entry)
        return if (Files.isRegularFile(binary)) binary else null
    }

    private fun readEntryPoint(dir: Path): Path {
        val manifestText = Files.readString(dir.resolve("manifest.json"))
        val entry = json.parseToJsonElement(manifestText)
            .jsonObject["server"]?.jsonObject
            ?.get("entry_point")?.jsonPrimitive?.content
            ?: throw StackqlMcpException("manifest.json has no server.entry_point")
        return sanitizeEntryPoint(entry)
    }

    /** Reject absolute or parent-traversing entry_point values. */
    private fun sanitizeEntryPoint(entry: String): Path {
        if (entry.isEmpty()) throw StackqlMcpException("empty entry_point")
        val path = Path.of(entry)
        val safe = !path.isAbsolute && path.none { it.toString() == ".." }
        if (!safe) throw StackqlMcpException("unsafe entry_point: $entry")
        return path
    }

    private fun unzipInto(bundle: InputStream, dir: Path) {
        Files.createDirectories(dir)
        ZipInputStream(bundle).use { zip ->
            var entry = zip.nextEntry
            while (entry != null) {
                val target = dir.resolve(entry.name).normalize()
                if (!target.startsWith(dir)) {
                    throw StackqlMcpException("zip entry escapes target dir: ${entry.name}")
                }
                if (entry.isDirectory) {
                    Files.createDirectories(target)
                } else {
                    target.parent?.let { Files.createDirectories(it) }
                    Files.newOutputStream(target).use { zip.copyTo(it, 64 * 1024) }
                }
                zip.closeEntry()
                entry = zip.nextEntry
            }
        }
    }

    private fun makeExecutable(binary: Path) {
        runCatching {
            val perms = Files.getPosixFilePermissions(binary).toMutableSet()
            perms.addAll(
                setOf(
                    java.nio.file.attribute.PosixFilePermission.OWNER_EXECUTE,
                    java.nio.file.attribute.PosixFilePermission.GROUP_EXECUTE,
                    java.nio.file.attribute.PosixFilePermission.OTHERS_EXECUTE,
                ),
            )
            Files.setPosixFilePermissions(binary, perms)
        } // No-op on Windows (no POSIX view); the .exe is runnable as-is.
    }
}
