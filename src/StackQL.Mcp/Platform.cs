using System.Runtime.InteropServices;

namespace StackQL.Mcp;

/// <summary>
/// Resolves the StackQL platform key and shared cache locations. The platform
/// keys and the <c>~/.stackql/mcp-server-bin/&lt;version&gt;/&lt;platform-key&gt;/</c>
/// cache path are shared verbatim with the npm/pypi/go/rust/kotlin/swift wrappers,
/// so cross-runtime cache reuse works.
/// </summary>
internal static class Platform
{
    /// <summary>Platform keys defined by the packaging repo.</summary>
    public const string LinuxX64 = "linux-x64";
    public const string LinuxArm64 = "linux-arm64";
    public const string WindowsX64 = "windows-x64";
    public const string DarwinUniversal = "darwin-universal";

    /// <summary>
    /// The platform key for the current runtime, e.g. <c>windows-x64</c>.
    /// macOS collapses to <c>darwin-universal</c> regardless of arch.
    /// </summary>
    /// <exception cref="PlatformNotSupportedException">
    /// The current OS/arch combination has no published StackQL bundle.
    /// </exception>
    public static string CurrentKey()
    {
        if (RuntimeInformation.IsOSPlatform(OSPlatform.OSX))
        {
            return DarwinUniversal;
        }

        var arch = RuntimeInformation.OSArchitecture;

        if (RuntimeInformation.IsOSPlatform(OSPlatform.Windows))
        {
            return arch switch
            {
                Architecture.X64 => WindowsX64,
                _ => throw Unsupported("Windows", arch),
            };
        }

        if (RuntimeInformation.IsOSPlatform(OSPlatform.Linux))
        {
            return arch switch
            {
                Architecture.X64 => LinuxX64,
                Architecture.Arm64 => LinuxArm64,
                _ => throw Unsupported("Linux", arch),
            };
        }

        throw new PlatformNotSupportedException(
            $"No StackQL MCP bundle for OS '{RuntimeInformation.OSDescription}'.");
    }

    /// <summary>The executable name inside a bundle for the current OS.</summary>
    public static string BinaryFileName() =>
        RuntimeInformation.IsOSPlatform(OSPlatform.Windows) ? "stackql.exe" : "stackql";

    /// <summary>
    /// The StackQL home directory, <c>~/.stackql</c>. Honors no override directly;
    /// callers that need a custom approot pass it through the builder.
    /// </summary>
    public static string StackqlHome()
    {
        var home = Environment.GetFolderPath(Environment.SpecialFolder.UserProfile);
        if (string.IsNullOrEmpty(home))
        {
            // Fall back to HOME on platforms where UserProfile is empty.
            home = Environment.GetEnvironmentVariable("HOME")
                   ?? throw new InvalidOperationException("Cannot resolve the user home directory.");
        }

        return Path.Combine(home, ".stackql");
    }

    /// <summary>
    /// Shared binary cache directory for a given version + platform key:
    /// <c>~/.stackql/mcp-server-bin/&lt;version&gt;/&lt;platform-key&gt;/</c>.
    /// </summary>
    public static string BinCacheDir(string version, string platformKey) =>
        Path.Combine(StackqlHome(), "mcp-server-bin", version, platformKey);

    private static PlatformNotSupportedException Unsupported(string os, Architecture arch) =>
        new($"No StackQL MCP bundle for {os} on {arch}.");
}
