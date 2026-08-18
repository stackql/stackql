import Foundation

/// How loud a finding is. Drives the menu bar icon state and whether a
/// notification fires.
public enum Severity: Int, Sendable, Comparable, Codable {
    case info = 0
    case attention = 1

    public static func < (lhs: Severity, rhs: Severity) -> Bool {
        lhs.rawValue < rhs.rawValue
    }
}

/// Which pulse produced a finding.
public enum PulseKind: String, Sendable, Codable, CaseIterable {
    case spend
    case exposure
    case posture  // the github org-posture demo pulse (null_auth)
}

/// One observation from a pulse: a human title, a detail line, the SQL that
/// produced it (surfaced in notifications so the finding is auditable), and a
/// stable identity for day-over-day diffing.
public struct Finding: Sendable, Codable, Identifiable, Equatable {
    public let kind: PulseKind
    public let severity: Severity
    public let title: String
    public let detail: String
    /// The StackQL behind the finding. Shown in the notification body.
    public let sql: String
    /// Stable identity used to diff "new since yesterday". Two findings with
    /// the same key are the same underlying thing even if their detail (for
    /// example a dollar amount) changed.
    public let key: String

    public var id: String { "\(kind.rawValue):\(key)" }

    public init(
        kind: PulseKind,
        severity: Severity,
        title: String,
        detail: String,
        sql: String,
        key: String
    ) {
        self.kind = kind
        self.severity = severity
        self.title = title
        self.detail = detail
        self.sql = sql
        self.key = key
    }
}

/// The result of running one pulse.
public struct PulseResult: Sendable, Codable, Equatable {
    public let kind: PulseKind
    public let findings: [Finding]
    /// Non-nil when the pulse could not run (missing creds, server error).
    /// A failed pulse is surfaced as a neutral state, not a false "all calm".
    public let error: String?

    public init(kind: PulseKind, findings: [Finding], error: String? = nil) {
        self.kind = kind
        self.findings = findings
        self.error = error
    }

    /// The loudest severity among this pulse's findings, or nil if none.
    public var topSeverity: Severity? {
        findings.map(\.severity).max()
    }
}
