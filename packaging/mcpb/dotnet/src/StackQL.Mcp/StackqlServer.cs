using ModelContextProtocol.Client;

namespace StackQL.Mcp;

/// <summary>
/// A running, embedded StackQL MCP server with a connected MCP client. Obtain one
/// via <see cref="StackqlMcp.CreateBuilder"/> and <see cref="StackqlMcpBuilder.StartAsync"/>.
/// Dispose to stop the server process and release the transport.
/// </summary>
public sealed class StackqlServer : IAsyncDisposable
{
    private readonly McpClient _client;
    private IReadOnlyList<McpClientTool>? _toolsCache;

    internal StackqlServer(McpClient client)
    {
        _client = client;
    }

    /// <summary>
    /// The connected MCP client (official C# SDK) for advanced use beyond the
    /// convenience methods on this type.
    /// </summary>
    public McpClient Client => _client;

    /// <summary>Lists the tools the StackQL server exposes (e.g. list_services,
    /// pull_provider, list_providers). Cached after the first call.</summary>
    public async Task<IReadOnlyList<McpClientTool>> ListToolsAsync(CancellationToken ct = default)
    {
        _toolsCache ??= (IReadOnlyList<McpClientTool>)await _client.ListToolsAsync(cancellationToken: ct)
            .ConfigureAwait(false);
        return _toolsCache;
    }

    /// <summary>Calls a StackQL tool by name with the given arguments.</summary>
    /// <param name="name">Tool name, e.g. <c>list_services</c>.</param>
    /// <param name="arguments">Tool arguments, or null for none.</param>
    public async Task<ToolResult> CallToolAsync(
        string name,
        IReadOnlyDictionary<string, object?>? arguments = null,
        CancellationToken ct = default)
    {
        var raw = await _client.CallToolAsync(name, arguments, cancellationToken: ct)
            .ConfigureAwait(false);
        return new ToolResult(raw);
    }

    /// <summary>
    /// Returns the argv an external harness (or Agent Framework's own MCP client)
    /// should spawn to run the StackQL server itself, so callers that prefer to own
    /// the process still get the canonical, cwd-independent arguments from us.
    /// Element 0 is the executable; the rest are arguments. The binary is resolved
    /// (and acquired if needed) the same way <see cref="StackqlMcpBuilder.StartAsync"/>
    /// resolves it.
    /// </summary>
    public static async Task<IReadOnlyList<string>> ResolveCommandAsync(
        StackqlMode mode = StackqlMode.ReadOnly,
        string? approot = null,
        string? binaryPath = null,
        string? bundlePath = null,
        HttpClient? httpClient = null,
        CancellationToken ct = default)
    {
        var ownsHttp = httpClient is null;
        var http = httpClient ?? new HttpClient();
        try
        {
            var resolver = new BinaryResolver(Pins.Load(), http);
            var exe = await resolver.ResolveAsync(binaryPath, bundlePath, ct).ConfigureAwait(false);
            var argv = new List<string> { exe };
            argv.AddRange(LaunchArgs.Build(mode, approot));
            return argv;
        }
        finally
        {
            if (ownsHttp)
            {
                http.Dispose();
            }
        }
    }

    /// <inheritdoc />
    public async ValueTask DisposeAsync()
    {
        await _client.DisposeAsync().ConfigureAwait(false);
    }
}
