import Foundation

/// Packaging platform keys. These match the stackql-mcpb-packaging release
/// asset names (`stackql-mcp-<key>.mcpb`) and the shared binary cache layout
/// used by the npm and pypi wrappers and the Go/Rust/Kotlin siblings. Both
/// macOS architectures map to the single universal binary.
public enum Platform: String, Sendable, CaseIterable {
    case linuxX64 = "linux-x64"
    case linuxArm64 = "linux-arm64"
    case windowsX64 = "windows-x64"
    case darwinUniversal = "darwin-universal"

    /// The platform key for an arbitrary OS/arch pair, or nil when no binary
    /// is published for it. Split out from `current` so it is unit-testable
    /// without depending on the host.
    public static func key(os: HostOS, arch: HostArch) -> Platform? {
        switch (os, arch) {
        case (.linux, .x86_64): return .linuxX64
        case (.linux, .arm64): return .linuxArm64
        case (.windows, .x86_64): return .windowsX64
        case (.macOS, .x86_64), (.macOS, .arm64): return .darwinUniversal
        default: return nil
        }
    }

    /// The platform key for the running host. nil on a platform with no
    /// published binary (for example Windows on arm64).
    public static var current: Platform? {
        key(os: HostOS.current, arch: HostArch.current)
    }

    /// The file name the extracted server binary uses in the shared cache.
    var executableName: String {
        self == .windowsX64 ? "stackql.exe" : "stackql"
    }
}

/// Host operating system, resolved from compile-time conditionals so it can
/// also be supplied explicitly in tests.
public enum HostOS: Sendable {
    case macOS
    case linux
    case windows
    case other

    public static var current: HostOS {
        #if os(macOS)
        return .macOS
        #elseif os(Linux)
        return .linux
        #elseif os(Windows)
        return .windows
        #else
        return .other
        #endif
    }
}

/// Host CPU architecture.
public enum HostArch: Sendable {
    case x86_64
    case arm64
    case other

    public static var current: HostArch {
        #if arch(x86_64)
        return .x86_64
        #elseif arch(arm64)
        return .arm64
        #else
        return .other
        #endif
    }
}
