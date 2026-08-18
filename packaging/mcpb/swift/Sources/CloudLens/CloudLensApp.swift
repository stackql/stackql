import SwiftUI
import AppKit
import CloudLensCore

/// CloudLens: a menu bar cloud sentinel. The menu bar icon reflects overall
/// state (calm / attention / unknown); the popover shows the three pulses; new
/// attention findings fire native notifications that include the SQL behind
/// them.
///
/// Built as a SwiftPM executable so CI can compile it. The signed, notarised
/// .app that bundles the stackql binary is assembled in the packaging step
/// documented in docs/bundling-and-notarisation.md.
@main
struct CloudLensApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) private var delegate

    var body: some Scene {
        MenuBarExtra {
            PopoverView(vm: delegate.viewModel)
        } label: {
            // A dedicated observing view so the menu bar glyph updates when the
            // view model's @Published state changes.
            MenuBarLabel(vm: delegate.viewModel)
        }
        .menuBarExtraStyle(.window)
    }
}

/// Owns the view model and the unattended schedule. Using an app delegate keeps
/// the sentinel running on its timer whether or not the popover is open, and
/// gives a clean place to hang the launch sequence.
@MainActor
final class AppDelegate: NSObject, NSApplicationDelegate {
    let viewModel = SentinelViewModel()

    /// How often the pulse suite runs unattended.
    private static let refreshInterval: TimeInterval = 15 * 60
    private var timer: Timer?

    func applicationDidFinishLaunching(_ notification: Notification) {
        Notifications.shared.requestAuthorization()
        Task { await viewModel.refresh() }
        timer = Timer.scheduledTimer(
            withTimeInterval: Self.refreshInterval, repeats: true
        ) { [viewModel] _ in
            Task { @MainActor in await viewModel.refresh() }
        }
    }
}
