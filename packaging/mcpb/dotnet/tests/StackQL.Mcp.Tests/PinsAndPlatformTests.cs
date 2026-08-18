using StackQL.Mcp;
using Xunit;

namespace StackQL.Mcp.Tests;

public class PinsAndPlatformTests
{
    [Fact]
    public void Pins_Load_HasVersionAndAllFourPlatforms()
    {
        var pins = Pins.Load();

        Assert.False(string.IsNullOrWhiteSpace(pins.Version));
        Assert.Equal(4, pins.Sha256ByPlatform.Count);
        Assert.Contains(Platform.LinuxX64, pins.Sha256ByPlatform.Keys);
        Assert.Contains(Platform.LinuxArm64, pins.Sha256ByPlatform.Keys);
        Assert.Contains(Platform.WindowsX64, pins.Sha256ByPlatform.Keys);
        Assert.Contains(Platform.DarwinUniversal, pins.Sha256ByPlatform.Keys);
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
    public void Pins_BundleAssetName_FollowsConvention(string platform, string expected)
    {
        Assert.Equal(expected, Pins.BundleAssetName(platform));
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
