// swift-tools-version: 6.1
//
// StackQLMCP: run an embedded StackQL MCP server from a binary located on
// disk (app bundle resource, shared cache, or pin-verified download),
// spawned over stdio and returned as a connected MCP client.
//
// Source of truth is packaging/mcpb/swift in stackql/stackql; the
// stackql/stackql-mcp-swift repository is the publish mirror SwiftPM
// resolves, tagged v<stackql version>. Sources/StackQLMCP/Resources/
// platforms.json and Sources/StackQLMCP/Version.swift are rendered by
// packaging/mcpb/scripts/render-platforms.sh (make swift-manifest).
//
// The single external dependency is the official MCP Swift SDK, which sets
// the floors: it is a swift-tools-version 6.1 package, so it needs Swift 6.1
// (Xcode 16.3+) and macOS 13.

import PackageDescription

let package = Package(
    name: "StackQLMCP",
    platforms: [
        .macOS(.v13)
    ],
    products: [
        .library(name: "StackQLMCP", targets: ["StackQLMCP"]),
        // Conformance launcher: what packaging/mcpb/scripts/smoke-test.py
        // --cmd "swift run stackql-mcp-launch" drives.
        .executable(name: "stackql-mcp-launch", targets: ["stackql-mcp-launch"]),
    ],
    dependencies: [
        .package(
            url: "https://github.com/modelcontextprotocol/swift-sdk.git",
            from: "0.11.0"
        )
    ],
    targets: [
        .target(
            name: "StackQLMCP",
            dependencies: [
                .product(name: "MCP", package: "swift-sdk")
            ],
            resources: [
                // The one pin source (version, baseUrl, per-platform sha256).
                .copy("Resources/platforms.json")
            ]
        ),
        .executableTarget(
            name: "stackql-mcp-launch",
            dependencies: ["StackQLMCP"]
        ),
        .testTarget(
            name: "StackQLMCPTests",
            dependencies: ["StackQLMCP"]
        ),
    ]
)
