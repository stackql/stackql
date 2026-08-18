using StackQL.Mcp;
using StackQL.Mcp.AgentFramework;
using Xunit;

namespace StackQL.Mcp.Tests;

/// <summary>
/// The shared embedded-MCP-family conformance check, ported from the packaging
/// repo's smoke test: initialize -> tools/list contains the core tools -> pull the
/// github provider (null_auth) -> list_services returns real services. Skips when
/// no StackQL binary is available so the unit suite stays runnable everywhere.
/// </summary>
[Collection("integration")]
public class ConformanceTests
{
    private static readonly TimeSpan Timeout = TimeSpan.FromMinutes(3);

    [SkippableFact]
    public async Task Conformance_InitializeListPullList()
    {
        Skip.IfNot(TestServer.BinaryAvailable, TestServer.SkipReason);

        using var cts = new CancellationTokenSource(Timeout);
        var ct = cts.Token;

        await using var server = await StackqlMcp.CreateBuilder()
            .WithMode(StackqlMode.ReadOnly)
            .WithAuth("github", "null_auth")
            .StartAsync(ct);

        // tools/list contains the core tools.
        var tools = await server.ListToolsAsync(ct);
        var toolNames = tools.Select(t => t.Name).ToHashSet();
        Assert.Contains("list_providers", toolNames);
        Assert.Contains("list_services", toolNames);
        Assert.Contains("pull_provider", toolNames);

        // pull github (null_auth).
        var pull = await server.CallToolAsync("pull_provider",
            new Dictionary<string, object?> { ["provider"] = "github" }, ct);
        Assert.False(pull.IsError, $"pull_provider failed: {pull.Text}");

        // list_services returns real services for github.
        var services = await server.CallToolAsync("list_services",
            new Dictionary<string, object?> { ["provider"] = "github", ["row_limit"] = 5 }, ct);
        Assert.False(services.IsError, $"list_services failed: {services.Text}");
        Assert.False(string.IsNullOrWhiteSpace(services.Text));
    }

    [SkippableFact]
    public async Task AsAgentTools_ReturnsStackqlToolsAsAITools()
    {
        Skip.IfNot(TestServer.BinaryAvailable, TestServer.SkipReason);

        using var cts = new CancellationTokenSource(Timeout);
        await using var server = await StackqlMcp.CreateBuilder()
            .WithMode(StackqlMode.ReadOnly)
            .StartAsync(cts.Token);

        var aiTools = await server.AsAgentToolsAsync(cts.Token);

        Assert.NotEmpty(aiTools);
        Assert.Contains(aiTools, t => t.Name == "list_services");
    }
}
