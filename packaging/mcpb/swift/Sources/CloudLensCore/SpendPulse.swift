import Foundation
import StackQLMCP

/// The spend pulse: top cost movers from AWS Cost Explorer. Needs AWS
/// credentials; when they are absent the pulse reports a neutral
/// "not configured" error rather than a false all-calm.
public struct SpendPulse: Pulse {
    public let kind: PulseKind = .spend
    /// Daily spend over this threshold (USD) for a service is an attention
    /// finding.
    public let alertThresholdUSD: Double

    public init(alertThresholdUSD: Double = 100) {
        self.alertThresholdUSD = alertThresholdUSD
    }

    public func run(_ server: StackQLServer) async -> PulseResult {
        // Top services by unblended cost over the trailing day. Exact
        // table/columns follow the aws.cost_explorer provider surface.
        let sql = """
        SELECT service, amount \
        FROM aws.cost_explorer.cost_and_usage \
        ORDER BY amount DESC
        """
        do {
            let result = try await server.call("run_select_query", stringArgs: ["query": sql])
            if result.isError {
                return PulseResult(kind: kind, findings: [],
                                   error: PulseErrors.classify(result.text, provider: "AWS"))
            }
            let rows = RowParser.rows(from: result.text)
            return PulseResult(kind: kind, findings: findings(from: rows, sql: sql))
        } catch {
            return PulseResult(kind: kind, findings: [], error: "\(error)")
        }
    }

    func findings(from rows: [[String: Any]], sql: String) -> [Finding] {
        var out: [Finding] = []
        // Top mover is always shown as context (info); anything over the
        // threshold is attention.
        for row in rows.prefix(5) {
            guard let service = row.string("service"),
                  let amount = row.double("amount") else { continue }
            let isHot = amount >= alertThresholdUSD
            out.append(Finding(
                kind: kind,
                severity: isHot ? .attention : .info,
                title: isHot
                    ? "High spend: \(service) $\(amount.rounded2())/day"
                    : "Top spend: \(service) $\(amount.rounded2())/day",
                detail: "Daily unblended cost for \(service).",
                sql: sql,
                key: "spend:\(service)"
            ))
        }
        return out
    }
}

extension Double {
    /// Two-decimal string for money display.
    func rounded2() -> String {
        String(format: "%.2f", self)
    }
}
