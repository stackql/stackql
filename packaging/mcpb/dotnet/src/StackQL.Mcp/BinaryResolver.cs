using System.IO.Compression;
using System.Security.Cryptography;

namespace StackQL.Mcp;

/// <summary>
/// Resolves the path to a ready-to-run StackQL executable, acquiring it if
/// necessary. Resolution order (first match wins):
/// <list type="number">
///   <item>Explicit binary path passed to the builder (<c>WithBinary</c>).</item>
///   <item><c>STACKQL_MCP_BIN</c> environment variable.</item>
///   <item>Already-extracted binary in the shared cache.</item>
///   <item>Explicit bundle (<c>WithBundlePath</c>) or <c>STACKQL_MCP_BUNDLE</c>.</item>
///   <item>Vendored bundle embedded in the assembly.</item>
///   <item>Sidecar download of the platform bundle, verified against pins.</item>
/// </list>
/// All extraction targets the shared cache so other runtimes reuse it.
/// </summary>
internal sealed class BinaryResolver
{
    private readonly Pins _pins;
    private readonly HttpClient _http;

    public BinaryResolver(Pins pins, HttpClient http)
    {
        _pins = pins;
        _http = http;
    }

    /// <summary>
    /// Resolves the executable path, performing acquisition if needed.
    /// </summary>
    /// <param name="explicitBinary">Caller-supplied binary path, or null.</param>
    /// <param name="explicitBundle">Caller-supplied bundle path, or null.</param>
    public async Task<string> ResolveAsync(
        string? explicitBinary,
        string? explicitBundle,
        CancellationToken ct)
    {
        // 1. Explicit binary path.
        if (!string.IsNullOrWhiteSpace(explicitBinary))
        {
            return RequireExisting(explicitBinary, "WithBinary path");
        }

        // 2. STACKQL_MCP_BIN override.
        var envBin = Environment.GetEnvironmentVariable("STACKQL_MCP_BIN");
        if (!string.IsNullOrWhiteSpace(envBin))
        {
            return RequireExisting(envBin, "STACKQL_MCP_BIN");
        }

        var platformKey = Platform.CurrentKey();
        var cacheDir = Platform.BinCacheDir(_pins.Version, platformKey);
        var cachedBinary = Path.Combine(cacheDir, Platform.BinaryFileName());

        // 3. Already extracted in the shared cache (cross-runtime reuse). The
        //    bundle's entry point is server/<binary>, so look there too.
        if (TryFindCached(cacheDir, cachedBinary) is { } cached)
        {
            return cached;
        }

        // 4. Explicit bundle, then STACKQL_MCP_BUNDLE.
        var bundlePath = explicitBundle
            ?? Environment.GetEnvironmentVariable("STACKQL_MCP_BUNDLE");
        if (!string.IsNullOrWhiteSpace(bundlePath))
        {
            var bundle = RequireExisting(bundlePath, "bundle path");
            return ExtractBundle(await File.ReadAllBytesAsync(bundle, ct), cacheDir, verifySha: false);
        }

        // 5. Vendored bundle embedded in the assembly.
        var vendored = TryReadVendoredBundle();
        if (vendored is not null)
        {
            // Vendored bytes were verified at pack time; extract without re-verify.
            return ExtractBundle(vendored, cacheDir, verifySha: false);
        }

        // 6. Sidecar download, verified against the pinned sha256.
        var bytes = await DownloadBundleAsync(platformKey, ct);
        return ExtractBundle(bytes, cacheDir, verifySha: true, platformKey: platformKey);
    }

    private async Task<byte[]> DownloadBundleAsync(string platformKey, CancellationToken ct)
    {
        // platforms.json baseUrl is the releases.stackql.io front door; the
        // per-vector User-Agent lets the proxy attribute the download.
        var url = _pins.BundleUrlFor(platformKey);
        using var req = new HttpRequestMessage(HttpMethod.Get, url);
        req.Headers.TryAddWithoutValidation("User-Agent", _pins.UserAgent);

        using var resp = await _http.SendAsync(req, HttpCompletionOption.ResponseHeadersRead, ct);
        resp.EnsureSuccessStatusCode();
        return await resp.Content.ReadAsByteArrayAsync(ct);
    }

