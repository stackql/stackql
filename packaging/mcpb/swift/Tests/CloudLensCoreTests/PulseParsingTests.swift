import XCTest
@testable import CloudLensCore

final class PulseParsingTests: XCTestCase {
    func testRowParserExtractsJSONArray() {
        let text = "some log line\n[{\"a\":1,\"b\":\"x\"},{\"a\":2,\"b\":\"y\"}]\ntrailing"
        let rows = RowParser.rows(from: text)
        XCTAssertEqual(rows.count, 2)
        XCTAssertEqual(rows[0].int("a"), 1)
        XCTAssertEqual(rows[1].string("b"), "y")
    }

    func testRowParserReturnsEmptyOnNoArray() {
        XCTAssertTrue(RowParser.rows(from: "no json here").isEmpty)
        XCTAssertTrue(RowParser.rows(from: "").isEmpty)
    }

    func testRowAccessorsCoerceTypes() {
        let row: [String: Any] = ["n": "42", "d": "1.5", "s": 7]
        XCTAssertEqual(row.int("n"), 42)
        XCTAssertEqual(row.double("d"), 1.5)
        XCTAssertEqual(row.string("s"), "7")
        XCTAssertNil(row.int("missing"))
    }

    func testPostureFindingsFlagArchivedPublicRepo() {
        let pulse = PosturePulse(org: "acme")
        let rows: [[String: Any]] = [
            ["name": "live", "visibility": "public", "archived": 0],
            ["name": "old", "visibility": "public", "archived": 1],
            ["name": "secret", "visibility": "private", "archived": 0],
        ]
        let findings = pulse.findings(from: rows, sql: "SELECT ...")
        // One summary (info) + one archived-public (attention).
        XCTAssertTrue(findings.contains { $0.severity == .attention && $0.title.contains("old") })
        let summary = findings.first { $0.key == "public-repo-count" }
        XCTAssertEqual(summary?.severity, .info)
        XCTAssertTrue(summary?.title.contains("2 public repos") ?? false)
    }

    func testSpendFindingsThresholdSeverity() {
        let pulse = SpendPulse(alertThresholdUSD: 100)
        let rows: [[String: Any]] = [
            ["service": "EC2", "amount": 250.0],
            ["service": "S3", "amount": 12.0],
        ]
        let findings = pulse.findings(from: rows, sql: "SELECT ...")
        let ec2 = findings.first { $0.key == "spend:EC2" }
        let s3 = findings.first { $0.key == "spend:S3" }
        XCTAssertEqual(ec2?.severity, .attention)
        XCTAssertEqual(s3?.severity, .info)
    }

    func testExposureFindingsAreAttention() {
        let pulse = ExposurePulse()
        let buckets: [[String: Any]] = [["bucket_name": "public-bucket"]]
        let sgs: [[String: Any]] = [
            ["group_id": "sg-123", "from_port": 0, "to_port": 65535]
        ]
        let b = pulse.bucketFindings(buckets, sql: "SELECT ...")
        let s = pulse.sgFindings(sgs, sql: "SELECT ...")
        XCTAssertEqual(b.first?.severity, .attention)
        XCTAssertTrue(b.first?.title.contains("public-bucket") ?? false)
        XCTAssertEqual(s.first?.severity, .attention)
        XCTAssertTrue(s.first?.title.contains("sg-123") ?? false)
    }

    func testPulseErrorsClassifyMissingCredentials() {
        let auth = PulseErrors.classify("error: missing AWS credentials", provider: "AWS")
        XCTAssertTrue(auth.contains("not configured"))
        let other = PulseErrors.classify("syntax error near SELECT", provider: "AWS")
        XCTAssertEqual(other, "syntax error near SELECT")
    }
}
