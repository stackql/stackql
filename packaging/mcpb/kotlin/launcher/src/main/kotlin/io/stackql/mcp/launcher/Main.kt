package io.stackql.mcp.launcher

import io.stackql.mcp.Mode
import io.stackql.mcp.StackqlMcp
import kotlin.system.exitProcess

/**
 * Conformance launcher: resolve (acquiring if needed) and run the server with
 * the canonical launch arguments and inherited stdio. Extra argv is forwarded
 * to the server verbatim. This is the command
 * `packaging/mcpb/scripts/smoke-test.py --cmd` drives.
 */
fun main(args: Array<String>) {
    val command = StackqlMcp.builder()
        .mode(Mode.ReadOnly)
        .extraArgs(args.toList())
        .commandLine()
    // Same Windows quoting fix as StackqlMcp.start(): without it Java passes
    // the JSON --mcp.config / --auth arguments with their inner quotes
    // stripped by CommandLineToArgvW.
    if (System.getProperty("os.name").lowercase().contains("win")) {
        System.setProperty("jdk.lang.Process.allowAmbiguousCommands", "false")
    }
    val process = ProcessBuilder(command)
        .inheritIO()
        .start()
    exitProcess(process.waitFor())
}
