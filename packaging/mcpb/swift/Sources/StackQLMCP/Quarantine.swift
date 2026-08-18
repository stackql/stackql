import Foundation

/// macOS quarantine handling for the download-at-runtime path.
///
/// A file written from a network download can carry the
/// `com.apple.quarantine` extended attribute, which makes Gatekeeper block
/// or warn on execution. Resources inside a notarised `.app` are never
/// quarantined, so this only matters when the binary is downloaded at
/// runtime rather than bundled. Clearing the attribute is best-effort: a
/// failure (for example on a non-Apple platform where the attribute does not
/// exist) is ignored.
enum Quarantine {
    static func clear(at url: URL) {
        #if os(macOS)
        let proc = Process()
        proc.executableURL = URL(fileURLWithPath: "/usr/bin/xattr")
        proc.arguments = ["-d", "com.apple.quarantine", url.path]
        proc.standardOutput = FileHandle.nullDevice
        proc.standardError = FileHandle.nullDevice
        // Missing attribute makes xattr exit non-zero; that is fine.
        try? proc.run()
        proc.waitUntilExit()
        #endif
    }
}
