using System.Reflection;
using System.Text.Json;

namespace StackQL.Mcp;

/// <summary>
/// The pinned StackQL version and per-platform sha256 bundle checksums baked
/// into the package. The values live in the embedded <c>pins.json</c> resource so
/// a version bump is a one-file change. Format mirrors the packaging repo's
/// planned consolidated <c>platforms.json</c> release asset.
/// </summary>
internal sealed record Pins(string Version, IReadOnlyDictionary<string, string> Sha256ByPlatform)
{
    private const string ResourceName = "StackQL.Mcp.pins.json";

    /// <summary>The sha256 (hex, lowercase) of the bundle for a platform key.</summary>
    public string Sha256For(string platformKey)
    {
        if (!Sha256ByPlatform.TryGetValue(platformKey, out var sha) || string.IsNullOrWhiteSpace(sha))
        {
            throw new PlatformNotSupportedException(
                $"No pinned StackQL bundle sha256 for platform '{platformKey}' in v{Version}.");
        }

        return sha;
    }

    /// <summary>The bundle asset name for a platform, e.g. stackql-mcp-windows-x64.mcpb.</summary>
    public static string BundleAssetName(string platformKey) => $"stackql-mcp-{platformKey}.mcpb";

    /// <summary>Loads and caches the embedded pins resource.</summary>
    public static Pins Load()
    {
        var asm = typeof(Pins).Assembly;
        using var stream = asm.GetManifestResourceStream(ResourceName)
            ?? throw new InvalidOperationException(
                $"Embedded resource '{ResourceName}' not found. Available: " +
                string.Join(", ", asm.GetManifestResourceNames()));

        var doc = JsonSerializer.Deserialize<PinsDto>(stream, JsonOpts)
            ?? throw new InvalidOperationException("pins.json deserialized to null.");

        if (string.IsNullOrWhiteSpace(doc.Version) || doc.Platforms is null)
        {
            throw new InvalidOperationException("pins.json is missing 'version' or 'platforms'.");
        }

        return new Pins(doc.Version, doc.Platforms);
    }

    private static readonly JsonSerializerOptions JsonOpts = new()
    {
        PropertyNameCaseInsensitive = true,
    };

    private sealed record PinsDto(string Version, Dictionary<string, string> Platforms);
}
