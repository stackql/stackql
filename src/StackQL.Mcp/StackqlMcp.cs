namespace StackQL.Mcp;

/// <summary>
/// Entry point for embedding the StackQL MCP server in a .NET application.
/// </summary>
/// <example>
/// <code>
/// await using var server = await StackqlMcp.CreateBuilder()
///     .WithMode(StackqlMode.ReadOnly)
///     .WithAuth("github", "null_auth")
///     .StartAsync();
///
/// var result = await server.CallToolAsync("list_services",
///     new() { ["provider"] = "github", ["row_limit"] = 5 });
/// Console.WriteLine(result.Text);
/// </code>
/// </example>
public static class StackqlMcp
{
    /// <summary>Creates a new builder with read-only mode as the default.</summary>
    public static StackqlMcpBuilder CreateBuilder() => new();
}
