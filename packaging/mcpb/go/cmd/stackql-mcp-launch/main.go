// Command stackql-mcp-launch resolves the StackQL MCP server binary (sidecar
// path: STACKQL_MCP_BIN / STACKQL_MCP_BUNDLE, the shared cache, or a
// pin-verified download) and runs it with the canonical launch arguments
// and inherited stdio. Extra argv is forwarded to the server verbatim.
//
// This is the command packaging/mcpb/scripts/smoke-test.py --cmd drives:
//
//	python scripts/smoke-test.py --cmd "go run ./cmd/stackql-mcp-launch"
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	stackqlmcp "github.com/stackql/stackql-mcp-go/embed"
)

func main() {
	path, args, err := stackqlmcp.CommandLine(stackqlmcp.Options{
		Mode:      stackqlmcp.ModeReadOnly,
		ExtraArgs: os.Args[1:],
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "stackql-mcp-launch:", err)
		os.Exit(1)
	}
	cmd := exec.Command(path, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, "stackql-mcp-launch:", err)
		os.Exit(1)
	}
}
