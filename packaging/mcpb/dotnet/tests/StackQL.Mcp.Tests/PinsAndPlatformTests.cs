using StackQL.Mcp;
using Xunit;

namespace StackQL.Mcp.Tests;

public class PinsAndPlatformTests
{
    [Fact]
    public void Pins_Load_HasVersionBaseUrlAndAllFourPlatforms()
    {
        var pins = Pins.Load();

        Assert.False(string.IsNullOrWhiteSpace(pins.Version));
        Assert.StartsWith("https://releases.stackql.io/stackql/", pins.BaseUrl);
        Assert.EndsWith(pins.Version, pins.BaseUrl);
        Assert.Equal(4, pins.Platforms.Count);
        foreach (var key in new[]
                 {
                     Platform.LinuxX64, Platform.LinuxArm64, Platform.WindowsX64, Platform.DarwinUniversal,
                 })
        {
            var sha = pins.Sha256For(key);
            Assert.Equal(64, sha.Length);
            Assert.True(sha.All(Uri.IsHexDigit), $"{key}: sha256 is not hex");
            Assert.Equal($"{pins.BaseUrl}/stackql-mcp-{key}.mcpb", pins.BundleUrlFor(key));
        }
        Assert.Equal($"stackql-mcp-server-dotnet/{pins.Version}", pins.UserAgent);
    }

    [Fact]
    public void Pins_Sha256For_UnknownPlatform_Throws()
    {
        var pins = Pins.Load();
        Assert.Throws<PlatformNotSupportedException>(() => pins.Sha256For("solaris-sparc"));
    }

    [Theory]
    [InlineData("linux-x64", "stackql-mcp-linux-x64.mcpb")]
    [InlineData("windows-x64", "stackql-mcp-windows-x64.mcpb")]
    [InlineData("darwin-universal", "stackql-mcp-darwin-universal.mcpb")]
    public void Pins_BundleNameFor_FollowsConvention(string platform, string expected)
    {
        Assert.Equal(expected, Pins.Load().BundleNameFor(platform));
    }

    [Fact]
    public void Platform_CurrentKey_IsOneOfTheFour()
    {
        var key = Platform.CurrentKey();
        Assert.Contains(key, new[]
        {
            Platform.LinuxX64, Platform.LinuxArm64, Platform.WindowsX64, Platform.DarwinUniversal,
        });
    }

    [Fact]
    public void Platform_BinCacheDir_MatchesSharedFamilyLayout()
    {
        var dir = Platform.BinCacheDir("0.10.500", "windows-x64");

        // ~/.stackql/mcp-server-bin/<version>/<platform-key>/
        var normalized = dir.Replace('\\', '/');
        Assert.Contains(".stackql/mcp-server-bin/0.10.500/windows-x64", normalized);
    }

    [Fact]
    public void Platform_BinaryFileName_HasExeOnWindowsOnly()
    {
        var name = Platform.BinaryFileName();
        if (OperatingSystem.IsWindows())
        {
            Assert.Equal("stackql.exe", name);
        }
        else
        {
            Assert.Equal("stackql", name);
        }
    }
}
