import XCTest
@testable import StackQLMCP

final class PlatformTests: XCTestCase {
    func testPlatformKeyMapping() {
        XCTAssertEqual(Platform.key(os: .linux, arch: .x86_64), .linuxX64)
        XCTAssertEqual(Platform.key(os: .linux, arch: .arm64), .linuxArm64)
        XCTAssertEqual(Platform.key(os: .windows, arch: .x86_64), .windowsX64)
        XCTAssertEqual(Platform.key(os: .macOS, arch: .x86_64), .darwinUniversal)
        XCTAssertEqual(Platform.key(os: .macOS, arch: .arm64), .darwinUniversal)
        // No published binary for these combinations.
        XCTAssertNil(Platform.key(os: .windows, arch: .arm64))
        XCTAssertNil(Platform.key(os: .other, arch: .x86_64))
    }

    func testExecutableName() {
        XCTAssertEqual(Platform.darwinUniversal.executableName, "stackql")
        XCTAssertEqual(Platform.linuxX64.executableName, "stackql")
        XCTAssertEqual(Platform.windowsX64.executableName, "stackql.exe")
    }

    func testCurrentPlatformOnMacIsDarwinUniversal() {
        // CI runs the unit suite on macOS runners.
        #if os(macOS)
        XCTAssertEqual(Platform.current, .darwinUniversal)
        #endif
    }

    func testEveryPlatformHasAPin() {
        for platform in Platform.allCases {
            XCTAssertNotNil(Pins.bundleSHA256[platform], "missing pin for \(platform)")
            XCTAssertEqual(Pins.bundleSHA256[platform]?.count, 64)
        }
    }
}
