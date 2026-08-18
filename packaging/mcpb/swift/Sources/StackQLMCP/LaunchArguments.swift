import Foundation

/// A provider auth document: provider name -> auth fields. Real auth
/// configs are nested string maps (for example
/// `["github": ["type": "null_auth"]]`), which keeps the type `Sendable`.
/// Serialised into the `--auth` flag.
public typealias AuthDocument = [String: [String: String]]

/// The StackQL MCP server safety mode. It bounds which tools the server will
/// actually execute. The server lists all tools regardless of mode but gates
/// calls, so `readOnly` lists `run_mutation_query` yet refuses to run it.
public enum Mode: String, Sendable {
    /// Metadata and SELECT tools only. The default. Anything more permissive
    /// is an explicit caller decision.
    case readOnly = "read_only"
    /// Additionally allows non-destructive mutations.
    case safe = "safe"
    /// Additionally allows deletes.
    case deleteSafe = "delete_safe"
    /// Removes mode restrictions.
    case fullAccess = "full_access"
}

public enum LaunchError: Error, CustomStringConvertible {
    case appRootNotAbsolute(String)
    case homeDirectoryUnavailable

    public var description: String {
        switch self {
        case .appRootNotAbsolute(let p):
            return "stackql mcp: approot must be absolute, got \(p)"
        case .homeDirectoryUnavailable:
            return "stackql mcp: could not resolve the home directory"
        }
    }
}

/// Builds the canonical launch arguments and resolves the canonical paths
/// the embedding contract requires.
public enum LaunchArguments {
    /// `~/.stackql`, the conventional application root shared with every
    /// other stackql distribution on the machine (provider registry cache,
    /// auth state).
    public static func defaultAppRoot() throws -> String {
        try homeDirectory().appendingPathComponent(".stackql").path
    }

    /// `~/.stackql/mcp-server-bin`, the shared binary cache root that the
    /// npm and pypi wrappers and the other language siblings also use, so
    /// multiple embedders on one machine share a single extraction.
    public static func defaultCacheDir() throws -> URL {
        try homeDirectory()
            .appendingPathComponent(".stackql")
            .appendingPathComponent("mcp-server-bin")
    }

    /// The canonical StackQL MCP launch arguments:
    ///
    ///     mcp --mcp.server.type=stdio --approot <approot>
    ///         --mcp.config {"server":{"mode":"<mode>","audit":{"disabled":true}}}
    ///         [--auth=<json>]
    ///
    /// Every path is absolute so the server is independent of the working
    /// directory. macOS MCP hosts routinely launch helpers with cwd `/`,
    /// which is read-only; a relative path there fails. This exact failure
    /// was found and fixed in Claude Desktop in June 2026, hence the
    /// cwd-independence requirement and the absolute-approot guard.
    ///
    /// `auth`, when non-nil, is serialised into a single `--auth=<json>`
    /// flag (for example the github `null_auth` fixture the conformance
    /// tests use).
    public static func build(
        mode: Mode = .readOnly,
        appRoot: String? = nil,
        auth: AuthDocument? = nil
    ) throws -> [String] {
        let root = try appRoot ?? defaultAppRoot()
        guard (root as NSString).isAbsolutePath else {
            throw LaunchError.appRootNotAbsolute(root)
        }

        var args = [
            "mcp",
            "--mcp.server.type=stdio",
            "--approot", root,
            "--mcp.config", try mcpConfigJSON(mode: mode),
        ]
        if let auth {
            args.append("--auth=" + (try jsonString(auth)))
        }
        return args
    }

    /// Serialise the `--mcp.config` document. Audit is disabled because an
    /// embedded server has no console session to audit to. Key order is
    /// fixed (server -> mode, audit -> disabled) so callers and tests can
    /// compare the literal string.
    static func mcpConfigJSON(mode: Mode) throws -> String {
        // Hand-built to guarantee key order; JSONSerialization does not
        // promise ordering for dictionaries.
        return "{\"server\":{\"mode\":\"\(mode.rawValue)\",\"audit\":{\"disabled\":true}}}"
    }

    static func jsonString(_ object: AuthDocument) throws -> String {
        let data = try JSONSerialization.data(
            withJSONObject: object,
            options: [.sortedKeys]
        )
        return String(decoding: data, as: UTF8.self)
    }

    private static func homeDirectory() throws -> URL {
        // FileManager.homeDirectoryForCurrentUser is reliable on macOS even
        // when HOME is unset (it consults the password database).
        let url = FileManager.default.homeDirectoryForCurrentUser
        guard !url.path.isEmpty else {
            throw LaunchError.homeDirectoryUnavailable
        }
        return url
    }
}
