import Foundation

/// Version and sha256 pins for the StackQL MCP server bundle.
///
/// Source of truth is stackql/stackql-mcpb-packaging. The pins below are the
/// published sha256 of each release `.mcpb` bundle for `defaultVersion`,
/// copied from the release's `.sha256` assets. They are the same values the
/// Go and Rust siblings carry. The download path verifies bundles against
/// these before extracting; for any other version it fetches the pin from
/// the release at download time (see `BundleFetcher`).
public enum Pins {
    /// The stackql release this package version was developed and
    /// conformance-tested against.
    public static let defaultVersion = "0.10.500"

    /// sha256 of the release `.mcpb` bundle for `defaultVersion`, keyed by
    /// platform. Lowercase hex.
    public static let bundleSHA256: [Platform: String] = [
        .linuxX64: "6615737747156b1a8413a976afb23af2e7eec29ebc98a6f0a0f65d1b153c44be",
        .linuxArm64: "594bedbabc3096dc3563c907724e845ce0b61a67de4b3fed4158b40c0363786c",
        .windowsX64: "d2ce895e88f9c6b557df07073158629808f56d75598f3a701164d65506b791b0",
        .darwinUniversal: "4eed70af5cfa67295ae0b42fa3a6dca71ac9acabd0d67914fd96ad1247a9b4cc",
    ]
}
