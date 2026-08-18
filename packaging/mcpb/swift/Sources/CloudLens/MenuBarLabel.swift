import SwiftUI

/// The menu bar glyph. Split into its own observing view so SwiftUI re-renders
/// the icon when the view model's @Published state changes.
struct MenuBarLabel: View {
    @ObservedObject var vm: SentinelViewModel

    var body: some View {
        Image(systemName: vm.stateSymbol)
    }
}
