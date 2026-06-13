using System.Text.Json;
using Microsoft.Extensions.Logging;
using StackQL.Mcp;

namespace Driftwatch;

/// <summary>
/// Runs a set of <see cref="DriftCheck"/>s against an embedded StackQL server in
/// read-only mode and collects the findings. The server is held read_only for the
/// entire run; nothing here can mutate cloud state.
/// </summary>
public sealed class DriftEngine
{
    private readonly StackqlServer _server;
    private readonly ILogger<DriftEngine> _logger;

    public DriftEngine(StackqlServer server, ILogger<DriftEngine> logger)
    {
        _server = server;
        _logger = logger;
    }

    public async Task<IReadOnlyList<DriftResult>> RunAsync(
        IEnumerable<DriftCheck> checks,
        CancellationToken ct)
    {
        var results = new List<DriftResult>();

        foreach (var check in checks)
        {
            ct.ThrowIfCancellationRequested();
            results.Add(await RunOneAsync(check, ct));
        }

        return results;
    }

    private async Task<DriftResult> RunOneAsync(DriftCheck check, CancellationToken ct)
    {
        _logger.LogInformation("Running drift check {Id}", check.Id);

        try
        {
            var result = await _server.CallToolAsync(
                "run_select_query",
                new Dictionary<string, object?> { ["sql"] = check.Sql },
                ct);

            if (result.IsError)
            {
                return new DriftResult { Check = check, Error = result.Text };
            }

            var rows = ParseRows(result.Text);
            return new DriftResult { Check = check, Findings = rows };
        }
        catch (Exception ex) when (ex is not OperationCanceledException)
        {
            _logger.LogWarning(ex, "Drift check {Id} failed", check.Id);
            return new DriftResult { Check = check, Error = ex.Message };
        }
    }

    /// <summary>
    /// The run_select_query tool returns text containing a JSON object with a
    /// "rows" array. Parse defensively: the text may be the JSON itself or wrap it.
    /// </summary>
    private static IReadOnlyList<IReadOnlyDictionary<string, object?>> ParseRows(string text)
    {
        if (string.IsNullOrWhiteSpace(text))
        {
            return Array.Empty<IReadOnlyDictionary<string, object?>>();
        }

        var json = ExtractJsonObject(text);
        if (json is null)
        {
            return Array.Empty<IReadOnlyDictionary<string, object?>>();
        }

        using var doc = JsonDocument.Parse(json);
        if (!doc.RootElement.TryGetProperty("rows", out var rowsEl)
            || rowsEl.ValueKind != JsonValueKind.Array)
        {
            return Array.Empty<IReadOnlyDictionary<string, object?>>();
        }

        var rows = new List<IReadOnlyDictionary<string, object?>>();
        foreach (var rowEl in rowsEl.EnumerateArray())
        {
            if (rowEl.ValueKind != JsonValueKind.Object)
            {
                continue;
            }

            var row = new Dictionary<string, object?>(StringComparer.Ordinal);
            foreach (var prop in rowEl.EnumerateObject())
            {
                row[prop.Name] = prop.Value.ValueKind switch
                {
                    JsonValueKind.String => prop.Value.GetString(),
                    JsonValueKind.Number => prop.Value.GetRawText(),
                    JsonValueKind.True => true,
                    JsonValueKind.False => false,
                    JsonValueKind.Null => null,
                    _ => prop.Value.GetRawText(),
                };
            }

            rows.Add(row);
        }

        return rows;
    }

    private static string? ExtractJsonObject(string text)
    {
        var start = text.IndexOf('{');
        var end = text.LastIndexOf('}');
        if (start < 0 || end <= start)
        {
            return null;
        }

        return text.Substring(start, end - start + 1);
    }
}
