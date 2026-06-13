using System.Text.Json.Serialization;

namespace Driftwatch;

/// <summary>
/// A single drift check: a named SQL query whose returned rows are findings.
/// Every row the query returns is a thing that drifted; an empty result is "clean".
/// The SQL is carried into the report so each finding shows exactly how it was
/// produced - inspectability is the whole point.
/// </summary>
public sealed class DriftCheck
{
    /// <summary>Stable id, e.g. "github-repos-without-branch-protection".</summary>
    [JsonPropertyName("id")]
    public string Id { get; set; } = string.Empty;

    /// <summary>Human title shown on the card, e.g. "Repos missing branch protection".</summary>
    [JsonPropertyName("title")]
    public string Title { get; set; } = string.Empty;

    /// <summary>Severity label: info | warning | high.</summary>
    [JsonPropertyName("severity")]
    public string Severity { get; set; } = "warning";

    /// <summary>The StackQL SELECT that produces findings (one finding per row).</summary>
    [JsonPropertyName("sql")]
    public string Sql { get; set; } = string.Empty;
}

/// <summary>The outcome of running one <see cref="DriftCheck"/>.</summary>
public sealed class DriftResult
{
    public required DriftCheck Check { get; init; }

    /// <summary>Findings (rows). Empty means the check passed.</summary>
    public IReadOnlyList<IReadOnlyDictionary<string, object?>> Findings { get; init; } =
        Array.Empty<IReadOnlyDictionary<string, object?>>();

    /// <summary>Set when the check could not run (e.g. provider/query error).</summary>
    public string? Error { get; init; }

    public bool HasFindings => Findings.Count > 0;
    public bool Failed => Error is not null;
}
