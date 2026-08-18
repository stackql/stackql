import Foundation
import Combine
import CloudLensCore
import StackQLMCP

/// SwiftUI-facing view model. It owns a `SentinelModel` (the framework-free
/// core) and republishes its state as `@Published` properties so the menu bar
/// and popover update. Kept in the app target so the core stays UI-agnostic.
@MainActor
final class SentinelViewModel: ObservableObject {
    @Published private(set) var state: SentinelState = .unknown
    @Published private(set) var findings: [Finding] = []
    @Published private(set) var lastRun: Date?
    @Published private(set) var isRunning = false

    private let model: SentinelModel

    init() {
        // Demo configuration: the github org-posture pulse in null_auth mode
        // (zero cloud creds) plus the cloud pulses, which degrade gracefully to
        // "not configured" until AWS credentials are added.
        let pulses: [any Pulse] = [
            PosturePulse(org: "stackql"),
            SpendPulse(),
            ExposurePulse(),
        ]
        var options = Options()
        options.mode = .readOnly
        options.auth = ["github": ["type": "null_auth"]]

        self.model = SentinelModel(
            pulses: pulses,
            serverOptions: options,
            onNewFindings: { fresh in
                Notifications.shared.post(fresh)
            }
        )
    }

    var stateSymbol: String {
        switch state {
        case .calm: return "checkmark.seal"
        case .attention: return "exclamationmark.triangle.fill"
        case .unknown: return "questionmark.circle"
        }
    }

    var stateLabel: String {
        switch state {
        case .calm: return "All calm"
        case .attention: return "Needs attention"
        case .unknown: return "Not checked yet"
        }
    }

    /// Run the pulse suite and copy the model's state across for the views.
    func refresh() async {
        isRunning = true
        await model.runOnce()
        state = model.state
        findings = model.orderedFindings
        lastRun = model.lastRun
        isRunning = model.isRunning
    }
}
