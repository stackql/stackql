import XCTest
@testable import CloudLensCore

final class FindingDiffTests: XCTestCase {
    private func finding(
        _ kind: PulseKind, _ sev: Severity, key: String, title: String = "t"
    ) -> Finding {
        Finding(kind: kind, severity: sev, title: title, detail: "d", sql: "SELECT 1", key: key)
    }

    func testStateUnknownWhenNoResults() {
        XCTAssertEqual(FindingDiff.state(for: []), .unknown)
    }

    func testStateUnknownWhenAllErrored() {
        let r = [
            PulseResult(kind: .spend, findings: [], error: "no creds"),
            PulseResult(kind: .exposure, findings: [], error: "no creds"),
        ]
        XCTAssertEqual(FindingDiff.state(for: r), .unknown)
    }

    func testStateCalmWhenSucceededWithNoAttention() {
        let r = [
            PulseResult(kind: .posture, findings: [finding(.posture, .info, key: "a")]),
            PulseResult(kind: .spend, findings: [], error: "no creds"),
        ]
        XCTAssertEqual(FindingDiff.state(for: r), .calm)
    }

    func testStateAttentionWins() {
        let r = [
            PulseResult(kind: .posture, findings: [
                finding(.posture, .info, key: "a"),
                finding(.posture, .attention, key: "b"),
            ]),
        ]
        XCTAssertEqual(FindingDiff.state(for: r), .attention)
    }

    func testNewFindingsAreThoseNotInPrevious() {
        let previous = [finding(.exposure, .attention, key: "bucket1")]
        let current = [
            finding(.exposure, .attention, key: "bucket1"),  // unchanged
            finding(.exposure, .attention, key: "bucket2"),  // new
        ]
        let fresh = FindingDiff.newFindings(current: current, previous: previous)
        XCTAssertEqual(fresh.map(\.key), ["bucket2"])
    }

    func testNewFindingsDistinguishesByKindAndKey() {
        // Same key under different pulses are different findings (id includes kind).
        let previous = [finding(.spend, .info, key: "x")]
        let current = [finding(.exposure, .info, key: "x")]
        let fresh = FindingDiff.newFindings(current: current, previous: previous)
        XCTAssertEqual(fresh.count, 1)
    }

    func testOrderedPutsAttentionFirstThenByTitle() {
        let r = [PulseResult(kind: .posture, findings: [
            finding(.posture, .info, key: "a", title: "zebra"),
            finding(.posture, .attention, key: "b", title: "yak"),
            finding(.posture, .info, key: "c", title: "ant"),
        ])]
        let ordered = FindingDiff.ordered(r)
        XCTAssertEqual(ordered.map(\.title), ["yak", "ant", "zebra"])
    }
}
