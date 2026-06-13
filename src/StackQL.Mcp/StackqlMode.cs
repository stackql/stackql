namespace StackQL.Mcp;

/// <summary>
/// Server safety mode. Maps to the StackQL MCP server's <c>server.mode</c> config.
/// The default is <see cref="ReadOnly"/>; escalation is always an explicit caller
/// opt-in, never a default.
/// </summary>
public enum StackqlMode
{
    /// <summary>Queries only. No mutations of any kind. The default.</summary>
    ReadOnly,

    /// <summary>Creates and updates, but no deletes.</summary>
    Safe,

    /// <summary>Creates, updates, and deletes the server classes as safe.</summary>
    DeleteSafe,

    /// <summary>All operations, including unrestricted deletes.</summary>
    FullAccess,
}

internal static class StackqlModeExtensions
{
    /// <summary>
    /// The wire token the StackQL MCP server expects in its <c>--mcp.config</c>
    /// <c>server.mode</c> field.
    /// </summary>
    public static string ToWireValue(this StackqlMode mode) => mode switch
    {
        StackqlMode.ReadOnly => "read_only",
        StackqlMode.Safe => "safe",
        StackqlMode.DeleteSafe => "delete_safe",
        StackqlMode.FullAccess => "full_access",
        _ => throw new ArgumentOutOfRangeException(nameof(mode), mode, "Unknown StackQL mode."),
    };
}
