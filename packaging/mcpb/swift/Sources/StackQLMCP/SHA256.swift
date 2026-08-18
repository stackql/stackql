import Foundation
import CryptoKit

/// sha256 helpers. CryptoKit ships with the OS on every supported platform
/// (macOS 13+), so this needs no extra dependency.
enum SHA256Hash {
    /// Lowercase hex sha256 of in-memory bytes.
    static func hex(of data: Data) -> String {
        SHA256.hash(data: data).hexString
    }

    /// Lowercase hex sha256 of a file, streamed so large binaries do not all
    /// sit in memory at once. Returns nil if the file cannot be read.
    static func hexOfFile(at url: URL) -> String? {
        guard let handle = try? FileHandle(forReadingFrom: url) else { return nil }
        defer { try? handle.close() }
        var hasher = SHA256()
        while true {
            let chunk: Data
            do {
                chunk = try handle.read(upToCount: 1 << 16) ?? Data()
            } catch {
                return nil
            }
            if chunk.isEmpty { break }
            hasher.update(data: chunk)
        }
        return hasher.finalize().hexString
    }

    /// Whether `url` exists and hashes to `expected` (lowercase hex). A stale
    /// or partial file (wrong hash) returns false, so callers re-extract.
    static func fileMatches(_ url: URL, expected: String) -> Bool {
        guard let got = hexOfFile(at: url) else { return false }
        return got == expected.lowercased()
    }
}

private extension Sequence where Element == UInt8 {
    var hexString: String {
        var out = ""
        out.reserveCapacity(64)
        for b in self { out += String(format: "%02x", b) }
        return out
    }
}
