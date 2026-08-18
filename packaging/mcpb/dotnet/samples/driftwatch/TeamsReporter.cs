using System.Text;
using System.Text.Json;
using Microsoft.Extensions.Logging;

namespace Driftwatch;

/// <summary>
/// Posts a drift report as a Teams Adaptive Card to an incoming-webhook URL. Each
/// finding renders the SQL that produced it, so the card doubles as an audit trail.
/// When no webhook is configured, the report is written to the log instead, so the
/// sample runs end-to-end with zero external setup.
/// </summary>
public sealed class TeamsReporter
{
    private readonly HttpClient _http;
    private readonly ILogger<TeamsReporter> _logger;
    private readonly string? _webhookUrl;

    public TeamsReporter(HttpClient http, ILogger<TeamsReporter> logger, string? webhookUrl)
    {
        _http = http;
        _logger = logger;
        _webhookUrl = webhookUrl;
    }

    public async Task ReportAsync(IReadOnlyList<DriftResult> results, CancellationToken ct)
    {
        var withFindings = results.Where(r => r.HasFindings).ToList();
        var failed = results.Where(r => r.Failed).ToList();

        if (string.IsNullOrWhiteSpace(_webhookUrl))
        {
            LogReport(results, withFindings, failed);
            return;
        }

        var card = BuildCard(results, withFindings, failed);
        using var content = new StringContent(card, Encoding.UTF8, "application/json");
        using var resp = await _http.PostAsync(_webhookUrl, content, ct);

        if (!resp.IsSuccessStatusCode)
        {
            var body = await resp.Content.ReadAsStringAsync(ct);
            _logger.LogWarning("Teams webhook returned {Status}: {Body}", resp.StatusCode, body);
        }
        else
        {
            _logger.LogInformation(
                "Posted drift card: {Findings} checks with findings, {Failed} failed",
                withFindings.Count, failed.Count);
        }
    }

    private void LogReport(
        IReadOnlyList<DriftResult> all,
        IReadOnlyList<DriftResult> withFindings,
        IReadOnlyList<DriftResult> failed)
    {
        _logger.LogInformation(
            "Drift report: {Total} checks, {Clean} clean, {Drift} with findings, {Failed} failed",
            all.Count, all.Count - withFindings.Count - failed.Count, withFindings.Count, failed.Count);

        foreach (var r in withFindings)
        {
            _logger.LogWarning(
                "[{Severity}] {Title}: {Count} finding(s)\n  SQL: {Sql}",
                r.Check.Severity, r.Check.Title, r.Findings.Count, r.Check.Sql);
        }
    }

    /// <summary>
    /// Builds the MessageCard envelope Teams incoming webhooks accept, embedding an
    /// Adaptive Card in the attachment. Hand-built so the sample carries no card SDK.
    /// </summary>
    private static string BuildCard(
        IReadOnlyList<DriftResult> all,
        IReadOnlyList<DriftResult> withFindings,
        IReadOnlyList<DriftResult> failed)
    {
        var clean = all.Count - withFindings.Count - failed.Count;
        var body = new List<object>
        {
            new
            {
                type = "TextBlock",
                size = "Large",
                weight = "Bolder",
                text = withFindings.Count == 0 ? "driftwatch: no drift detected" : "driftwatch: drift detected",
            },
            new
            {
                type = "TextBlock",
                isSubtle = true,
                wrap = true,
                text = $"{all.Count} checks - {clean} clean, {withFindings.Count} with findings, {failed.Count} errored",
            },
        };

        foreach (var r in withFindings)
        {
            body.Add(new
            {
                type = "TextBlock",
                weight = "Bolder",
                wrap = true,
                color = SeverityColor(r.Check.Severity),
                text = $"{r.Check.Title} ({r.Findings.Count})",
            });
            body.Add(new
            {
                type = "TextBlock",
                wrap = true,
                isSubtle = true,
                fontType = "Monospace",
                text = r.Check.Sql,
            });
            body.Add(new
            {
                type = "TextBlock",
                wrap = true,
                text = SummarizeFindings(r.Findings),
            });
        }

        var adaptiveCard = new
        {
            type = "AdaptiveCard",
            schema = "http://adaptivecards.io/schemas/adaptive-card.json",
            version = "1.4",
            body,
        };

        var envelope = new
        {
            type = "message",
            attachments = new[]
            {
                new
                {
                    contentType = "application/vnd.microsoft.card.adaptive",
                    content = adaptiveCard,
                },
            },
        };

        // Rename "schema" -> "$schema" which the C# anonymous type cannot express.
        var json = JsonSerializer.Serialize(envelope);
        return json.Replace("\"schema\":", "\"$schema\":");
    }

    private static string SeverityColor(string severity) => severity.ToLowerInvariant() switch
    {
        "high" => "Attention",
        "warning" => "Warning",
        _ => "Default",
    };

    private static string SummarizeFindings(IReadOnlyList<IReadOnlyDictionary<string, object?>> findings)
    {
        // Show up to the first 5 findings as compact key=value lines.
        var lines = findings.Take(5).Select(f =>
            "- " + string.Join(", ", f.Select(kv => $"{kv.Key}={kv.Value}")));
        var text = string.Join("\n", lines);
        if (findings.Count > 5)
        {
            text += $"\n- ... and {findings.Count - 5} more";
        }

        return text;
    }
}
