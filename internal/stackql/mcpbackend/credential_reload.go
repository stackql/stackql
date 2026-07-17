package mcpbackend

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/stackql/any-sdk/pkg/dto"

	mcp_dto "github.com/stackql/stackql/pkg/mcp_server/dto"
)

// Credential (re)sourcing for the MCP server (issue #688).
//
// Credentials are resolved lazily by any-sdk on every upstream HTTP call
// (os.Getenv / os.ReadFile at auth time), so nothing here needs to invalidate
// cached secrets: there are none.  The problem this solves is upstream of
// resolution: a process environment block is fixed at spawn, so an MCP server
// child process (eg spawned by Claude Desktop) that never received the
// credential variables can never see them, no matter how often they are
// re-read.  The fix is to (re)source variables from a mutable store - a
// dotenv-style file nominated in mcp.config as server.env_file - and inject
// them into this process's own environment, where the existing lazy
// resolution picks them up unchanged.
//
// Semantics are identical on every platform:
//   - Only keys present in the file with non-empty values are set.
//   - Existing process env vars are overwritten (the file is the source of
//     truth on reload); nothing is ever unset.
//   - Values are never echoed back to the client, logged, or audited - only
//     variable names and per-provider resolution statuses are reported.

const (
	credentialStatusOK         = "ok"
	credentialStatusUnresolved = "unresolved"
	credentialStatusNotChecked = "not_checked"

	credentialSourceInline = "inline"
	credentialSourceNone   = "none"
)

// parseEnvFile reads a dotenv-style file: one KEY=VALUE per line, `#` comments
// and blank lines ignored, optional single or double quotes around the value,
// optional `export ` prefix, CRLF tolerated.  Keys with empty values are
// dropped (a reload can never blank out a previously working credential).
func parseEnvFile(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	rv := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, val, found := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !found || key == "" {
			continue
		}
		val = strings.TrimSpace(val)
		if len(val) >= 2 { //nolint:mnd // shortest quoted form is ""
			if (val[0] == '"' && val[len(val)-1] == '"') ||
				(val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		if val == "" {
			continue
		}
		rv[key] = val
	}
	return rv, nil
}

// sourceEnvFile injects the file's key/value pairs into the process
// environment and returns the sorted key names that were set.  A missing file
// is not an error: it means "nothing to source yet" and callers report it.
func sourceEnvFile(path string) ([]string, bool, error) {
	if path == "" {
		return nil, false, nil
	}
	vars, err := parseEnvFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	keys := make([]string, 0, len(vars))
	for k, v := range vars {
		if setErr := os.Setenv(k, v); setErr != nil {
			return nil, false, setErr
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, true, nil
}

// credentialSource describes where a provider's credentials come from,
// mirroring the precedence of any-sdk's AuthCtx.GetCredentialsBytes().  Names
// and paths only; never values.
func credentialSource(ac *dto.AuthCtx) string {
	switch {
	case ac.KeyEnvVar != "":
		return "env:" + ac.KeyEnvVar
	case ac.KeyFilePathEnvVar != "":
		return "env:" + ac.KeyFilePathEnvVar
	case ac.KeyFilePath != "":
		return "file:" + ac.KeyFilePath
	case ac.EnvVarUsername != "" && ac.EnvVarPassword != "":
		return fmt.Sprintf("env:%s,env:%s", ac.EnvVarUsername, ac.EnvVarPassword)
	case ac.EnvVarAPIKeyStr != "" && ac.EnvVarAPISecretStr != "":
		return fmt.Sprintf("env:%s,env:%s", ac.EnvVarAPIKeyStr, ac.EnvVarAPISecretStr)
	case ac.Username != "" || ac.APIKeyStr != "":
		return credentialSourceInline
	default:
		return credentialSourceNone
	}
}

// isCredentialCheckSupported reports whether AuthCtx.GetCredentialsBytes() is
// a meaningful dry run for the auth type.  Types that authenticate outside
// the credential-bytes path (interactive gcloud, azure CLI, oauth2 client
// credentials, assume-role chains, OCI signing, null) are reported but not
// checked.
func isCredentialCheckSupported(authType string) bool {
	switch strings.ToLower(authType) {
	case dto.AuthAPIKeyStr, dto.AuthBearerStr, dto.AuthBasicStr,
		dto.AuthServiceAccountStr, dto.AuthAWSSigningv4Str, dto.AuthCustomStr:
		return true
	default:
		return false
	}
}

// providerCredentialStatus dry-runs credential resolution for one provider.
// The resolved bytes are discarded immediately; only the outcome is reported.
func providerCredentialStatus(providerName string, ac *dto.AuthCtx) mcp_dto.ProviderCredentialStatusDTO {
	rv := mcp_dto.ProviderCredentialStatusDTO{
		Provider:    providerName,
		AuthType:    ac.Type,
		SourcedFrom: credentialSource(ac),
		Status:      credentialStatusNotChecked,
	}
	if !isCredentialCheckSupported(ac.Type) {
		return rv
	}
	if _, err := ac.GetCredentialsBytes(); err != nil {
		rv.Status = credentialStatusUnresolved
		rv.Detail = err.Error()
		return rv
	}
	rv.Status = credentialStatusOK
	return rv
}

// ReloadCredentials implements the reload_credentials MCP tool: (re)source the
// configured env file into the process environment, then report per-provider
// credential resolution status.  With no env file configured the sourcing
// step is skipped and the call degrades to a pure status probe.
func (b *stackqlMCPService) ReloadCredentials(
	_ context.Context,
	input mcp_dto.CredentialsReloadInput,
) (mcp_dto.CredentialsReloadDTO, error) {
	rv := mcp_dto.CredentialsReloadDTO{EnvFile: b.envFile}
	sourcedVars, sourced, err := sourceEnvFile(b.envFile)
	if err != nil {
		return rv, fmt.Errorf("failed to source env file '%s': %w", b.envFile, err)
	}
	rv.EnvFileSourced = sourced
	rv.SourcedVars = sourcedVars
	authContexts := b.handlerCtx.GetAuthContexts()
	if input.Provider != "" {
		ac, ok := authContexts[input.Provider]
		if !ok {
			return rv, fmt.Errorf("cannot find AUTH context for provider = '%s'", input.Provider)
		}
		rv.Providers = append(rv.Providers, providerCredentialStatus(input.Provider, ac))
		return rv, nil
	}
	providerNames := make([]string, 0, len(authContexts))
	for name := range authContexts {
		providerNames = append(providerNames, name)
	}
	sort.Strings(providerNames)
	for _, name := range providerNames {
		rv.Providers = append(rv.Providers, providerCredentialStatus(name, authContexts[name]))
	}
	return rv, nil
}
