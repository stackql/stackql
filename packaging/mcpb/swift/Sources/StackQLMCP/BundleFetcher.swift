import Foundation

public enum FetchError: Error, CustomStringConvertible {
    case http(url: String, status: Int)
    case transport(url: String, message: String)
    case malformedPin(asset: String)
    case checksumMismatch(asset: String, got: String, want: String)
    case badBundle(String)
    case unsafeEntryPoint(String)

    public var description: String {
        switch self {
        case .http(let url, let status):
            return "stackql mcp: GET \(url): HTTP \(status)"
        case .transport(let url, let message):
            return "stackql mcp: GET \(url): \(message)"
        case .malformedPin(let asset):
            return "stackql mcp: malformed sha256 asset for \(asset)"
        case .checksumMismatch(let asset, let got, let want):
            return "stackql mcp: bundle sha256 mismatch for \(asset): got \(got), want \(want)"
        case .badBundle(let msg):
            return "stackql mcp: bad bundle: \(msg)"
        case .unsafeEntryPoint(let entry):
            return "stackql mcp: unsafe entry_point: \(entry)"
        }
    }
}

/// Downloads a StackQL MCP `.mcpb` release bundle, verifies it against the
/// published sha256 pin, and extracts the server binary named by the bundle
/// manifest's `entry_point`. Mirrors the Go `internal/fetch` package and the
/// Rust `download`/`bundle` modules.
///
/// Bundles are published as assets of the matching `stackql/stackql` release:
/// `https://github.com/stackql/stackql/releases/download/v<version>/stackql-mcp-<key>.mcpb`.
public struct BundleFetcher: Sendable {
    /// Asset download root.
    public static let releaseURLBase =
        "https://github.com/stackql/stackql/releases/download"

    let session: URLSession

    public init(session: URLSession = .shared) {
        self.session = session
    }

    static func bundleName(_ platform: Platform) -> String {
        "stackql-mcp-\(platform.rawValue).mcpb"
    }

    static func assetURL(version: String, name: String) -> URL {
        URL(string: "\(releaseURLBase)/v\(version)/\(name)")!
    }

    /// Result of a successful, verified fetch.
    public struct Result: Sendable {
        /// The extracted server binary bytes.
        public let data: Data
        /// The lowercase-hex sha256 of `data`.
        public let sha256: String
        /// The published pin the enclosing bundle was verified against.
        public let bundleSHA256: String
        public let version: String
        public let platform: Platform
    }

    /// Resolve the published sha256 pin for a bundle. Order: the in-package
    /// pin table (for `Pins.defaultVersion`), the consolidated
    /// `platforms.json` release asset if present, then the per-bundle
    /// `.sha256` asset.
    public func resolvePin(version: String, platform: Platform) async throws -> String {
        if version == Pins.defaultVersion, let pin = Pins.bundleSHA256[platform] {
            return pin
        }
        // try? flattens the nested optional, so this is a single String?.
        if let pin = try? await pinFromPlatformsJSON(version: version, platform: platform) {
            return pin
        }
        let name = Self.bundleName(platform) + ".sha256"
        let raw = try await get(Self.assetURL(version: version, name: name))
        let text = String(decoding: raw, as: UTF8.self)
        guard let first = text.split(whereSeparator: { $0 == " " || $0 == "\n" || $0 == "\t" }).first,
              first.count == 64 else {
            throw FetchError.malformedPin(asset: name)
        }
        return first.lowercased()
    }

    /// Try the consolidated `platforms.json` asset (planned by the packaging
    /// repo). Both a flat `{ "<key>": {"sha256": ...} }` map and a
    /// `{ "platforms": { ... } }` wrapper, with either an object or a bare
    /// string value, are accepted. A missing asset returns nil.
    func pinFromPlatformsJSON(version: String, platform: Platform) async throws -> String? {
        let raw = try await get(Self.assetURL(version: version, name: "platforms.json"))
        guard var doc = try JSONSerialization.jsonObject(with: raw) as? [String: Any] else {
            return nil
        }
        if let inner = doc["platforms"] as? [String: Any] {
            doc = inner
        }
        guard let entry = doc[platform.rawValue] else { return nil }
        if let obj = entry as? [String: Any], let sha = obj["sha256"] as? String, sha.count == 64 {
            return sha.lowercased()
        }
        if let bare = entry as? String, bare.count == 64 {
            return bare.lowercased()
        }
        return nil
    }

