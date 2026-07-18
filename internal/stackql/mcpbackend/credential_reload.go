package mcpbackend

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/stackql/any-sdk/pkg/dto"

	"github.com/stackql/stackql/internal/stackql/envfile"
	mcp_dto "github.com/stackql/stackql/pkg/mcp_server/dto"
)

// Credential (re)sourcing for the MCP server (issue #688).  A process env
// block is fixed at spawn, so a stdio child launched without credential vars
// can never see them; re-sourcing the --env.file dotenv file into this
// process's environment bridges that gap.  Values are never echoed, logged
// or audited - only names and per-provider statuses.

const (
	credentialStatusOK         = "ok"
	credentialStatusUnresolved = "unresolved"
	credentialStatusNotChecked = "not_checked"

	credentialSourceInline = "inline"
	credentialSourceNone   = "none"
)

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
// a meaningful dry run for the auth type; other types are reported unchecked.
func isCredentialCheckSupported(authType string) bool {
	switch strings.ToLower(authType) {
	case dto.AuthAPIKeyStr, dto.AuthBearerStr, dto.AuthBasicStr,
		dto.AuthServiceAccountStr, dto.AuthAWSSigningv4Str, dto.AuthCustomStr:
		return true
	default:
		return false
	}
}

// providerCredentialStatus dry-runs credential resolution for one provider;
// resolved bytes are discarded, only the outcome is reported.
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

// ReloadCredentials implements the reload_credentials MCP tool: (re)source
// the env file, then report per-provider credential resolution status.  With
// no env file configured it degrades to a pure status probe.
func (b *stackqlMCPService) ReloadCredentials(
	_ context.Context,
	input mcp_dto.CredentialsReloadInput,
) (mcp_dto.CredentialsReloadDTO, error) {
	rv := mcp_dto.CredentialsReloadDTO{EnvFile: b.envFile}
	sourcedVars, sourced, err := envfile.Source(b.envFile)
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
