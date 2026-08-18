package io.stackql.mcp

import java.net.URI
import java.net.http.HttpClient
import java.net.http.HttpRequest
import java.net.http.HttpResponse
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.StandardCopyOption
import java.security.DigestInputStream
import java.security.MessageDigest
import java.time.Duration

/** Thrown when a downloaded bundle's sha256 does not match the published pin. */
class ChecksumMismatchException(url: String, expected: String, actual: String) :
    StackqlMcpException("bundle sha256 mismatch for $url: got $actual, want $expected")

/**
 * Verified download of release bundles. The stream is hashed while it is
 * written to a temp file; the file is renamed into [dest] only after the hash
 * matches, so [dest] never holds an unverified or partial bundle.
 */
object Download {
    private val client: HttpClient = HttpClient.newBuilder()
        .connectTimeout(Duration.ofSeconds(30))
        .followRedirects(HttpClient.Redirect.NORMAL)
        .build()

    /**
     * Download [url] to [dest], verifying the stream against [expectedSha256]
     * (lowercase hex). Creates parent directories. Leaves no temp file behind
     * on failure.
     */
    fun verified(url: String, expectedSha256: String, dest: Path) {
        dest.parent?.let { Files.createDirectories(it) }
        val tmp = dest.resolveSibling("${dest.fileName}.part-${ProcessHandle.current().pid()}")
        try {
            val request = HttpRequest.newBuilder(URI.create(url))
                .timeout(Duration.ofMinutes(5))
                // Per-vector User-Agent so the download proxy can attribute traffic.
                .header("User-Agent", Pins.USER_AGENT)
                .GET()
                .build()
            val response = client.send(request, HttpResponse.BodyHandlers.ofInputStream())
            if (response.statusCode() != 200) {
                throw StackqlMcpException("GET $url: HTTP ${response.statusCode()}")
            }
            val digest = MessageDigest.getInstance("SHA-256")
            DigestInputStream(response.body(), digest).use { input ->
                Files.newOutputStream(tmp).use { output ->
                    input.copyTo(output, bufferSize = 64 * 1024)
                }
            }
            val actual = digest.digest().joinToString("") { "%02x".format(it) }
            if (actual != expectedSha256) {
                throw ChecksumMismatchException(url, expectedSha256, actual)
            }
            Files.move(tmp, dest, StandardCopyOption.REPLACE_EXISTING)
        } finally {
            Files.deleteIfExists(tmp)
        }
    }
}
