package mcpbackend

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/stackql/any-sdk/pkg/dto"

	"github.com/stackql/stackql/internal/stackql/envfile"
	"github.com/stackql/stackql/internal/stackql/intrinsic"
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

// credentialFingerprint digests the credential material an auth context
// (successor chain included) resolves to right now; the digest is compared,
// never emitted, so a reload can report whether rotation took effect.
func credentialFingerprint(ac *dto.AuthCtx) string {
	h := sha256.New()
	for cur := ac; cur != nil; cur = cur.Successor {
		if b, err := cur.GetCredentialsBytes(); err == nil {
			h.Write(b)
		}
		if keyID, err := cur.GetKeyIDString(); err == nil {
			h.Write([]byte(keyID))
		}
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
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

// installedProviderNames enumerates registry-installed providers (the
// list_providers set less the intrinsic provider, document-first aliases and
// SQL data sources), sorted.
func (b *stackqlMCPService) installedProviderNames() ([]string, error) {
	supported, err := b.handlerCtx.GetSupportedProviders(false)
	if err != nil {
		return nil, err
	}
	rv := make([]string, 0, len(supported))
	for name := range supported {
		if name == intrinsic.ProviderName || strings.HasPrefix(name, intrinsic.UnstablePrefix) {
			continue
		}
		if _, isSQLDataSource := b.handlerCtx.GetSQLDataSource(name); isSQLDataSource {
			continue
		}
		rv = append(rv, name)
	}
	sort.Strings(rv)
	return rv, nil
}

// resolveAuthContext registers the provider (lazily, exactly as a query
// would) and returns its effective auth context: the --auth override when
// present, else the provider document default.
func (b *stackqlMCPService) resolveAuthContext(providerName string) (*dto.AuthCtx, error) {
	if _, err := b.handlerCtx.GetProvider(providerName); err != nil {
		return nil, err
	}
	return b.handlerCtx.GetAuthContext(providerName)
}

func (b *stackqlMCPService) providerCredentialRow(providerName, priorFingerprint string) mcp_dto.ProviderCredentialStatusDTO {
	ac, err := b.resolveAuthContext(providerName)
	if err != nil {
		return mcp_dto.ProviderCredentialStatusDTO{
			Provider:    providerName,
			SourcedFrom: credentialSourceNone,
			Status:      credentialStatusNotChecked,
			Detail:      err.Error(),
		}
	}
	rv := providerCredentialStatus(providerName, ac)
	rv.Changed = credentialFingerprint(ac) != priorFingerprint
	return rv
}

// ReloadCredentials implements the reload_credentials MCP tool as three
// ordered phases: re-source the env file, invalidate lazily registered auth
// contexts, then report resolution status for every installed provider
// against the now-current environment.  With no env file configured it
// degrades to a pure status probe.
func (b *stackqlMCPService) ReloadCredentials(
	_ context.Context,
	input mcp_dto.CredentialsReloadInput,
) (mcp_dto.CredentialsReloadDTO, error) {
	rv := mcp_dto.CredentialsReloadDTO{
		EnvFile:   b.envFile,
		Providers: []mcp_dto.ProviderCredentialStatusDTO{},
	}
	providerNames, namesErr := b.installedProviderNames()
	if namesErr != nil {
		return rv, fmt.Errorf("failed to enumerate installed providers: %w", namesErr)
	}
	if input.Provider != "" {
		idx := sort.SearchStrings(providerNames, input.Provider)
		if idx == len(providerNames) || providerNames[idx] != input.Provider {
			return rv, fmt.Errorf("provider '%s' is not installed", input.Provider)
		}
		providerNames = providerNames[idx : idx+1]
	}
	priorFingerprints := make(map[string]string, len(providerNames))
	for _, name := range providerNames {
		if ac, err := b.resolveAuthContext(name); err == nil {
			priorFingerprints[name] = credentialFingerprint(ac)
		}
	}
	sourcedVars, sourced, err := envfile.Source(b.envFile)
	if err != nil {
		return rv, fmt.Errorf("failed to source env file '%s': %w", b.envFile, err)
	}
	if b.envFile != "" && !sourced {
		return rv, fmt.Errorf("env file '%s' not found", b.envFile)
	}
	rv.EnvFileSourced = sourced
	rv.SourcedVars = sourcedVars
	b.handlerCtx.InvalidateAuthContexts(input.Provider)
	for _, name := range providerNames {
		rv.Providers = append(rv.Providers, b.providerCredentialRow(name, priorFingerprints[name]))
	}
	return rv, nil
}
