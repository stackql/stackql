import XCTest
@testable import StackQLMCP

final class BundleFetcherTests: XCTestCase {
    /// Build a fake .mcpb (zip) with the given manifest entry_point and a
    /// server binary at server/stackql, using the system `zip`.
    private func makeBundle(entryPoint: String) throws -> Data {
        let work = FileManager.default.temporaryDirectory
            .appendingPathComponent("mcpb-build-\(UUID().uuidString)")
        let staging = work.appendingPathComponent("staging")
        try FileManager.default.createDirectory(
            at: staging.appendingPathComponent("server"), withIntermediateDirectories: true)
        addTeardownBlock { try? FileManager.default.removeItem(at: work) }

        let manifest = #"{"server": {"entry_point": "\#(entryPoint)"}}"#
        try manifest.data(using: .utf8)!.write(
            to: staging.appendingPathComponent("manifest.json"))
        try Data("#!/bin/sh\necho fake stackql\n".utf8).write(
            to: staging.appendingPathComponent("server/stackql"))

        let archive = work.appendingPathComponent("bundle.mcpb")
        let zip = Process()
        zip.executableURL = URL(fileURLWithPath: "/usr/bin/zip")
        zip.currentDirectoryURL = staging
        zip.arguments = ["-q", "-r", archive.path, "."]
        zip.standardOutput = FileHandle.nullDevice
        zip.standardError = FileHandle.nullDevice
        try zip.run()
        zip.waitUntilExit()
        XCTAssertEqual(zip.terminationStatus, 0)
        return try Data(contentsOf: archive)
    }

    func testExtractsEntryPoint() throws {
        let bundle = try makeBundle(entryPoint: "server/stackql")
        let binary = try BundleFetcher.extractEntryPoint(fromBundle: bundle)
        XCTAssertEqual(String(decoding: binary, as: UTF8.self), "#!/bin/sh\necho fake stackql\n")
    }

    func testRejectsTraversalEntryPoint() throws {
        let bundle = try makeBundle(entryPoint: "../../evil")
        XCTAssertThrowsError(try BundleFetcher.extractEntryPoint(fromBundle: bundle)) { error in
            guard case FetchError.unsafeEntryPoint = error else {
                return XCTFail("expected unsafeEntryPoint, got \(error)")
            }
        }
    }

    func testMissingEntryPointIsAnError() throws {
        let bundle = try makeBundle(entryPoint: "server/nope")
        XCTAssertThrowsError(try BundleFetcher.extractEntryPoint(fromBundle: bundle))
    }

    func testAssetURLConstruction() {
        let url = BundleFetcher.assetURL(version: "0.10.500", name: "stackql-mcp-darwin-universal.mcpb")
        XCTAssertEqual(
            url.absoluteString,
            "https://github.com/stackql/stackql/releases/download/v0.10.500/stackql-mcp-darwin-universal.mcpb"
        )
    }

    func testResolvePinUsesTableForDefaultVersion() async throws {
        let pin = try await BundleFetcher().resolvePin(
            version: Pins.defaultVersion, platform: .darwinUniversal)
        XCTAssertEqual(pin, Pins.bundleSHA256[.darwinUniversal])
    }
}