    /// <summary>
    /// Verifies (optionally), extracts the bundle into <paramref name="cacheDir"/>,
    /// and returns the executable path. Extraction is atomic-ish: write to a temp
    /// sibling dir then move into place, so a concurrent runtime never sees a
    /// half-extracted cache.
    /// </summary>
    private string ExtractBundle(
        byte[] bundleBytes,
        string cacheDir,
        bool verifySha,
        string? platformKey = null)
    {
        if (verifySha)
        {
            var expected = _pins.Sha256For(platformKey!);
            var actual = Sha256Hex(bundleBytes);
            if (!string.Equals(actual, expected, StringComparison.OrdinalIgnoreCase))
            {
                throw new InvalidOperationException(
                    $"StackQL bundle sha256 mismatch for {platformKey}: expected {expected}, got {actual}. " +
                    "Refusing to run an unverified binary. Set STACKQL_MCP_BIN or " +
                    "STACKQL_MCP_BUNDLE to run a local build instead.");
            }
        }

        var binaryName = Platform.BinaryFileName();
        var finalBinary = Path.Combine(cacheDir, binaryName);

        // Another process may have won the race while we downloaded.
        if (File.Exists(finalBinary))
        {
            return finalBinary;
        }

        var parent = Directory.GetParent(cacheDir)?.FullName
            ?? throw new InvalidOperationException($"Cache dir '{cacheDir}' has no parent.");
        Directory.CreateDirectory(parent);

        var tempDir = Path.Combine(parent, $".{Path.GetFileName(cacheDir)}.tmp-{Guid.NewGuid():N}");
        Directory.CreateDirectory(tempDir);
        try
        {
            using (var ms = new MemoryStream(bundleBytes, writable: false))
            using (var zip = new ZipArchive(ms, ZipArchiveMode.Read))
            {
                ExtractAll(zip, tempDir);
            }

            var extractedBinary = Path.Combine(tempDir, binaryName);
            if (!File.Exists(extractedBinary))
            {
                // Some bundles nest the binary under a bin/ dir; search for it.
                extractedBinary = Directory
                    .EnumerateFiles(tempDir, binaryName, SearchOption.AllDirectories)
                    .FirstOrDefault()
                    ?? throw new InvalidOperationException(
                        $"Bundle did not contain expected binary '{binaryName}'.");
            }

            MakeExecutable(extractedBinary);

            // Move temp into place. If we lost the race, just use the winner.
            if (!File.Exists(finalBinary))
            {
                try
                {
                    Directory.Move(tempDir, cacheDir);
                    tempDir = string.Empty; // moved; do not clean up
                }
                catch (IOException) when (Directory.Exists(cacheDir))
                {
                    // Lost the race; the winner's cache is authoritative.
                }
            }

            // Re-resolve from the final cache location (handles nested layouts).
            if (File.Exists(finalBinary))
            {
                return finalBinary;
            }

            return Directory
                .EnumerateFiles(cacheDir, binaryName, SearchOption.AllDirectories)
                .First();
        }
        finally
        {
            if (!string.IsNullOrEmpty(tempDir) && Directory.Exists(tempDir))
            {
                try { Directory.Delete(tempDir, recursive: true); } catch { /* best effort */ }
            }
        }
    }

    private static void ExtractAll(ZipArchive zip, string destDir)
    {
        var destFull = Path.GetFullPath(destDir);
        foreach (var entry in zip.Entries)
        {
            // Skip directory entries.
            if (string.IsNullOrEmpty(entry.Name))
            {
                continue;
            }

            var target = Path.GetFullPath(Path.Combine(destDir, entry.FullName));

            // Zip-slip guard: refuse entries that escape the destination.
            if (!target.StartsWith(destFull + Path.DirectorySeparatorChar, StringComparison.Ordinal)
                && !string.Equals(target, destFull, StringComparison.Ordinal))
            {
                throw new InvalidOperationException(
                    $"Bundle entry '{entry.FullName}' escapes the extraction directory.");
            }

            Directory.CreateDirectory(Path.GetDirectoryName(target)!);
            entry.ExtractToFile(target, overwrite: true);
        }
    }

    private static void MakeExecutable(string path)
    {
        if (OperatingSystem.IsWindows())
        {
            return;
        }

#if NET8_0_OR_GREATER
        var mode = File.GetUnixFileMode(path);
        File.SetUnixFileMode(path,
            mode | UnixFileMode.UserExecute | UnixFileMode.GroupExecute | UnixFileMode.OtherExecute);
#endif
    }

    private byte[]? TryReadVendoredBundle()
    {
        var asm = typeof(BinaryResolver).Assembly;
        using var stream = asm.GetManifestResourceStream("StackQL.Mcp.vendored-bundle.mcpb");
        if (stream is null)
        {
            return null;
        }

        using var ms = new MemoryStream();
        stream.CopyTo(ms);
        return ms.ToArray();
    }

    private static string? TryFindCached(string cacheDir, string preferred)
    {
        if (File.Exists(preferred))
        {
            return preferred;
        }

        if (!Directory.Exists(cacheDir))
        {
            return null;
        }

        return Directory
            .EnumerateFiles(cacheDir, Path.GetFileName(preferred), SearchOption.AllDirectories)
            .FirstOrDefault();
    }

    private static string RequireExisting(string path, string source)
    {
        if (!File.Exists(path))
        {
            throw new FileNotFoundException($"StackQL binary from {source} not found: {path}", path);
        }

        return path;
    }

    private static string Sha256Hex(byte[] bytes)
    {
        var hash = SHA256.HashData(bytes);
        return Convert.ToHexString(hash).ToLowerInvariant();
    }
}
