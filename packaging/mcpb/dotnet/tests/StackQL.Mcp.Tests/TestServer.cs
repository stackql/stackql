namespace StackQL.Mcp.Tests;

/// <summary>
/// Helpers for the integration tests that need a real StackQL binary. The binary
/// is only available when a developer/CI set STACKQL_MCP_BIN or STACKQL_MCP_BUNDLE,
/// or once pins.json carries real release shas. Until then these tests skip rather
/// than fail, so the unit suite stays green everywhere.
/// </summary>
internal static class TestServer
{
    /// <summary>
    /// True when a StackQL binary can be resolved without a verified download,
    /// i.e. an explicit binary or bundle override is present.
    /// </summary>
    public static bool BinaryAvailable =>
        HasFile(Environment.GetEnvironmentVariable("STACKQL_MCP_BIN")) ||
        HasFile(Environment.GetEnvironmentVariable("STACKQL_MCP_BUNDLE"));

    public const string SkipReason =
        "No StackQL binary available. Set STACKQL_MCP_BIN or STACKQL_MCP_BUNDLE to run integration tests.";

    private static bool HasFile(string? path) =>
        !string.IsNullOrWhiteSpace(path) && File.Exists(path);
}
