package mcp_server //nolint:revive // fine for now

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCP extension identifiers, as advertised under capabilities.extensions and
// declared by clients under clientCapabilities.extensions.
const (
	skillsExtensionID = "io.modelcontextprotocol/skills"
	tasksExtensionID  = "io.modelcontextprotocol/tasks"
	filesExtensionID  = "io.modelcontextprotocol/files"
)

// buildServerCapabilities is the single place ServerOptions.Capabilities is
// built.  A non-nil Capabilities replaces the SDK's default {"logging":{}},
// so logging is always included; the SDK still infers tools, prompts and
// resources from what is registered.  Extensions appear only when enabled, so
// with every extension off the advertised capabilities equal the SDK default.
func buildServerCapabilities(cfg *Config) *mcp.ServerCapabilities {
	caps := &mcp.ServerCapabilities{Logging: &mcp.LoggingCapabilities{}}
	extensions := cfg.Server.Extensions
	if extensions.Skills.Enabled {
		// No directoryRead: skill files are listed in each skill's manifest.
		caps.AddExtension(skillsExtensionID, nil)
	}
	if extensions.Tasks.Enabled {
		caps.AddExtension(tasksExtensionID, nil)
	}
	if extensions.Files.Enabled {
		// No upload: without it a compliant client sends files inline only.
		caps.AddExtension(filesExtensionID, map[string]any{
			"maxInlineBytes": extensions.Files.GetMaxInlineBytes(),
		})
	}
	return caps
}

// clientDeclaresExtension reports whether the client capabilities of one
// request declare the extension.  Callers pass req.ClientCapabilities(), which
// reads the 2026-07-28 per-request _meta or, for older revisions, the
// initialize params.  The server knows only the initial revision of each
// extension, so a settings object carrying a revision field is treated as not
// declared (the files extension rule, applied to every extension).  A null
// settings value counts as an empty object, as the SDK normalises it.
func clientDeclaresExtension(caps *mcp.ClientCapabilities, id string) bool {
	if caps == nil || caps.Extensions == nil {
		return false
	}
	raw, declared := caps.Extensions[id]
	if !declared {
		return false
	}
	if raw == nil {
		return true
	}
	settings, isObject := raw.(map[string]any)
	if !isObject {
		return false
	}
	_, hasRevision := settings["revision"]
	return !hasRevision
}
