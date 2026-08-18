using Microsoft.Extensions.AI;
using ModelContextProtocol.Client;

namespace StackQL.Mcp.AgentFramework;

/// <summary>
/// Bridges an embedded StackQL MCP server into Microsoft Agent Framework.
///
/// <para>
/// StackQL's tools come back from the MCP SDK as <see cref="McpClientTool"/>,
/// which derives from <see cref="AIFunction"/> (and thus <see cref="AITool"/>),
/// the exact abstraction Agent Framework agents consume. So the bridge is a thin,
/// stable adapter: list the tools, hand them to the agent's <c>tools</c> array.
/// </para>
///
/// <para>Wiring StackQL into an agent (Agent Framework 1.0, Microsoft.Agents.AI):</para>
/// <code>
/// await using var server = await StackqlMcp.CreateBuilder()
///     .WithMode(StackqlMode.ReadOnly)
///     .WithAuth("github", "null_auth")
///     .StartAsync();
///
/// var tools = await server.AsAgentToolsAsync();
///
/// // chatClient is any IChatClient (Azure OpenAI, Anthropic, etc.)
/// AIAgent agent = chatClient.CreateAIAgent(
///     instructions: "You answer cloud-posture questions using StackQL.",
///     tools: tools.ToArray());
///
/// var reply = await agent.RunAsync("Which GitHub repos lack branch protection?");
/// </code>
///
/// <para>
/// If you would rather let Agent Framework own the StackQL process via its native
/// MCP-over-stdio support, get the canonical argv from
/// <see cref="StackqlServer.ResolveCommandAsync"/> instead and register it as an
/// MCP server with the framework's own client.
/// </para>
/// </summary>
public static class StackqlAgentToolsExtensions
{
    /// <summary>
    /// Returns the StackQL server's tools as Agent Framework /
    /// Microsoft.Extensions.AI <see cref="AITool"/> instances, ready to pass to an
    /// agent's <c>tools</c> collection.
    /// </summary>
    public static async Task<IList<AITool>> AsAgentToolsAsync(
        this StackqlServer server,
        CancellationToken ct = default)
    {
        ArgumentNullException.ThrowIfNull(server);

        var tools = await server.ListToolsAsync(ct).ConfigureAwait(false);
        // McpClientTool : AIFunction : AITool - upcast directly, no wrapping.
        return tools.Cast<AITool>().ToList();
    }
}
