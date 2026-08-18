// Conformance launcher: resolve the StackQL MCP server binary (STACKQL_MCP_BIN
// / STACKQL_MCP_BUNDLE, an app-bundled copy, the shared cache, or a
// pin-verified download) and run it with the canonical launch arguments and
// inherited stdio. Extra argv is forwarded to the server verbatim.
//
// This is the command packaging/mcpb/scripts/smoke-test.py --cmd drives:
//
//   python scripts/smoke-test.py --cmd "swift run --package-path swift stackql-mcp-launch"

import Foundation
import StackQLMCP

var options = Options()
options.mode = .readOnly
options.extraArgs = Array(CommandLine.arguments.dropFirst())

do {
    let (path, args) = try await StackQLServer.resolveCommand(options)
    let process = Process()
    process.executableURL = path
    process.arguments = args
    process.standardInput = FileHandle.standardInput
    process.standardOutput = FileHandle.standardOutput
    process.standardError = FileHandle.standardError
    try process.run()
    process.waitUntilExit()
    exit(process.terminationStatus)
} catch {
    FileHandle.standardError.write(Data("stackql-mcp-launch: \(error)\n".utf8))
    exit(1)
}
