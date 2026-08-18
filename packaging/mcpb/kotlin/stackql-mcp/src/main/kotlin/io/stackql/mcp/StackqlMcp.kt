package io.stackql.mcp

import io.modelcontextprotocol.kotlin.sdk.client.Client
import io.modelcontextprotocol.kotlin.sdk.client.StdioClientTransport
import io.modelcontextprotocol.kotlin.sdk.types.Implementation
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.io.asSink
import kotlinx.io.asSource
import kotlinx.io.buffered
import kotlinx.serialization.json.JsonElement
import java.nio.file.Path
import kotlin.concurrent.thread

/**
 * The library version, reported as the MCP client version. Equal to the
 * stackql release the library embeds ([Pins.STACKQL_VERSION]).
 */
val LIBRARY_VERSION: String get() = Pins.STACKQL_VERSION

/**
 * A running embedded stackql MCP server and the connected client session.
 *
 * Obtain one with [StackqlMcp.builder]. The server starts in [Mode.ReadOnly]
 * unless a more permissive mode is requested explicitly. [close] shuts the
 * session down and terminates the server process.
 *
 * ```kotlin
 * val server = StackqlMcp.builder()
 *     .mode(Mode.ReadOnly)
 *     .auth(LaunchArgs.authFor("github", "null_auth"))
 *     .start()
 * val tools = server.client.listTools().tools
 * server.close()
 * ```
 */
class StackqlMcp internal constructor(
    /** The connected MCP client session. Use it for listTools, callTool, etc. */
    val client: Client,
    private val process: Process,
    /** Where the server binary was resolved to. */
    val binaryPath: Path,
    /** The safety mode the server was started with. */
    val mode: Mode,
) : AutoCloseable {

    /** Shut down the MCP session and terminate the server process. */
    override fun close() {
        runCatching { kotlinx.coroutines.runBlocking { client.close() } }
        process.destroy()
        if (!process.waitFor(5, java.util.concurrent.TimeUnit.SECONDS)) {
            process.destroyForcibly()
        }
    }

    /** Builder for [StackqlMcp]. Every field has a contract-compliant default. */
    class Builder {
        private var mode: Mode = Mode.ReadOnly
        private var appRoot: Path = LaunchArgs.defaultAppRoot()
        private var cacheRoot: Path = Cache.defaultRoot()
        private var auth: JsonElement? = null
        private var binary: Path? = null
        private var bundlePath: Path? = null
        private var extraArgs: List<String> = emptyList()
        private var clientInfo: Implementation =
            Implementation(name = "stackql-mcp-kotlin", version = LIBRARY_VERSION)
        private var stderr: ((String) -> Unit)? = null

        /** Safety mode. Defaults to [Mode.ReadOnly]; escalation is explicit. */
        fun mode(mode: Mode) = apply { this.mode = mode }

        /** StackQL application root (registry cache, auth state). Default `~/.stackql`. */
        fun appRoot(path: Path) = apply { this.appRoot = path }

        /** Binary extraction cache root. Default `~/.stackql/mcp-server-bin`. */
        fun cacheRoot(path: Path) = apply { this.cacheRoot = path }


        /** Provider auth override, serialized into `--auth`. See [LaunchArgs.authFor]. */
        fun auth(auth: JsonElement) = apply { this.auth = auth }

        /** Run this binary directly, skipping acquisition (overrides everything else). */
        fun binary(path: Path) = apply { this.binary = path }

        /** Extract this local `.mcpb` instead of downloading. */
        fun bundlePath(path: Path) = apply { this.bundlePath = path }

        /** Arguments appended verbatim after the canonical ones. */
        fun extraArgs(args: List<String>) = apply { this.extraArgs = args }

        /** MCP client identity sent in initialize. */
        fun clientInfo(impl: Implementation) = apply { this.clientInfo = impl }

        /** Sink for the server's stderr diagnostics, one call per line. Default: System.err. */
        fun onStderr(sink: (String) -> Unit) = apply { this.stderr = sink }

        /**
         * Resolve the exact command this builder would run, extracting the
         * binary first. Lets external conformance harnesses (the packaging
         * repo's smoke-test.py --cmd mode) exercise the launcher without
         * starting a session.
         */
        fun commandLine(): List<String> {
            val path = Acquire.resolveBinary(
                Acquire.Inputs(binary, bundlePath, cacheRoot),
            )
            return listOf(path.toString()) + LaunchArgs.build(mode, appRoot, auth) + extraArgs
        }

        /** Acquire the binary, spawn the server, complete the handshake, connect. */
        suspend fun start(): StackqlMcp = withContext(Dispatchers.IO) {
            val command = commandLine()
            val binaryPath = Path.of(command.first())

            // On Windows, the JDK launcher leaves the embedded double-quotes in
            // our JSON --auth / --mcp.config arguments un-escaped, so stackql's
            // CommandLineToArgvW parsing mangles them. Forcing Java's verbatim
            // quoting off (allowAmbiguousCommands=false) makes the launcher
            // escape the inner quotes; the property is read per spawn.
            if (Companion.isWindows()) {
                System.setProperty("jdk.lang.Process.allowAmbiguousCommands", "false")
            }

            // The server writes a log file to its cwd; point cwd at the
            // approot so it lands beside the rest of stackql's state rather
            // than in the caller's project directory. The launch args are
            // already absolute, so this does not affect resolution.
            val workingDir = appRoot.toFile().apply { mkdirs() }

            val process = ProcessBuilder(command)
                .directory(workingDir)
                .redirectErrorStream(false)
                .start()

            // Pump the server's stderr; stdout belongs to the protocol.
            val stderrSink = stderr ?: { line -> System.err.println(line) }
            thread(isDaemon = true, name = "stackql-mcp-stderr") {
                process.errorStream.bufferedReader().forEachLine(stderrSink)
            }

            val input = process.inputStream.asSource().buffered()
            val output = process.outputStream.asSink().buffered()
            val transport = StdioClientTransport(input = input, output = output)
            val client = Client(clientInfo)
            try {
                client.connect(transport)
            } catch (e: Exception) {
                process.destroyForcibly()
                throw StackqlMcpException("connecting to embedded server", e)
            }
            StackqlMcp(client, process, binaryPath, mode)
        }
    }

    companion object {
        fun builder(): Builder = Builder()

        internal fun isWindows(): Boolean =
            System.getProperty("os.name").lowercase().contains("windows")
    }
}
