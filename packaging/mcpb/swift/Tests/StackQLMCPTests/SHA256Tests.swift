import XCTest
@testable import StackQLMCP

final class SHA256Tests: XCTestCase {
    func testKnownVectorAbc() {
        // sha256("abc")
        XCTAssertEqual(
            SHA256Hash.hex(of: Data("abc".utf8)),
            "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
        )
    }

    func testFileHashMatchesInMemory() throws {
        let url = FileManager.default.temporaryDirectory
            .appendingPathComponent("sha-\(UUID().uuidString).bin")
        let data = Data((0..<200_000).map { UInt8($0 & 0xff) })
        try data.write(to: url)
        defer { try? FileManager.default.removeItem(at: url) }

        XCTAssertEqual(SHA256Hash.hexOfFile(at: url), SHA256Hash.hex(of: data))
        XCTAssertTrue(SHA256Hash.fileMatches(url, expected: SHA256Hash.hex(of: data)))
        XCTAssertFalse(SHA256Hash.fileMatches(url, expected: String(repeating: "0", count: 64)))
    }

    func testMissingFileDoesNotMatch() {
        let missing = URL(fileURLWithPath: "/no/such/file/\(UUID().uuidString)")
        XCTAssertNil(SHA256Hash.hexOfFile(at: missing))
        XCTAssertFalse(SHA256Hash.fileMatches(missing, expected: String(repeating: "0", count: 64)))
    }
}
