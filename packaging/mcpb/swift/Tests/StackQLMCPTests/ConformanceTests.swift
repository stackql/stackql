import XCTest
@testable import StackQLMCP

/// Conformance test mirroring stackql-mcpb-packaging scripts/smoke-test.py
/// and the Go/Rust siblings: spawn the server over stdio, complete the MCP
/// handshake, then exercise the github provider with null_auth (no creds):
/// initialize -> tools/list -> pull_provider -> list_services. Also asserts
/// the default read_only mode refuses mutation execution.
///
/// Skipped unless STACKQL_MCP_INTEGRATION=1, because it spawns the real
/// server and reaches registry.stackql.app. CI sets that variable on the
/// macOS runners. A prebuilt binary may be supplied via
/// STACKQL_MCP_TEST_BINARY; otherwise the pin-verified release bundle is
/// downloaded into a hermetic temp cache.
final class ConformanceTests: XCTestCase {
    private var integrationEnabled: Bool {
        ProcessInfo.processInfo.environment["STACKQL_MCP_INTEGRATION"] == "1"
    }

    /// A hermetic options set: temp approot and cache, as the packaging smoke
    /// test does with its substituted ${HOME}. Proves cwd-independence and
    /// no writes outside the configured roots.
    private func hermeticOptions(mode: Mode, auth: AuthDocument?) throws -> (Options, URL) {
        let tmp = FileManager.default.temporaryDirectory
            .appendingPathComponent("stackql-conformance-\(UUID().uuidString)")
        try FileManager.default.createDirectory(at: tmp, withIntermediateDirectories: true)

        var opts = Options()
        opts.mode = mode
        opts.appRoot = tmp.appendingPathComponent(".stackql").path
        opts.cacheDir = tmp.appendingPathComponent("bin-cache")
        opts.auth = auth
        if let override = ProcessInfo.processInfo.environment["STACKQL_MCP_TEST_BINARY"],
           !override.isEmpty {
            opts.binaryOverride = override
        }
        return (opts, tmp)
    }

    func testGithubNullAuthConformance() async throws {
        try XCTSkipUnless(integrationEnabled, "set STACKQL_MCP_INTEGRATION=1 to run")

        let (opts, tmp) = try hermeticOptions(
            mode: .readOnly, auth: ["github": ["type": "null_auth"]])
        let server = try await StackQLServer.start(opts)
        defer {
            Task { await server.stop() }
            try? FileManager.default.removeItem(at: tmp)
        }

        let tools = try await server.listToolNames()
        XCTAssertGreaterThan(tools.count, 0)
        for required in ["pull_provider", "list_services", "list_providers"] {
            XCTAssertTrue(tools.contains(required), "missing required tool \(required)")
        }

        let pull = try await server.call("pull_provider", ["provider": "github"])
        XCTAssertFalse(pull.isError, "pull_provider failed: \(pull.text)")

        let services = try await server.call(
            "list_services", ["provider": "github", "row_limit": 5])
        XCTAssertFalse(services.isError, "list_services failed: \(services.text)")
        XCTAssertTrue(
            services.text.contains("actions") || services.text.contains("apps"),
            "list_services did not include expected github services: \(services.text)")
    }

    func testReadOnlyModeRefusesMutations() async throws {
        try XCTSkipUnless(integrationEnabled, "set STACKQL_MCP_INTEGRATION=1 to run")

        let (opts, tmp) = try hermeticOptions(mode: .readOnly, auth: nil)
        let server = try await StackQLServer.start(opts)
        defer {
            Task { await server.stop() }
            try? FileManager.default.removeItem(at: tmp)
        }

        let result = try await server.call("run_mutation_query", [
            "sql": "INSERT INTO github.repos.repos(data__name) SELECT 'should-never-run'"
        ])
        XCTAssertTrue(result.isError, "read_only server did not flag mutation as error")
        XCTAssertTrue(
            result.text.contains("read_only"),
            "expected read_only refusal, got: \(result.text)")
    }
}
