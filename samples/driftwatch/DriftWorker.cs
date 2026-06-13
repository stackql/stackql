using System.Text.Json;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Options;
using StackQL.Mcp;

namespace Driftwatch;

public sealed class DriftOptions
{
    public const string SectionName = "Driftwatch";

    /// <summary>How often to run the drift suite.</summary>
    public TimeSpan Interval { get; set; } = TimeSpan.FromHours(6);

    /// <summary>Run once immediately on startup before the first interval wait.</summary>
    public bool RunOnStartup { get; set; } = true;

    /// <summary>Teams incoming-webhook URL. When empty, findings are logged.</summary>
    public string? TeamsWebhookUrl { get; set; }

    /// <summary>Path to the checks file (JSON array of DriftCheck).</summary>
    public string ChecksPath { get; set; } = "checks.json";

    /// <summary>Providers to pull at startup, e.g. ["github"]. null_auth providers
    /// (like github) need no credentials and make the sample runnable in CI.</summary>
    public string[] Providers { get; set; } = new[] { "github" };
}

/// <summary>
/// The driftwatch worker: embeds a read-only StackQL server, pulls the configured
/// providers, then runs the drift suite on a schedule and reports findings.
/// </summary>
public sealed class DriftWorker : BackgroundService
{
    private readonly ILogger<DriftWorker> _logger;
    private readonly DriftOptions _options;
    private readonly TeamsReporter _reporter;
    private readonly ILoggerFactory _loggerFactory;

    public DriftWorker(
        ILogger<DriftWorker> logger,
        IOptions<DriftOptions> options,
        TeamsReporter reporter,
        ILoggerFactory loggerFactory)
    {
        _logger = logger;
        _options = options.Value;
        _reporter = reporter;
        _loggerFactory = loggerFactory;
    }

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        _logger.LogInformation("driftwatch starting; embedding StackQL (read_only)");

        await using var server = await StackqlMcp.CreateBuilder()
            .WithMode(StackqlMode.ReadOnly)
            .WithAuth("github", "null_auth")
            .StartAsync(stoppingToken);

        await PullProvidersAsync(server, stoppingToken);

        var checks = LoadChecks();
        var engine = new DriftEngine(server, _loggerFactory.CreateLogger<DriftEngine>());

        if (_options.RunOnStartup)
        {
            await RunSuiteAsync(engine, checks, stoppingToken);
        }

        using var timer = new PeriodicTimer(_options.Interval);
        while (await timer.WaitForNextTickAsync(stoppingToken))
        {
            await RunSuiteAsync(engine, checks, stoppingToken);
        }
    }

    private async Task PullProvidersAsync(StackqlServer server, CancellationToken ct)
    {
        foreach (var provider in _options.Providers)
        {
            _logger.LogInformation("Pulling provider {Provider}", provider);
            var result = await server.CallToolAsync(
                "pull_provider",
                new Dictionary<string, object?> { ["provider"] = provider },
                ct);

            if (result.IsError)
            {
                _logger.LogWarning("pull_provider {Provider} failed: {Error}", provider, result.Text);
            }
        }
    }

    private async Task RunSuiteAsync(DriftEngine engine, IReadOnlyList<DriftCheck> checks, CancellationToken ct)
    {
        try
        {
            var results = await engine.RunAsync(checks, ct);
            await _reporter.ReportAsync(results, ct);
        }
        catch (OperationCanceledException) when (ct.IsCancellationRequested)
        {
            throw;
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Drift suite run failed");
        }
    }

    private IReadOnlyList<DriftCheck> LoadChecks()
    {
        var path = _options.ChecksPath;
        if (!File.Exists(path))
        {
            _logger.LogWarning("Checks file {Path} not found; no checks will run", path);
            return Array.Empty<DriftCheck>();
        }

        var json = File.ReadAllText(path);
        var checks = JsonSerializer.Deserialize<List<DriftCheck>>(json, new JsonSerializerOptions
        {
            PropertyNameCaseInsensitive = true,
        }) ?? new List<DriftCheck>();

        _logger.LogInformation("Loaded {Count} drift checks from {Path}", checks.Count, path);
        return checks;
    }
}
