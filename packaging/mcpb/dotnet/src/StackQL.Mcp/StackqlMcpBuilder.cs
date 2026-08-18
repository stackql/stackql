using System.Text;
using System.Text.Json;
using ModelContextProtocol.Client;

namespace StackQL.Mcp;

/// <summary>
/// Fluent builder for an embedded StackQL MCP server. Defaults to
/// <see cref="StackqlMode.ReadOnly"/>; every escalation and override is explicit.
/// </summary>
public sealed class StackqlMcpBuilder
{
    private StackqlMode _mode = StackqlMode.ReadOnly;
    private string? _approot;
    private string? _binaryPath;
    private string? _bundlePath;
    private IReadOnlyList<string>? _command;
    private HttpClient? _httpClient;
    private readonly Dictionary<string, string> _auth = new(StringComparer.Ordinal);
    private readonly Dictionary<string, string> _env = new(StringComparer.Ordinal);

    internal StackqlMcpBuilder()
    {
    }

    /// <summary>Sets the server safety mode. Default is <see cref="StackqlMode.ReadOnly"/>.</summary>
    public StackqlMcpBuilder WithMode(StackqlMode mode)
    {
        _mode = mode;
        return this;
    }

    /// <summary>
    /// Configures auth for a provider. <paramref name="authType"/> is a StackQL
    /// auth type such as <c>null_auth</c>, <c>aws_signing</c>, <c>service_account</c>,
    /// or <c>access_token</c>. Credential material is read by StackQL from the
    /// environment; this only declares which auth type a provider uses.
    /// </summary>
    public StackqlMcpBuilder WithAuth(string provider, string authType)
    {
        ArgumentException.ThrowIfNullOrWhiteSpace(provider);
        ArgumentException.ThrowIfNullOrWhiteSpace(authType);
        _auth[provider] = authType;
        return this;
    }

    /// <summary>Overrides the approot directory. Defaults to <c>~/.stackql</c>.</summary>
    public StackqlMcpBuilder WithApproot(string approot)
    {
        _approot = approot;
        return this;
    }

    /// <summary>Uses an explicit StackQL binary, bypassing acquisition.</summary>
    public StackqlMcpBuilder WithBinary(string binaryPath)
    {
        _binaryPath = binaryPath;
        return this;
    }

    /// <summary>Uses an explicit <c>.mcpb</c> bundle, bypassing download.</summary>
    public StackqlMcpBuilder WithBundlePath(string bundlePath)
    {
        _bundlePath = bundlePath;
        return this;
    }

    /// <summary>
    /// Overrides the entire launch command (argv). Element 0 is the executable.
    /// For advanced hosts that fully own argv construction; bypasses
    /// <see cref="WithMode"/>/<see cref="WithApproot"/> arg generation.
    /// </summary>
    public StackqlMcpBuilder WithCommand(IReadOnlyList<string> command)
    {
        if (command is null || command.Count == 0)
        {
            throw new ArgumentException("Command must contain at least the executable.", nameof(command));
        }

        _command = command;
        return this;
    }

    /// <summary>Sets an extra environment variable for the server process.</summary>
    public StackqlMcpBuilder WithEnvironment(string name, string value)
    {
        ArgumentException.ThrowIfNullOrWhiteSpace(name);
        _env[name] = value;
        return this;
    }

    /// <summary>Supplies an <see cref="HttpClient"/> for sidecar downloads.</summary>
    public StackqlMcpBuilder WithHttpClient(HttpClient httpClient)
    {
        _httpClient = httpClient;
        return this;
    }

    /// <summary>Resolves the binary if needed, spawns the server, and connects.</summary>
    public async Task<StackqlServer> StartAsync(CancellationToken ct = default)
    {
        var (command, arguments) = await ResolveArgvAsync(ct).ConfigureAwait(false);

        var options = new StdioClientTransportOptions
        {
            Name = "stackql",
            Command = command,
            Arguments = arguments.ToArray(),
            EnvironmentVariables = BuildEnvironment(),
        };

        var transport = new StdioClientTransport(options);
        var client = await McpClient.CreateAsync(transport, cancellationToken: ct).ConfigureAwait(false);
        return new StackqlServer(client);
    }

    private async Task<(string Command, IReadOnlyList<string> Arguments)> ResolveArgvAsync(CancellationToken ct)
    {
        if (_command is not null)
        {
            return (_command[0], _command.Skip(1).ToList());
        }

        var ownsHttp = _httpClient is null;
        var http = _httpClient ?? new HttpClient();
        try
        {
            var resolver = new BinaryResolver(Pins.Load(), http);
            var exe = await resolver.ResolveAsync(_binaryPath, _bundlePath, ct).ConfigureAwait(false);
            return (exe, LaunchArgs.Build(_mode, _approot));
        }
        finally
        {
            if (ownsHttp)
            {
                http.Dispose();
            }
        }
    }

    private Dictionary<string, string?> BuildEnvironment()
    {
        var env = new Dictionary<string, string?>(StringComparer.Ordinal);

        // Declare provider auth types via StackQL's AUTH env var (JSON map).
        if (_auth.Count > 0)
        {
            env["AUTH"] = BuildAuthJson();
        }

        foreach (var kv in _env)
        {
            env[kv.Key] = kv.Value;
        }

        return env;
    }

    private string BuildAuthJson()
    {
        // StackQL AUTH format: { "<provider>": { "type": "<auth_type>" }, ... }
        var sb = new StringBuilder();
        sb.Append('{');
        var first = true;
        foreach (var kv in _auth)
        {
            if (!first)
            {
                sb.Append(',');
            }

            first = false;
            sb.Append(JsonSerializer.Serialize(kv.Key));
            sb.Append(":{\"type\":");
            sb.Append(JsonSerializer.Serialize(kv.Value));
            sb.Append('}');
        }

        sb.Append('}');
        return sb.ToString();
    }
}
