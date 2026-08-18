import Foundation

public enum BinaryError: Error, CustomStringConvertible {
    case noPublishedBinary(os: String, arch: String)
    case notFound(searched: [String])
    case shaMismatch(path: String, got: String, want: String)
    case cacheWriteFailed(String)

    public var description: String {
        switch self {
        case .noPublishedBinary(let os, let arch):
            return "stackql mcp: no published binary for \(os)/\(arch)"
        case .notFound(let searched):
            return "stackql mcp: server binary not found. searched: \(searched.joined(separator: ", "))"
        case .shaMismatch(let path, let got, let want):
            return "stackql mcp: sha256 mismatch for \(path): got \(got), want \(want)"
        case .cacheWriteFailed(let msg):
            return "stackql mcp: writing to cache: \(msg)"
        }
    }
}

/// Locates the StackQL server binary for the running host, in priority order:
///
/// 1. An explicit override path (`STACKQL_MCP_BINARY` env var or
///    `Options.binaryOverride`) - used by CI and tests.
/// 2. A binary bundled inside the calling app's `.app` (the shipping path:
///    `Contents/Resources/stackql` or `Contents/Helpers/stackql`). Resources
///    inside a notarised app are not quarantined and keep their own
///    Developer ID signature, so this path needs no download and no
///    quarantine handling.
/// 3. The shared on-disk cache `~/.stackql/mcp-server-bin/<version>/<key>/`,
///    populated earlier by this package or by the npm/pypi wrappers.
/// 4. Download the pin-verified release bundle into the shared cache.
///
/// Steps 1-3 are offline; step 4 reaches the network and is only taken when
/// the earlier steps miss. Shipping apps should bundle the binary (step 2)
/// so they never depend on step 4 at runtime.
public struct BinaryResolver {
    public let platform: Platform
    public let version: String
    public let cacheDir: URL

    /// Bundles to search for an embedded binary. Defaults to `Bundle.main`
    /// (the host app) plus `Bundle.module` so a resource shipped with this
    /// package is also found.
    let searchBundles: [Bundle]
    let binaryOverride: String?
    let fileManager: FileManager

    public init(
        platform: Platform? = nil,
        version: String = Pins.defaultVersion,
        cacheDir: URL? = nil,
        searchBundles: [Bundle]? = nil,
        binaryOverride: String? = nil,
        fileManager: FileManager = .default
    ) throws {
        guard let platform = platform ?? Platform.current else {
            throw BinaryError.noPublishedBinary(
                os: "\(HostOS.current)", arch: "\(HostArch.current)")
        }
        self.platform = platform
        self.version = version
        self.cacheDir = try cacheDir ?? LaunchArguments.defaultCacheDir()
        self.searchBundles = searchBundles ?? BinaryResolver.defaultBundles
        self.binaryOverride = binaryOverride
            ?? ProcessInfo.processInfo.environment["STACKQL_MCP_BINARY"]
        self.fileManager = fileManager
    }

    /// The shipping search path is the host app bundle. A library target
    /// without resources has no `Bundle.module`, so callers that bundle the
    /// binary elsewhere pass their own bundle via `searchBundles`.
    static var defaultBundles: [Bundle] {
        [Bundle.main]
    }

    /// The canonical extraction path inside the shared cache:
    /// `<cacheDir>/<version>/<platform-key>/stackql[.exe]`.
    public var cachedBinaryPath: URL {
        cacheDir
            .appendingPathComponent(version)
            .appendingPathComponent(platform.rawValue)
            .appendingPathComponent(platform.executableName)
    }

    /// Resolve the binary using only the offline steps (1-3). Returns nil if
    /// none hit, so callers can decide whether to download.
    public func locateOffline() -> URL? {
        // An empty override is treated as "no override": URL(fileURLWithPath:
        // "") resolves to the cwd, which must not be mistaken for the binary.
        if let override = binaryOverride, !override.isEmpty {
            let url = URL(fileURLWithPath: override)
            if fileManager.isExecutableFile(atPath: url.path) { return url }
        }
        if let bundled = bundledBinary() { return bundled }
        let cached = cachedBinaryPath
        if fileManager.isExecutableFile(atPath: cached.path) { return cached }
        return nil
    }

    /// Search the configured bundles for an embedded server binary at the
    /// conventional `.app` locations.
    func bundledBinary() -> URL? {
        let name = platform.executableName
        for bundle in searchBundles {
            // Contents/Resources/<name>
            if let url = bundle.url(forResource: name, withExtension: nil),
               fileManager.isExecutableFile(atPath: url.path) {
                return url
            }
            // Contents/Helpers/<name> (an auxiliary executable location some
            // apps prefer for embedded tools).
            let helpers = bundle.bundleURL
                .appendingPathComponent("Contents/Helpers")
                .appendingPathComponent(name)
            if fileManager.isExecutableFile(atPath: helpers.path) {
                return helpers
            }
        }
        return nil
    }

    /// Atomically install verified binary bytes into the shared cache and
    /// return the path. If a file with the expected sha is already present it
    /// is reused. Writes to a temp file in the target directory and renames
    /// into place, so concurrent installers race safely.
    @discardableResult
    public func installToCache(data: Data, expectedSHA256: String) throws -> URL {
        let target = cachedBinaryPath
        if SHA256Hash.fileMatches(target, expected: expectedSHA256) {
            return target
        }
        let got = SHA256Hash.hex(of: data)
        guard got == expectedSHA256.lowercased() else {
            throw BinaryError.shaMismatch(
                path: target.path, got: got, want: expectedSHA256.lowercased())
        }

        let dir = target.deletingLastPathComponent()
        do {
            try fileManager.createDirectory(
                at: dir, withIntermediateDirectories: true)
            let tmp = dir.appendingPathComponent(
                ".extract-\(ProcessInfo.processInfo.processIdentifier)")
            try data.write(to: tmp, options: .atomic)
            try fileManager.setAttributes(
                [.posixPermissions: 0o755], ofItemAtPath: tmp.path)
            // Replace any existing (possibly stale) file at target.
            if fileManager.fileExists(atPath: target.path) {
                try fileManager.removeItem(at: target)
            }
            try fileManager.moveItem(at: tmp, to: target)
        } catch {
            // Another installer may have won the race with a valid copy.
            if SHA256Hash.fileMatches(target, expected: expectedSHA256) {
                return target
            }
            throw BinaryError.cacheWriteFailed(error.localizedDescription)
        }
        return target
    }
}
