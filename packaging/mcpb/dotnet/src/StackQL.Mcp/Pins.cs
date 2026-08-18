using System.Text.Json;

namespace StackQL.Mcp;

/// <summary>
/// The pinned StackQL version, download base URL and per-platform bundle
/// sha256 checksums shipped inside the assembly. The values are the embedded
/// <c>platforms.json</c> resource, rendered by
/// <c>packaging/mcpb/scripts/render-platforms.sh</c> from the published
/// <c>.mcpb.sha256</c> release assets - the same manifest every other StackQL
/// wrapper (npm, PyPI, the other SDKs) ships. Nothing else in this package pins
/// a version or a hash.
/// </summary>
internal sealed record Pins(
    string Version,
    string BaseUrl,
    IReadOnlyDictionary<string, Pins.PlatformPin> Platforms)
{
    /// <summary>One platform's bundle name and sha256 (hex, lowercase).</summary>
    public sealed record PlatformPin(string Bundle, string Sha256);

    private const string ResourceName = "StackQL.Mcp.platforms.json";

    /// <summary>The User-Agent sent to the download proxy, per vector and version.</summary>
    public string UserAgent => $"stackql-mcp-server-dotnet/{Version}";

    /// <summary>The pin for a platform key, or <see cref="PlatformNotSupportedException"/>.</summary>
    public PlatformPin PinFor(string platformKey)
    {
        if (!Platforms.TryGetValue(platformKey, out var pin)
            || string.IsNullOrWhiteSpace(pin.Bundle)
            || string.IsNullOrWhiteSpace(pin.Sha256))
        {
            throw new PlatformNotSupportedException(
                $"No pinned StackQL bundle for platform '{platformKey}' in v{Version}.");
        }

        return pin;
    }

    /// <summary>The sha256 (hex, lowercase) of the bundle for a platform key.</summary>
    public string Sha256For(string platformKey) => PinFor(platformKey).Sha256;

    /// <summary>The bundle asset name for a platform, e.g. stackql-mcp-windows-x64.mcpb.</summary>
    public string BundleNameFor(string platformKey) => PinFor(platformKey).Bundle;

    /// <summary>The download URL for a platform's bundle: <c>&lt;baseUrl&gt;/&lt;bundle&gt;</c>.</summary>
    public string BundleUrlFor(string platformKey) => $"{BaseUrl}/{BundleNameFor(platformKey)}";

    private static readonly Lazy<Pins> Cached = new(LoadUncached);

    /// <summary>Loads (once) the embedded platforms.json resource.</summary>
    public static Pins Load() => Cached.Value;

    private static Pins LoadUncached()
    {
        var asm = typeof(Pins).Assembly;
        using var stream = asm.GetManifestResourceStream(ResourceName)
            ?? throw new InvalidOperationException(
                $"Embedded resource '{ResourceName}' not found (render it with " +
                "'make dotnet-manifest VERSION=X.Y.Z' from packaging/mcpb). Available: " +
                string.Join(", ", asm.GetManifestResourceNames()));

        var doc = JsonSerializer.Deserialize<ManifestDto>(stream, JsonOpts)
            ?? throw new InvalidOperationException("platforms.json deserialized to null.");

        if (string.IsNullOrWhiteSpace(doc.Version)
            || string.IsNullOrWhiteSpace(doc.BaseUrl)
            || doc.Platforms is null || doc.Platforms.Count == 0)
        {
            throw new InvalidOperationException(
                "platforms.json is missing 'version', 'baseUrl' or 'platforms'.");
        }

        var platforms = doc.Platforms.ToDictionary(
            kv => kv.Key,
            kv => new PlatformPin(kv.Value.Bundle, kv.Value.Sha256),
            StringComparer.Ordinal);
        return new Pins(doc.Version, doc.BaseUrl, platforms);
    }

    private static readonly JsonSerializerOptions JsonOpts = new()
    {
        PropertyNameCaseInsensitive = true,
    };

    private sealed record ManifestDto(string Version, string BaseUrl, Dictionary<string, PlatformDto> Platforms);

    private sealed record PlatformDto(string Bundle, string Sha256);
}
