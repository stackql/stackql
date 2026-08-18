import XCTest
@testable import CloudLensCore

final class AnthropicAgentTests: XCTestCase {
    func testExtractTextJoinsTextBlocks() throws {
        let json = """
        {"content":[{"type":"text","text":"line one"},
                    {"type":"tool_use","name":"x"},
                    {"type":"text","text":"line two"}]}
        """
        let text = try AnthropicAgent.extractText(from: Data(json.utf8))
        XCTAssertEqual(text, "line one\nline two")
    }

    func testExtractTextThrowsOnNoText() {
        let json = #"{"content":[{"type":"tool_use","name":"x"}]}"#
        XCTAssertThrowsError(try AnthropicAgent.extractText(from: Data(json.utf8)))
    }

    func testExtractTextThrowsOnMalformed() {
        XCTAssertThrowsError(try AnthropicAgent.extractText(from: Data("not json".utf8)))
    }

    func testBuildPromptListsFindings() {
        let findings = [
            Finding(kind: .exposure, severity: .attention, title: "Public bucket",
                    detail: "ACL public", sql: "SELECT 1", key: "b1")
        ]
        let prompt = AnthropicAgent.buildPrompt(findings)
        XCTAssertTrue(prompt.contains("Public bucket"))
        XCTAssertTrue(prompt.contains("attention"))
    }

    func testBuildPromptHandlesEmpty() {
        let prompt = AnthropicAgent.buildPrompt([])
        XCTAssertTrue(prompt.lowercased().contains("calm"))
    }

    func testSummariseThrowsWithoutKey() async {
        do {
            _ = try await AnthropicAgent().summarise([], apiKey: "")
            XCTFail("expected noAPIKey")
        } catch let e as AnthropicAgent.AgentError {
            guard case .noAPIKey = e else { return XCTFail("wrong error \(e)") }
        } catch {
            XCTFail("wrong error type \(error)")
        }
    }
}
