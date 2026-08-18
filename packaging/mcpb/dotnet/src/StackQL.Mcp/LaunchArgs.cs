using System.Text;
using System.Text.Json;

namespace StackQL.Mcp;

/// <summary>
/// Builds the canonical, cwd-independent argv for launching the StackQL MCP
/// server over stdio. cwd-independence is mandatory: MCP hosts may launch with
/// cwd <c>/</c> (read-only on macOS), so the approot must always be absolute and
/// explicit. This was a real Claude Desktop bug fixed in June 2026.
/// </summary>
internal static class LaunchArgs
{
    /// <summary>
    /// Builds the argument list (excluding the executable itself) for:
    /// <code>
    /// mcp --mcp.server.type=stdio --approot &lt;approot&gt;
    ///     --mcp.config {"server": {"mode": "&lt;mode&gt;", "audit": {"disabled": true}}}
    /// </code>
    /// </summary>
    /// <param name="mode">Server safety mode (default read_only).</param>
    /// <param name="approot">
    /// Absolute approot directory. Defaults to <c>~/.stackql</c> when null/empty.
    /// </param>
    public static IReadOnlyList<string> Build(StackqlMode mode, string? approot = null)
    {
        var resolvedApproot = string.IsNullOrWhiteSpace(approot)
            ? Platform.StackqlHome()
            : Path.GetFullPath(approot);

        return new[]
        {
            "mcp",
            "--mcp.server.type=stdio",
            "--approot",
            resolvedApproot,
            "--mcp.config",
            BuildConfigJson(mode),
        };
    }

    /// <summary>
    /// The <c>--mcp.config</c> JSON payload. Audit is disabled because the
    /// embedded host owns audit/observability; the SQL inspectability is the
    /// audit trail for the embedded story.
    /// </summary>
    public static string BuildConfigJson(StackqlMode mode)
    {
        // Hand-build deterministic, compact JSON so the exact byte string is
        // assertable in unit tests and stable across runtimes.
        var sb = new StringBuilder();
        sb.Append("{\"server\": {\"mode\": ");
        sb.Append(JsonSerializer.Serialize(mode.ToWireValue()));
        sb.Append(", \"audit\": {\"disabled\": true}}}");
        return sb.ToString();
    }
}
