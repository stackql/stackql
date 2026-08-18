import XCTest
@testable import StackQLMCP

final class BinaryResolverTests: XCTestCase {
    private func tempDir() throws -> URL {
        let url = FileManager.default.temporaryDirectory
            .appendingPathComponent("stackql-resolver-\(UUID().uuidString)")
        try FileManager.default.createDirectory(at: url, withIntermediateDirectories: true)
        addTeardownBlock { try? FileManager.default.removeItem(at: url) }
        return url
    }

    func testCachedBinaryPathLayout() throws {
        let cache = try tempDir()
        let resolver = try BinaryResolver(
            platform: .darwinUniversal, version: "0.10.500", cacheDir: cache)
        let expected = cache
            .appendingPathComponent("0.10.500")
            .appendingPathComponent("darwin-universal")
            .appendingPathComponent("stackql")
        XCTAssertEqual(resolver.cachedBinaryPath.path, expected.path)
    }

    func testInstallToCacheWritesExecutableAndIsReused() throws {
        let cache = try tempDir()
        let resolver = try BinaryResolver(
            platform: .darwinUniversal, version: "0.0.1", cacheDir: cache)
        let data = Data("#!/bin/sh\necho fake-stackql\n".utf8)
        let sha = SHA256Hash.hex(of: data)

        let path = try resolver.installToCache(data: data, expectedSHA256: sha)
        XCTAssertTrue(FileManager.default.isExecutableFile(atPath: path.path))
        XCTAssertEqual(try Data(contentsOf: path), data)

        // Second install with the same sha is a no-op reuse: mtime unchanged.
        let attrs1 = try FileManager.default.attributesOfItem(atPath: path.path)
        let mtime1 = attrs1[.modificationDate] as? Date
        let path2 = try resolver.installToCache(data: data, expectedSHA256: sha)
        XCTAssertEqual(path.path, path2.path)
        let attrs2 = try FileManager.default.attributesOfItem(atPath: path.path)
        let mtime2 = attrs2[.modificationDate] as? Date
        XCTAssertEqual(mtime1, mtime2, "cached binary was rewritten on reuse")
    }

    func testInstallRejectsTamperedData() throws {
        let cache = try tempDir()
        let resolver = try BinaryResolver(
            platform: .darwinUniversal, version: "0.0.1", cacheDir: cache)
        let data = Data("good".utf8)
        let wrongSHA = SHA256Hash.hex(of: Data("different".utf8))
        XCTAssertThrowsError(try resolver.installToCache(data: data, expectedSHA256: wrongSHA))
    }

    func testInstallReplacesCorruptCache() throws {
        let cache = try tempDir()
        let resolver = try BinaryResolver(
            platform: .darwinUniversal, version: "0.0.1", cacheDir: cache)
        let target = resolver.cachedBinaryPath
        try FileManager.default.createDirectory(
            at: target.deletingLastPathComponent(), withIntermediateDirectories: true)
        try Data("corrupt".utf8).write(to: target)

        let good = Data("good".utf8)
        let path = try resolver.installToCache(data: good, expectedSHA256: SHA256Hash.hex(of: good))
        XCTAssertEqual(try Data(contentsOf: path), good)
    }

    func testBinaryOverrideTakesPriority() throws {
        let dir = try tempDir()
        let exe = dir.appendingPathComponent("my-stackql")
        try Data("#!/bin/sh\n".utf8).write(to: exe)
        try FileManager.default.setAttributes([.posixPermissions: 0o755], ofItemAtPath: exe.path)

        let resolver = try BinaryResolver(
            platform: .darwinUniversal,
            version: "0.10.500",
            cacheDir: try tempDir(),
            binaryOverride: exe.path
        )
        XCTAssertEqual(resolver.locateOffline()?.path, exe.path)
    }

    func testLocateOfflineReturnsNilWhenNothingPresent() throws {
        let resolver = try BinaryResolver(
            platform: .darwinUniversal,
            version: "0.10.500",
            cacheDir: try tempDir(),
            searchBundles: [],
            binaryOverride: ""
        )
        XCTAssertNil(resolver.locateOffline())
    }
}
