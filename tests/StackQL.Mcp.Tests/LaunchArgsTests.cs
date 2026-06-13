using StackQL.Mcp;
using Xunit;

namespace StackQL.Mcp.Tests;

public class LaunchArgsTests
{
    [Fact]
    public void Build_UsesCanonicalArgvShape()
    {
        var args = LaunchArgs.Build(StackqlMode.ReadOnly, approot: "/tmp/approot");

        Assert.Equal("mcp", args[0]);
        Assert.Equal("--mcp.server.type=stdio", args[1]);
        Assert.Equal("--approot", args[2]);
        Assert.Equal("--mcp.config", args[4]);
        Assert.Equal(6, args.Count);
    }

    [Fact]
    public void Build_ApprootIsAbsolute_EvenFromRelativeInput()
    {
        // cwd-independence is mandatory: relative input must be made absolute.
        var args = LaunchArgs.Build(StackqlMode.ReadOnly, approot: "relative/dir");
        var approot = args[3];

        Assert.True(Path.IsPathRooted(approot), $"approot '{approot}' must be absolute");
    }

    [Fact]
    public void Build_NullApproot_DefaultsToStackqlHome()
    {
        var args = LaunchArgs.Build(StackqlMode.ReadOnly, approot: null);
        var approot = args[3];

        Assert.EndsWith(".stackql", approot);
        Assert.True(Path.IsPathRooted(approot));
    }

    [Theory]
    [InlineData(StackqlMode.ReadOnly, "read_only")]
    [InlineData(StackqlMode.Safe, "safe")]
    [InlineData(StackqlMode.DeleteSafe, "delete_safe")]
    [InlineData(StackqlMode.FullAccess, "full_access")]
    public void ConfigJson_EncodesModeAndDisablesAudit(StackqlMode mode, string wire)
    {
        var json = LaunchArgs.BuildConfigJson(mode);

        Assert.Equal(
            $"{{\"server\": {{\"mode\": \"{wire}\", \"audit\": {{\"disabled\": true}}}}}}",
            json);
    }

    [Fact]
    public void ConfigJson_DefaultModeIsReadOnly()
    {
        // The default for the public API is ReadOnly; assert the wire token here.
        var json = LaunchArgs.BuildConfigJson(StackqlMode.ReadOnly);
        Assert.Contains("\"mode\": \"read_only\"", json);
    }
}
