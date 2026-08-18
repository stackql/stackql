using System.Diagnostics;
using StackQL.Mcp;

// Conformance launcher, driven by packaging/mcpb/scripts/smoke-test.py:
//   python scripts/smoke-test.py --cmd "dotnet run --project dotnet/samples/Launcher --"
// Resolve (acquiring if needed) and run the server with inherited stdio.
// stdout belongs to the MCP protocol; diagnostics go to stderr.
var argv = await StackqlServer.ResolveCommandAsync(StackqlMode.ReadOnly);

var psi = new ProcessStartInfo(argv[0]) { UseShellExecute = false };
foreach (var a in argv.Skip(1).Concat(args))
{
    psi.ArgumentList.Add(a);
}

using var proc = Process.Start(psi)
    ?? throw new InvalidOperationException($"failed to start {argv[0]}");
await proc.WaitForExitAsync();
return proc.ExitCode;
