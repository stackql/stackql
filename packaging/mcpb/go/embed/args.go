package embed

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Mode is the stackql MCP server safety mode. It bounds which tools the
// server exposes and what SQL they may run.
type Mode string

const (
	// ModeReadOnly exposes metadata and SELECT tools only. The default.
	ModeReadOnly Mode = "read_only"
	// ModeSafe additionally allows non-destructive mutations.
	ModeSafe Mode = "safe"
	// ModeDeleteSafe additionally allows deletes.
	ModeDeleteSafe Mode = "delete_safe"
	// ModeFullAccess removes mode restrictions.
	ModeFullAccess Mode = "full_access"
)

func (m Mode) validate() error {
	switch m {
	case ModeReadOnly, ModeSafe, ModeDeleteSafe, ModeFullAccess:
		return nil
	}
	return fmt.Errorf("stackql mcp: unknown mode %q", string(m))
}

// DefaultAppRoot returns ~/.stackql, the conventional stackql application
// root shared with every other stackql distribution on the machine.
func DefaultAppRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("stackql mcp: resolving home dir: %w", err)
	}
	return filepath.Join(home, ".stackql"), nil
}

// mcpConfig mirrors the --mcp.config document accepted by stackql. Audit is
// disabled because an embedded server has no console session to audit to.
type mcpConfig struct {
	Server struct {
		Mode  string `json:"mode"`
		Audit struct {
			Disabled bool `json:"disabled"`
		} `json:"audit"`
	} `json:"server"`
}

// BuildArgs returns the canonical stackql MCP launch arguments:
//
//	mcp --mcp.server.type=stdio --approot <approot>
//	    --mcp.config {"server":{"mode":"<mode>","audit":{"disabled":true}}}
//	    [--auth=<json>]
//
// Every path is absolute so the server is independent of the working
// directory: MCP hosts may launch with cwd "/", which is read-only on
// macOS. mode defaults to ModeReadOnly; appRoot defaults to ~/.stackql.
// auth, if non-nil, is marshalled into a --auth flag (for example the
// github null_auth fixture used by the conformance tests).
func BuildArgs(mode Mode, appRoot string, auth map[string]any) ([]string, error) {
	if mode == "" {
		mode = ModeReadOnly
	}
	if err := mode.validate(); err != nil {
		return nil, err
	}
	if appRoot == "" {
		var err error
		appRoot, err = DefaultAppRoot()
		if err != nil {
			return nil, err
		}
	}
	if !filepath.IsAbs(appRoot) {
		return nil, fmt.Errorf("stackql mcp: approot must be absolute, got %q", appRoot)
	}
	var cfg mcpConfig
	cfg.Server.Mode = string(mode)
	cfg.Server.Audit.Disabled = true
	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("stackql mcp: marshalling mcp config: %w", err)
	}
	args := []string{
		"mcp",
		"--mcp.server.type=stdio",
		"--approot", appRoot,
		"--mcp.config", string(cfgJSON),
	}
	if auth != nil {
		authJSON, err := json.Marshal(auth)
		if err != nil {
			return nil, fmt.Errorf("stackql mcp: marshalling auth: %w", err)
		}
		args = append(args, "--auth="+string(authJSON))
	}
	return args, nil
}
