import Foundation
import System
import MCP

/// Configuration for starting an embedded StackQL MCP server.
public struct Options: Sendable {
    /// Safety mode. Defaults to `.readOnly`; anything more permissive is an
    /// explicit caller decision.
    public var mode: Mode = .readOnly

    /// stackql release version to locate/download. Defaults to
    /// `Pins.defaultVersion`.
    public var version: String = Pins.defaultVersion

    /// Application root (provider registry cache, auth state). Defaults to
    /// `~/.stackql`. Must be absolute if set.
    public var appRoot: String? = nil

    /// Binary extraction cache root. Defaults to `~/.stackql/mcp-server-bin`.
    public var cacheDir: URL? = nil

    /// Provider auth document. When set, serialised into `--auth`, overriding
    /// environment auth. Example for the conformance fixture:
    /// `["github": ["type": "null_auth"]]`.
    public var auth: AuthDocument? = nil

    /// Explicit binary path, highest priority in resolution. Also settable
    /// via the `STACKQL_MCP_BINARY` environment variable.
    public var binaryOverride: String? = nil

    /// Whether to download the pin-verified bundle if the binary is not found
    /// offline (bundled in the app or already cached). Defaults to true.
    /// Shipping apps that bundle the binary can set this false to guarantee
    /// no runtime network access.
    public var allowDownload: Bool = true

    /// MCP client identity sent in `initialize`.
    public var clientName: String = "stackql-mcp-swift"
    public var clientVersion: String = "0.1.0"

    /// Extra arguments appended verbatim after the canonical arguments.
    public var extraArgs: [String] = []

    public init() {}
}

public enum ServerError: Error, CustomStringConvertible {
    case spawnFailed(String)
    case binaryUnavailable(String)

    public var description: String {
        switch self {
        case .spawnFailed(let m): return "stackql mcp: spawning server: \(m)"
        case .binaryUnavailable(let m): return "stackql mcp: \(m)"
        }
    }
}

/// A running embedded StackQL MCP server and the connected client session.
///
/// `StackQLServer.start` locates the server binary (bundled app resource,
/// shared cache, or pin-verified download), spawns it as an MCP stdio server
/// with the canonical launch arguments, performs the handshake, and returns
/// a connected instance. Call `stop()` when finished.
public final class StackQLServer: @unchecked Sendable {
    /// The connected MCP client. Use it for `listTools`, `callTool`, and the
    /// rest of the protocol surface.
    public let client: Client

    /// Where the server binary was resolved from.
    public let binaryPath: URL

    /// The mode the server was started with.
    public let mode: Mode

    private let process: Process

    private init(client: Client, binaryPath: URL, mode: Mode, process: Process) {
        self.client = client
        self.binaryPath = binaryPath
        self.mode = mode
        self.process = process
    }

    /// Resolve the binary path that `start` would use, without spawning.
    /// Exposed so external conformance harnesses (the packaging repo's
    /// `smoke-test.py --cmd` mode) can exercise the launcher.
    public static func resolveCommand(
        _ options: Options = Options()
    ) async throws -> (path: URL, args: [String]) {
        let resolver = try BinaryResolver(
            version: options.version,
            cacheDir: options.cacheDir,
            binaryOverride: options.binaryOverride
        )
        let path = try await resolveBinary(resolver: resolver, options: options)
        var args = try LaunchArguments.build(
            mode: options.mode, appRoot: options.appRoot, auth: options.auth)
        args.append(contentsOf: options.extraArgs)
        return (path, args)
    }

    /// Locate the binary offline, downloading the pin-verified bundle into
    /// the shared cache as a last resort when allowed.
    static func resolveBinary(resolver: BinaryResolver, options: Options) async throws -> URL {
        if let found = resolver.locateOffline() {
            return found
        }
        guard options.allowDownload else {
            throw ServerError.binaryUnavailable(
                "binary not found offline and download is disabled")
        }
        let fetcher = BundleFetcher()
        let result = try await fetcher.fetch(version: options.version, platform: resolver.platform)
        let installed = try resolver.installToCache(
            data: result.data, expectedSHA256: result.sha256)
        // A freshly downloaded file may carry com.apple.quarantine; clear it
        // so the spawned process is not blocked by Gatekeeper. Binaries that
        // came from inside a notarised app bundle are never quarantined, so
        // this only applies to the download path.
        Quarantine.clear(at: installed)
        return installed
    }

    /// Start the embedded server and return a connected instance.
    public static func start(_ options: Options = Options()) async throws -> StackQLServer {
        let (path, args) = try await resolveCommand(options)

        let process = Process()
        process.executableURL = path
        process.arguments = args

        // stdout carries the MCP protocol; stdin carries our requests. stderr
        // is diagnostics and is forwarded to the parent's stderr so it does
        // not pollute the protocol stream.
        let toServer = Pipe()    // parent writes -> server stdin
        let fromServer = Pipe()  // server stdout -> parent reads
        process.standardInput = toServer
        process.standardOutput = fromServer
        process.standardError = FileHandle.standardError

        do {
            try process.run()
        } catch {
            throw ServerError.spawnFailed(error.localizedDescription)
        }

        // The StdioTransport reads from the server's stdout and writes to the
        // server's stdin. Hand it the matching pipe file descriptors.
        let readFD = FileDescriptor(rawValue: fromServer.fileHandleForReading.fileDescriptor)
        let writeFD = FileDescriptor(rawValue: toServer.fileHandleForWriting.fileDescriptor)
        let transport = StdioTransport(input: readFD, output: writeFD)

        let client = Client(name: options.clientName, version: options.clientVersion)
        do {
            // connect performs the MCP initialize handshake internally.
            _ = try await client.connect(transport: transport)
        } catch {
            process.terminate()
            throw ServerError.spawnFailed("MCP handshake failed: \(error)")
        }

        return StackQLServer(
            client: client, binaryPath: path, mode: options.mode, process: process)
    }

    /// Disconnect the client and terminate the server process.
    public func stop() async {
        await client.disconnect()
        if process.isRunning {
            process.terminate()
        }
    }
}
