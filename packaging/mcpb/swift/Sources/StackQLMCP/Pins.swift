import Foundation

/// Version, download base URL and sha256 pins for the StackQL MCP server
/// bundle.
///
/// Nothing is hand-written here: the values are the `platforms.json` package
/// resource, rendered by `packaging/mcpb/scripts/render-platforms.sh` from the
/// published `.mcpb.sha256` release assets - the same manifest every other
/// StackQL wrapper (npm, PyPI, the other SDKs) ships. Render it before
/// building: `make swift-manifest VERSION=X.Y.Z` (from packaging/mcpb).
public enum Pins {
    /// One platform's bundle name and lowercase-hex sha256.
    public struct PlatformPin: Sendable, Decodable {
        public let bundle: String
        public let sha256: String
    }

    /// The parsed platforms.json.
    public struct Manifest: Sendable, Decodable {
        /// The stackql release this package is version-locked to (no leading v).
        public let version: String
        /// Front door bundles are downloaded from
        /// (`https://releases.stackql.io/stackql/<version>`).
        public let baseUrl: String
        /// Platform key -> pin.
        public let platforms: [String: PlatformPin]
    }

    /// The embedded manifest. Missing or malformed is a build/packaging error,
    /// not a runtime condition, so it is fatal.
    public static let manifest: Manifest = {
        guard let url = Bundle.module.url(forResource: "platforms", withExtension: "json"),
              let data = try? Data(contentsOf: url),
              let manifest = try? JSONDecoder().decode(Manifest.self, from: data),
              !manifest.version.isEmpty, !manifest.baseUrl.isEmpty, !manifest.platforms.isEmpty
        else {
            fatalError(
                "stackql mcp: platforms.json resource missing or invalid - render it with "
                    + "'make swift-manifest VERSION=X.Y.Z' from packaging/mcpb")
        }
        return manifest
    }()

    /// The stackql release this package is version-locked to.
    public static var defaultVersion: String { manifest.version }

    /// Front door bundles are downloaded from.
    public static var baseURL: String { manifest.baseUrl }

    /// User-Agent sent to the download proxy, per vector and version.
    public static var userAgent: String { "stackql-mcp-server-swift/\(manifest.version)" }

    /// sha256 of the release `.mcpb` bundle for `defaultVersion`, keyed by
    /// platform. Lowercase hex.
    public static var bundleSHA256: [Platform: String] {
        var out: [Platform: String] = [:]
        for platform in Platform.allCases {
            if let pin = manifest.platforms[platform.rawValue] {
                out[platform] = pin.sha256.lowercased()
            }
        }
        return out
    }

    /// The pin for a platform, or nil when none is published.
    public static func pin(for platform: Platform) -> PlatformPin? {
        manifest.platforms[platform.rawValue]
    }

    /// The bundle asset name for a platform (`stackql-mcp-<key>.mcpb`).
    public static func bundleName(_ platform: Platform) -> String {
        pin(for: platform)?.bundle ?? "stackql-mcp-\(platform.rawValue).mcpb"
    }

    /// The download URL for a platform's bundle: `<baseUrl>/<bundle>`.
    public static func bundleURL(_ platform: Platform) -> URL {
        URL(string: "\(baseURL)/\(bundleName(platform))")!
    }
}
