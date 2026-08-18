import XCTest
@testable import StackQLMCP

final class LaunchArgumentsTests: XCTestCase {
    func testCanonicalArgs() throws {
        let approot = NSTemporaryDirectory() + "stackql-test/.stackql"
        let args = try LaunchArguments.build(mode: .readOnly, appRoot: approot, auth: nil)
        let want = [
            "mcp",
            "--mcp.server.type=stdio",
            "--approot", approot,
            "--mcp.config", #"{"server":{"mode":"read_only","audit":{"disabled":true}}}"#,
        ]
        XCTAssertEqual(args, want)
    }

    func testModeAndAuth() throws {
        let approot = NSTemporaryDirectory() + "stackql-test/.stackql"
        let args = try LaunchArguments.build(
            mode: .safe,
            appRoot: approot,
            auth: ["github": ["type": "null_auth"]]
        )
        let joined = args.joined(separator: " ")
        XCTAssertTrue(joined.contains(#""mode":"safe""#), joined)

        let last = try XCTUnwrap(args.last)
        XCTAssertTrue(last.hasPrefix("--auth="), last)
        let json = String(last.dropFirst("--auth=".count))
        let obj = try JSONSerialization.jsonObject(with: Data(json.utf8)) as? [String: Any]
        let github = obj?["github"] as? [String: Any]
        XCTAssertEqual(github?["type"] as? String, "null_auth")
    }

    func testRejectsRelativeAppRoot() {
        XCTAssertThrowsError(try LaunchArguments.build(appRoot: "relative/path"))
    }

    func testDefaultModeIsReadOnly() throws {
        let approot = NSTemporaryDirectory() + "stackql-test/.stackql"
        let args = try LaunchArguments.build(appRoot: approot)
        XCTAssertTrue(args.joined(separator: " ").contains(#""mode":"read_only""#))
    }

    func testDefaultAppRootIsAbsoluteUnderDotStackql() throws {
        let root = try LaunchArguments.defaultAppRoot()
        XCTAssertTrue((root as NSString).isAbsolutePath)
        XCTAssertTrue(root.hasSuffix(".stackql"), root)
    }
}
