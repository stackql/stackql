import Foundation
import StackQLMCP

/// The github org-posture pulse: the demo pulse that runs with zero cloud
/// credentials (github in null_auth mode). It surfaces public-facing posture
/// signals for an org's repositories, standing in for the cloud exposure
/// pulse when no cloud creds are configured. This is what CI exercises.
public struct PosturePulse: Pulse {
    public let kind: PulseKind = .posture
    public let org: String

    public init(org: String) {
        self.org = org
    }

    public func run(_ server: StackQLServer) async -> PulseResult {
        // Ensure the github provider is available, then count public repos in
        // the org. A high public count is informational posture context, not
        // an alarm, so these are info-severity findings.
        let sql = """
        SELECT name, visibility, archived \
        FROM github.repos.repos \
        WHERE org = '\(org)'
        """
        do {
            _ = try await server.call("pull_provider", stringArgs: ["provider": "github"])
            let result = try await server.call("run_select_query", stringArgs: ["query": sql])
            if result.isError {
                return PulseResult(kind: kind, findings: [], error: result.text)
            }
            let rows = RowParser.rows(from: result.text)
            return PulseResult(kind: kind, findings: findings(from: rows, sql: sql))
        } catch {
            return PulseResult(kind: kind, findings: [], error: "\(error)")
        }
    }

    func findings(from rows: [[String: Any]], sql: String) -> [Finding] {
        let publicRepos = rows.filter { ($0.string("visibility") ?? "") == "public" }
        let total = rows.count
        var out: [Finding] = []

        out.append(Finding(
            kind: kind,
            severity: .info,
            title: "\(publicRepos.count) public repos in \(org)",
            detail: "\(total) repos scanned, \(publicRepos.count) public.",
            sql: sql,
            key: "public-repo-count"
        ))

        // An archived-but-still-public repo is worth a glance: stale code left
        // exposed. Flag as attention so the demo shows a non-calm state.
        let archivedPublic = publicRepos.filter { ($0.int("archived") ?? 0) == 1
            || ($0.string("archived") ?? "") == "true" }
        for repo in archivedPublic {
            let name = repo.string("name") ?? "(unknown)"
            out.append(Finding(
                kind: kind,
                severity: .attention,
                title: "Archived repo still public: \(name)",
                detail: "Archived repositories left public expose stale code.",
                sql: sql,
                key: "archived-public:\(name)"
            ))
        }
        return out
    }
}
