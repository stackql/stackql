import Foundation
import StackQLMCP

/// The observable model behind CloudLens: it owns the embedded server, runs the
/// pulse suite on demand or on a schedule, derives the menu bar state, and
/// computes what is new since the previous run (which drives notifications).
///
/// It is a MainActor type so SwiftUI can bind to its published-style state
/// directly; the heavy work (spawning the server, running SQL) happens off the
/// main actor inside the awaited calls.
@MainActor
public final class SentinelModel {
    public private(set) var results: [PulseResult] = []
    public private(set) var state: SentinelState = .unknown
    public private(set) var lastRun: Date?
    public private(set) var isRunning = false

    /// Findings from the previous completed run, used to diff "new since last".
    private var previousFindings: [Finding] = []

    private let pulses: [any Pulse]
    private let serverOptions: Options
    /// Called with the findings that are new since the previous run, after each
    /// run. The app wires this to native notifications.
    private let onNewFindings: ([Finding]) -> Void
    /// Injectable clock so the model is testable without wall-clock time.
    private let now: () -> Date

    public init(
        pulses: [any Pulse],
        serverOptions: Options = Options(),
        now: @escaping () -> Date = Date.init,
        onNewFindings: @escaping ([Finding]) -> Void = { _ in }
    ) {
        self.pulses = pulses
        self.serverOptions = serverOptions
        self.now = now
        self.onNewFindings = onNewFindings
    }

    /// The findings to show in the popover, attention first.
    public var orderedFindings: [Finding] {
        FindingDiff.ordered(results)
    }

    /// Run the full pulse suite once: spawn the server, run every pulse, update
    /// state, diff against the previous run, and fire the new-findings hook.
    /// Re-entrancy is guarded so overlapping ticks don't double-run.
    public func runOnce() async {
        guard !isRunning else { return }
        isRunning = true
        defer { isRunning = false }

        let server: StackQLServer
        do {
            server = try await StackQLServer.start(serverOptions)
        } catch {
            // Could not even start the server: surface as an errored run for
            // every pulse so the UI shows unknown, not a false calm.
            results = pulses.map {
                PulseResult(kind: $0.kind, findings: [], error: "\(error)")
            }
            state = FindingDiff.state(for: results)
            lastRun = now()
            return
        }
        defer { Task { await server.stop() } }

        var collected: [PulseResult] = []
        for pulse in pulses {
            collected.append(await pulse.run(server))
        }

        let current = collected.flatMap(\.findings)
        let fresh = FindingDiff.newFindings(current: current, previous: previousFindings)

        results = collected
        state = FindingDiff.state(for: collected)
        lastRun = now()
        previousFindings = current

        if !fresh.isEmpty {
            onNewFindings(fresh)
        }
    }
}
