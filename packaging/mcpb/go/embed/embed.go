// Package embed runs an embedded StackQL MCP server.
//
// Two acquisition paths behind one API:
//
//   - vendored: the calling program embeds the platform's signed stackql
//     binary (see cmd/stackql-mcp-fetch, which downloads it from the release
//     bundle, verifies the sha256 pin from platforms.json, and generates the
//     go:embed glue) and passes it as Options.Binary
//   - sidecar (Options.Binary unset): the binary is resolved at first run
//     from the shared cache ~/.stackql/mcp-server-bin/<version>/<platform>/,
//     downloading and pin-verifying the release bundle when absent
//
// Either way STACKQL_MCP_BIN (run this binary) and STACKQL_MCP_BUNDLE
// (extract this local .mcpb) take precedence. StartServer spawns the binary
// as an MCP stdio server with the canonical launch arguments and returns a
// connected client backed by github.com/modelcontextprotocol/go-sdk.
//
// Because the package is named embed, import it with an alias when the
// file also blank-imports the standard library embed package:
//
//	import (
//		_ "embed"
//
//		stackqlmcp "github.com/stackql/stackql-mcp-go/embed"
//	)
package embed

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Options configures StartServer.
type Options struct {
	// Binary is an embedded server binary (vendored path). When zero, the
	// sidecar path resolves one: env overrides, the shared cache, then a
	// pin-verified download of the release bundle.
	Binary Binary

	// Mode is the server safety mode. Defaults to ModeReadOnly; anything
	// more permissive is an explicit caller decision.
	Mode Mode

	// AppRoot is the stackql application root (provider registry cache,
	// auth state). Defaults to ~/.stackql. Must be absolute if set.
	AppRoot string

	// CacheDir overrides the binary extraction cache root. Defaults to
	// ~/.stackql/mcp-server-bin (shared with the npm/pypi wrappers).
	CacheDir string

	// Auth, if non-nil, is marshalled into the --auth flag, overriding
	// provider auth from the environment. Example:
	// map[string]any{"github": map[string]any{"type": "null_auth"}}.
	Auth map[string]any

	// Stderr receives the server's diagnostics. Defaults to os.Stderr.
	// stdout belongs to the MCP protocol and is not configurable.
	Stderr io.Writer

	// ExtraArgs are appended verbatim after the canonical arguments.
	ExtraArgs []string

	// ClientInfo overrides the MCP client identity sent in initialize.
	ClientInfo *mcp.Implementation
}

// Client is a running embedded server and the MCP session connected to it.
type Client struct {
	// Session is the initialized MCP session. Use it directly for
	// ListTools, CallTool, and the rest of the protocol surface.
	Session *mcp.ClientSession

	// BinaryPath is the server binary that was launched.
	BinaryPath string

	// Mode is the safety mode the server was started with.
	Mode Mode
}

// Close shuts down the MCP session and terminates the server process.
func (c *Client) Close() error {
	return c.Session.Close()
}

// CommandLine resolves the exact command this package would run for the
// given options, acquiring the binary first. It exists so external
// conformance harnesses (packaging/mcpb/scripts/smoke-test.py --cmd via
// cmd/stackql-mcp-launch) can exercise the launcher without going through
// StartServer.
func CommandLine(opts Options) (path string, args []string, err error) {
	if p, ok, oerr := envOverride(opts.CacheDir); ok || oerr != nil {
		path, err = p, oerr
	} else if len(opts.Binary.Data) > 0 {
		path, err = EnsureExtracted(opts.CacheDir, opts.Binary)
	} else {
		path, err = ResolveBinary(opts.CacheDir)
	}
	if err != nil {
		return "", nil, err
	}
	args, err = BuildArgs(opts.Mode, opts.AppRoot, opts.Auth)
	if err != nil {
		return "", nil, err
	}
	return path, append(args, opts.ExtraArgs...), nil
}

// StartServer acquires the binary, spawns it as an MCP stdio server,
// performs the MCP handshake, and returns a connected Client.
// ctx bounds the startup (extraction and handshake) only; the returned
// Client outlives it and runs until Close.
func StartServer(ctx context.Context, opts Options) (*Client, error) {
	path, args, err := CommandLine(opts)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(path, args...)
	if opts.Stderr != nil {
		cmd.Stderr = opts.Stderr
	} else {
		cmd.Stderr = os.Stderr
	}

	impl := opts.ClientInfo
	if impl == nil {
		impl = &mcp.Implementation{Name: "stackql-mcp-go", Version: DefaultVersion}
	}
	client := mcp.NewClient(impl, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		return nil, fmt.Errorf("stackql mcp: connecting to embedded server: %w", err)
	}
	mode := opts.Mode
	if mode == "" {
		mode = ModeReadOnly
	}
	return &Client{Session: session, BinaryPath: path, Mode: mode}, nil
}
