// Package embed runs an embedded StackQL MCP server from a binary carried
// inside the calling program.
//
// The calling program embeds the platform's signed stackql binary (see
// cmd/stackql-mcp-fetch, which downloads it from the release bundle,
// verifies the published sha256 pin, and generates the go:embed glue).
// StartServer extracts the binary to the shared on-disk cache, spawns it as
// an MCP stdio server with the canonical launch arguments, and returns a
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

// moduleVersion is reported as the MCP client version.
const moduleVersion = "0.1.0"

// Options configures StartServer.
type Options struct {
	// Binary is the embedded server binary. Required.
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

	// BinaryPath is where the server binary was extracted.
	BinaryPath string

	// Mode is the safety mode the server was started with.
	Mode Mode
}

// Close shuts down the MCP session and terminates the server process.
func (c *Client) Close() error {
	return c.Session.Close()
}

// CommandLine resolves the exact command this package would run for the
// given options, extracting the binary to the cache first. It exists so
// external conformance harnesses (the packaging repo's smoke-test.py --cmd
// mode) can exercise the launcher without going through StartServer.
func CommandLine(opts Options) (path string, args []string, err error) {
	path, err = EnsureExtracted(opts.CacheDir, opts.Binary)
	if err != nil {
		return "", nil, err
	}
	args, err = BuildArgs(opts.Mode, opts.AppRoot, opts.Auth)
	if err != nil {
		return "", nil, err
	}
	return path, append(args, opts.ExtraArgs...), nil
}

// StartServer extracts the embedded binary, spawns it as an MCP stdio
// server, performs the MCP handshake, and returns a connected Client.
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
		impl = &mcp.Implementation{Name: "stackql-mcp-go", Version: moduleVersion}
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
