// swift-tools-version: 6.1
//
// StackQLMCP: run an embedded StackQL MCP server from a binary located on
// disk (app bundle resource, shared cache, or download), spawned over stdio
// and returned as a connected MCP client.
//
// The single external dependency is the official MCP Swift SDK, which sets
// the floors: it is a swift-tools-version 6.1 package, so it needs Swift 6.1
// (Xcode 16.3+) and macOS 13. The project CLAUDE.md mentions Swift 5.10+,
// but the SDK requires 6.1, so that is the effective minimum.

import PackageDescription

let package = Package(
    name: "StackQLMCP",
    platforms: [
        .macOS(.v13)
    ],
    products: [
        .library(name: "StackQLMCP", targets: ["StackQLMCP"]),
        // CloudLens: the menu bar cloud sentinel demo app. Built as a SwiftPM
        // executable so CI can compile it; the signed/notarised .app is
        // assembled in the packaging step documented in docs/.
        .executable(name: "CloudLens", targets: ["CloudLens"]),
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
            ]
        ),
        .testTarget(
            name: "StackQLMCPTests",
            dependencies: ["StackQLMCP"]
        ),
        // CloudLensCore holds the testable app logic (pulses, finding diff,
        // the Anthropic agent client, Keychain access) with no SwiftUI, so it
        // can be unit-tested on CI without a GUI.
        .target(
            name: "CloudLensCore",
            dependencies: ["StackQLMCP"]
        ),
        // CloudLens is the thin SwiftUI MenuBarExtra shell: @main App, menu
        // bar icon state, popover, notifications.
        .executableTarget(
            name: "CloudLens",
            dependencies: ["CloudLensCore", "StackQLMCP"]
        ),
        .testTarget(
            name: "CloudLensCoreTests",
            dependencies: ["CloudLensCore"]
        ),
    ]
)