    /// Download the bundle for `version`/`platform`, verify it against the
    /// published pin, and extract the server binary.
    public func fetch(version: String, platform: Platform) async throws -> Result {
        let pin = try await resolvePin(version: version, platform: platform)
        let name = Self.bundleName(platform)
        let bundleData = try await get(Self.assetURL(version: version, name: name))
        let bundleSHA = SHA256Hash.hex(of: bundleData)
        guard bundleSHA == pin else {
            throw FetchError.checksumMismatch(asset: name, got: bundleSHA, want: pin)
        }
        let binary = try Self.extractEntryPoint(fromBundle: bundleData)
        return Result(
            data: binary,
            sha256: SHA256Hash.hex(of: binary),
            bundleSHA256: pin,
            version: version,
            platform: platform
        )
    }

    /// Extract the server binary from verified `.mcpb` bytes. The `.mcpb` is
    /// a zip containing `manifest.json` and the binary at the manifest's
    /// `server.entry_point`. Unzipping uses the system `unzip` (always
    /// present on macOS) so no zip dependency is needed.
    static func extractEntryPoint(fromBundle data: Data) throws -> Data {
        let work = FileManager.default.temporaryDirectory
            .appendingPathComponent("stackql-mcpb-\(UUID().uuidString)")
        try FileManager.default.createDirectory(at: work, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: work) }

        let archive = work.appendingPathComponent("bundle.mcpb")
        try data.write(to: archive)
        let extractDir = work.appendingPathComponent("x")
        try unzip(archive: archive, into: extractDir)

        let manifestURL = extractDir.appendingPathComponent("manifest.json")
        guard let manifestData = try? Data(contentsOf: manifestURL) else {
            throw FetchError.badBundle("manifest.json missing from bundle")
        }
        guard
            let manifest = try? JSONSerialization.jsonObject(with: manifestData) as? [String: Any],
            let server = manifest["server"] as? [String: Any],
            let entry = server["entry_point"] as? String,
            !entry.isEmpty
        else {
            throw FetchError.badBundle("manifest.json has no server.entry_point")
        }
        // Reject absolute or parent-traversing entry_point values before we
        // touch the filesystem with them.
        let comps = entry.split(separator: "/", omittingEmptySubsequences: false)
        guard !entry.hasPrefix("/"), !comps.contains(".."), !comps.contains("") else {
            throw FetchError.unsafeEntryPoint(entry)
        }
        let binaryURL = extractDir.appendingPathComponent(entry)
        guard let binary = try? Data(contentsOf: binaryURL) else {
            throw FetchError.badBundle("entry_point \(entry) not found in bundle")
        }
        return binary
    }

    private static func unzip(archive: URL, into dir: URL) throws {
        try FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        let proc = Process()
        proc.executableURL = URL(fileURLWithPath: "/usr/bin/unzip")
        proc.arguments = ["-q", "-o", archive.path, "-d", dir.path]
        proc.standardOutput = FileHandle.nullDevice
        proc.standardError = FileHandle.nullDevice
        try proc.run()
        proc.waitUntilExit()
        guard proc.terminationStatus == 0 else {
            throw FetchError.badBundle("unzip exited \(proc.terminationStatus)")
        }
    }

    /// GET `url` and return the body, following redirects (release asset URLs
    /// redirect to a CDN).
    func get(_ url: URL) async throws -> Data {
        do {
            let (data, response) = try await session.data(from: url)
            guard let http = response as? HTTPURLResponse else {
                throw FetchError.transport(url: url.absoluteString, message: "no HTTP response")
            }
            guard (200...299).contains(http.statusCode) else {
                throw FetchError.http(url: url.absoluteString, status: http.statusCode)
            }
            return data
        } catch let e as FetchError {
            throw e
        } catch {
            throw FetchError.transport(url: url.absoluteString, message: error.localizedDescription)
        }
    }
}
