import Foundation
import StackQLMCP

/// The exposure pulse: newly public S3 buckets and wide-open security groups.
/// Needs AWS credentials; degrades to a neutral "not configured" error when
/// absent.
public struct ExposurePulse: Pulse {
    public let kind: PulseKind = .exposure

    public init() {}

    public func run(_ server: StackQLServer) async -> PulseResult {
        let bucketSQL = """
        SELECT bucket_name, acl_public \
        FROM aws.s3.bucket_acls \
        WHERE acl_public = true
        """
        let sgSQL = """
        SELECT group_id, cidr_ip, from_port, to_port \
        FROM aws.ec2.security_group_rules \
        WHERE cidr_ip = '0.0.0.0/0'
        """
        do {
            let buckets = try await server.call("run_select_query", stringArgs: ["query": bucketSQL])
            if buckets.isError {
                return PulseResult(kind: kind, findings: [],
                                   error: PulseErrors.classify(buckets.text, provider: "AWS"))
            }
            let sgs = try await server.call("run_select_query", stringArgs: ["query": sgSQL])
            if sgs.isError {
                return PulseResult(kind: kind, findings: [],
                                   error: PulseErrors.classify(sgs.text, provider: "AWS"))
            }
            var out = bucketFindings(RowParser.rows(from: buckets.text), sql: bucketSQL)
            out += sgFindings(RowParser.rows(from: sgs.text), sql: sgSQL)
            return PulseResult(kind: kind, findings: out)
        } catch {
            return PulseResult(kind: kind, findings: [], error: "\(error)")
        }
    }

    func bucketFindings(_ rows: [[String: Any]], sql: String) -> [Finding] {
        rows.compactMap { row in
            guard let name = row.string("bucket_name") else { return nil }
            return Finding(
                kind: kind,
                severity: .attention,
                title: "Public S3 bucket: \(name)",
                detail: "Bucket ACL grants public access.",
                sql: sql,
                key: "public-bucket:\(name)"
            )
        }
    }

    func sgFindings(_ rows: [[String: Any]], sql: String) -> [Finding] {
        rows.compactMap { row in
            guard let gid = row.string("group_id") else { return nil }
            let from = row.int("from_port").map(String.init) ?? "?"
            let to = row.int("to_port").map(String.init) ?? "?"
            return Finding(
                kind: kind,
                severity: .attention,
                title: "Open security group: \(gid) (\(from)-\(to))",
                detail: "Security group allows 0.0.0.0/0 ingress on \(from)-\(to).",
                sql: sql,
                key: "open-sg:\(gid):\(from)-\(to)"
            )
        }
    }
}

/// Shared classification of tool errors into user-facing pulse errors.
public enum PulseErrors {
    /// Turn a raw server error string into a concise message, recognising the
    /// missing-credentials case so the UI can show "not configured" instead
    /// of an alarming stack trace.
    public static func classify(_ raw: String, provider: String) -> String {
        let lower = raw.lowercased()
        if lower.contains("auth") || lower.contains("credential")
            || lower.contains("token") || lower.contains("unauthorized") {
            return "\(provider) not configured - add credentials in Settings."
        }
        return raw
    }
}
