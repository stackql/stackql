import SwiftUI
import CloudLensCore

/// The popover content: overall state header, the three pulses grouped, and a
/// refresh control. Each finding shows its title, detail, and the SQL behind
/// it (the same SQL that rides along in the notification).
struct PopoverView: View {
    @ObservedObject var vm: SentinelViewModel

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            header
            Divider()
            if vm.findings.isEmpty {
                Text(vm.isRunning ? "Checking..." : "No findings yet.")
                    .foregroundStyle(.secondary)
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: 10) {
                        ForEach(PulseKind.allCases, id: \.self) { kind in
                            pulseSection(kind)
                        }
                    }
                }
                .frame(maxHeight: 320)
            }
            Divider()
            footer
        }
        .padding(14)
        .frame(width: 360)
    }

    private var header: some View {
        HStack(spacing: 8) {
            Image(systemName: vm.stateSymbol)
                .foregroundStyle(color(for: vm.state))
            Text(vm.stateLabel).font(.headline)
            Spacer()
            if vm.isRunning { ProgressView().controlSize(.small) }
        }
    }

    private func pulseSection(_ kind: PulseKind) -> some View {
        let items = vm.findings.filter { $0.kind == kind }
        return Group {
            if !items.isEmpty {
                VStack(alignment: .leading, spacing: 6) {
                    Text(title(for: kind))
                        .font(.caption).bold()
                        .foregroundStyle(.secondary)
                    ForEach(items) { finding in
                        findingRow(finding)
                    }
                }
            }
        }
    }

    private func findingRow(_ finding: Finding) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            HStack(alignment: .top, spacing: 6) {
                Image(systemName: finding.severity == .attention
                    ? "exclamationmark.triangle.fill" : "info.circle")
                    .foregroundStyle(finding.severity == .attention ? .orange : .secondary)
                    .font(.caption)
                VStack(alignment: .leading, spacing: 2) {
                    Text(finding.title).font(.callout)
                    Text(finding.detail).font(.caption).foregroundStyle(.secondary)
                    Text(finding.sql)
                        .font(.system(.caption2, design: .monospaced))
                        .foregroundStyle(.tertiary)
                        .textSelection(.enabled)
                }
            }
        }
    }

    private var footer: some View {
        HStack {
            if let lastRun = vm.lastRun {
                Text("Last checked \(lastRun.formatted(date: .omitted, time: .shortened))")
                    .font(.caption).foregroundStyle(.secondary)
            }
            Spacer()
            Button("Refresh") { Task { await vm.refresh() } }
                .disabled(vm.isRunning)
            Button("Quit") { NSApplication.shared.terminate(nil) }
        }
    }

    private func title(for kind: PulseKind) -> String {
        switch kind {
        case .spend: return "Spend pulse"
        case .exposure: return "Exposure pulse"
        case .posture: return "Org posture (github)"
        }
    }

    private func color(for state: SentinelState) -> Color {
        switch state {
        case .calm: return .green
        case .attention: return .orange
        case .unknown: return .secondary
        }
    }
}
