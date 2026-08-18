import Foundation
import MCP

/// Convenience wrappers over the raw MCP client for the common StackQL flows.
/// These forward to `client` and keep call sites concise; reach for `client`
/// directly for anything not covered here.
extension StackQLServer {
    /// List the tools the server exposes. The server lists every tool
    /// regardless of mode and gates execution by mode at call time.
    public func listToolNames() async throws -> [String] {
        let (tools, _) = try await client.listTools()
        return tools.map(\.name)
    }

    /// The result of a tool call: the concatenated text content and whether
    /// the server flagged it as an error.
    public struct ToolResult: Sendable {
        public let text: String
        public let isError: Bool
    }

    /// Call a tool and collect its text content. `arguments` uses the MCP
    /// `Value` type, which is expressible from literals, for example
    /// `["provider": "github", "row_limit": 5]`.
    @discardableResult
    public func call(_ name: String, _ arguments: [String: Value]? = nil) async throws -> ToolResult {
        let (content, isError) = try await client.callTool(name: name, arguments: arguments)
        return ToolResult(text: Self.joinText(content), isError: isError ?? false)
    }

    /// Convenience overload for the common all-string-arguments case, so
    /// callers can pass runtime `String` values without wrapping each in
    /// `Value` (a `[String: String]` literal of runtime strings does not
    /// implicitly convert to `[String: Value]`). Example:
    /// `try await server.call("run_select_query", stringArgs: ["query": sql])`.
    @discardableResult
    public func call(_ name: String, stringArgs: [String: String]) async throws -> ToolResult {
        let args = arguments(from: stringArgs)
        return try await call(name, args)
    }

    /// Map a string dictionary to MCP `Value` arguments. Uses the
    /// ExpressibleByStringLiteral initializer, which is part of Value's public
    /// API, rather than assuming a specific enum case name.
    static func valueArgs(_ stringArgs: [String: String]) -> [String: Value] {
        stringArgs.mapValues { Value(stringLiteral: $0) }
    }

    private func arguments(from stringArgs: [String: String]) -> [String: Value] {
        Self.valueArgs(stringArgs)
    }

    /// Concatenate the text parts of tool content, one per line.
    static func joinText(_ content: [Tool.Content]) -> String {
        var lines: [String] = []
        for item in content {
            if case let .text(text, _, _) = item {
                lines.append(text)
            }
        }
        return lines.joined(separator: "\n")
    }
}
