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

/// Downloads the StackQL MCP `.mcpb` release bundle for the package's pinned
/// version, verifies it against the `platforms.json` sha256 pin, and extracts
/// the server binary named by the bundle manifest's `entry_point`. Mirrors the
/// Go `embed` fetch path and the Rust `download`/`bundle` modules.
///
/// Bundles are downloaded from the manifest's `baseUrl` (the
/// `https://releases.stackql.io` front door) with a per-vector User-Agent.
/// There is no version override: the package is version-locked to
/// `Pins.defaultVersion`.
public struct BundleFetcher: Sendable {
    let session: URLSession

    public init(session: URLSession = .shared) {
        self.session = session
    }

    static func bundleName(_ platform: Platform) -> String {
        Pins.bundleName(platform)
    }

    /// The download URL for a platform's bundle: `<baseUrl>/<bundle>`.
    public static func bundleURL(_ platform: Platform) -> URL {
        Pins.bundleURL(platform)
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

    /// The published sha256 pin for a platform's bundle, from platforms.json.
    public func resolvePin(platform: Platform) throws -> String {
        guard let pin = Pins.pin(for: platform) else {
            throw FetchError.malformedPin(asset: Self.bundleName(platform))
        }
        return pin.sha256.lowercased()
    }

    /// Download the bundle for `platform` at the pinned version, verify it
    /// against the published pin, and extract the server binary.
    public func fetch(platform: Platform) async throws -> Result {
        let pin = try resolvePin(platform: platform)
        let name = Self.bundleName(platform)
        let bundleData = try await get(Self.bundleURL(platform))
        let bundleSHA = SHA256Hash.hex(of: bundleData)
        guard bundleSHA == pin else {
            throw FetchError.checksumMismatch(asset: name, got: bundleSHA, want: pin)
        }
        let binary = try Self.extractEntryPoint(fromBundle: bundleData)
        return Result(
            data: binary,
            sha256: SHA256Hash.hex(of: binary),
            bundleSHA256: pin,
            version: Pins.defaultVersion,
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

    /// GET `url` and return the body, following redirects (the front door
    /// redirects to the release asset), with the per-vector User-Agent.
    func get(_ url: URL) async throws -> Data {
        do {
            var request = URLRequest(url: url)
            request.setValue(Pins.userAgent, forHTTPHeaderField: "User-Agent")
            let (data, response) = try await session.data(for: request)
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
