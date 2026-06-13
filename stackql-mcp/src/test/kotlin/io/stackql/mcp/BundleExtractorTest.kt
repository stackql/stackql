package io.stackql.mcp

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertFalse
import org.junit.jupiter.api.Assertions.assertThrows
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import java.io.ByteArrayOutputStream
import java.nio.file.Files
import java.nio.file.Path
import java.util.zip.ZipEntry
import java.util.zip.ZipOutputStream

class BundleExtractorTest {

    /** A minimal .mcpb: manifest.json pointing at the given entry_point plus a stub binary. */
    private fun fakeBundle(entryPoint: String, binaryRelPath: String = "server/stackql"): ByteArray {
        val out = ByteArrayOutputStream()
        ZipOutputStream(out).use { zip ->
            zip.putNextEntry(ZipEntry("manifest.json"))
            zip.write("""{"server": {"entry_point": "$entryPoint"}}""".toByteArray())
            zip.closeEntry()
            zip.putNextEntry(ZipEntry(binaryRelPath))
            zip.write("#!/bin/sh\necho fake stackql\n".toByteArray())
            zip.closeEntry()
        }
        return out.toByteArray()
    }

    @Test
    fun extractsAndReturnsTheEntryPoint(@TempDir tmp: Path) {
        val dest = tmp.resolve("bundle")
        val binary = BundleExtractor.extract(fakeBundle("server/stackql").inputStream(), dest)
        assertEquals(dest.resolve("server").resolve("stackql"), binary)
        assertTrue(Files.isRegularFile(binary))
    }

    @Test
    fun secondCallIsACacheHit(@TempDir tmp: Path) {
        val dest = tmp.resolve("bundle")
        val first = BundleExtractor.extract(fakeBundle("server/stackql").inputStream(), dest)
        // A second extract finds the manifest already present and short-circuits.
        val again = BundleExtractor.extract(fakeBundle("server/stackql").inputStream(), dest)
        assertEquals(first, again)
        assertTrue(Files.isRegularFile(again))
    }

    @Test
    fun rejectsTraversalInEntryPoint(@TempDir tmp: Path) {
        val dest = tmp.resolve("bundle")
        assertThrows(StackqlMcpException::class.java) {
            BundleExtractor.extract(fakeBundle("../../evil", "evil").inputStream(), dest)
        }
        assertFalse(Files.exists(dest), "failed extraction must not populate the cache")
    }

    @Test
    fun missingEntryPointIsAnError(@TempDir tmp: Path) {
        val dest = tmp.resolve("bundle")
        // Manifest names server/nope, but the only binary is server/stackql.
        assertThrows(StackqlMcpException::class.java) {
            BundleExtractor.extract(fakeBundle("server/nope").inputStream(), dest)
        }
    }

    @Test
    fun extractedBinaryIsExecutableOnPosix(@TempDir tmp: Path) {
        val isWindows = System.getProperty("os.name").lowercase().contains("windows")
        org.junit.jupiter.api.Assumptions.assumeFalse(isWindows, "POSIX permission check")
        val dest = tmp.resolve("bundle")
        val binary = BundleExtractor.extract(fakeBundle("server/stackql").inputStream(), dest)
        val perms = Files.getPosixFilePermissions(binary)
        assertTrue(perms.contains(java.nio.file.attribute.PosixFilePermission.OWNER_EXECUTE))
    }

    @Test
    fun sha256MatchesKnownVector(@TempDir tmp: Path) {
        // sha256("abc")
        val f = tmp.resolve("abc.txt")
        Files.writeString(f, "abc")
        assertEquals(
            "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
            Cache.sha256(f),
        )
        assertEquals(
            "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
            Cache.sha256("abc".toByteArray()),
        )
    }
}
