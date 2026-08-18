import Foundation

/// The overall state the menu bar icon reflects.
public enum SentinelState: String, Sendable {
    /// Nothing notable, or only informational findings.
    case calm
    /// At least one attention-level finding is present.
    case attention
    /// No pulse has run yet, or every pulse errored - we genuinely do not
    /// know, which must not look like "calm".
    case unknown
}

/// Pure functions over pulse results: deriving the icon state and computing
/// what is new since the previous run. Kept free of UI and IO so it is fully
/// unit-testable.
public enum FindingDiff {
    /// Derive the menu bar state from the latest pulse results. attention wins
    /// over calm; if there are no findings and at least one pulse succeeded it
    /// is calm; if every pulse errored (or there are none) it is unknown.
    public static func state(for results: [PulseResult]) -> SentinelState {
        if results.isEmpty { return .unknown }
        let anySucceeded = results.contains { $0.error == nil }
        if !anySucceeded { return .unknown }
        let loudest = results.compactMap(\.topSeverity).max()
        switch loudest {
        case .attention: return .attention
        case .info, nil: return .calm
        }
    }

    /// Findings present in `current` whose id was not present in `previous`.
    /// This is the "new since yesterday" set that drives notifications, so an
    /// unchanged finding does not re-notify every run.
    public static func newFindings(
        current: [Finding],
        previous: [Finding]
    ) -> [Finding] {
        let seen = Set(previous.map(\.id))
        return current.filter { !seen.contains($0.id) }
    }

    /// Flatten pulse results to their findings, attention first then by title,
    /// for stable display in the popover.
    public static func ordered(_ results: [PulseResult]) -> [Finding] {
        results.flatMap(\.findings).sorted { a, b in
            if a.severity != b.severity { return a.severity > b.severity }
            return a.title < b.title
        }
    }
}
