package dto

// HierarchyInput identifies a point in the provider/service/resource/method hierarchy.
type HierarchyInput struct {
	Provider string `json:"provider,omitempty" yaml:"provider,omitempty"`
	Service  string `json:"service,omitempty" yaml:"service,omitempty"`
	Resource string `json:"resource,omitempty" yaml:"resource,omitempty"`
	Method   string `json:"method,omitempty" yaml:"method,omitempty"`
	RowLimit int    `json:"row_limit,omitempty" yaml:"row_limit,omitempty"`
	Format   string `json:"format,omitempty" yaml:"format,omitempty" jsonschema:"text content render format: markdown (default) or json"` //nolint:lll // schema doc
}

// ServerInfoOutput is the backend-facing server info payload.
// ReadOnly (JSON tag `is_read_only`) is kept for back-compat with PR1
// consumers; new consumers should prefer the Mode field.
type ServerInfoOutput struct {
	Version          string `json:"version,omitempty" jsonschema:"stackql semver"`
	Commit           string `json:"commit,omitempty" jsonschema:"git short commit SHA"`
	BuildDate        string `json:"build_date,omitempty" jsonschema:"build timestamp"`
	Platform         string `json:"platform,omitempty" jsonschema:"build platform"`
	Transport        string `json:"transport,omitempty" jsonschema:"MCP transport (http, stdio, reverse_proxy)"`
	SQLBackend       string `json:"sql_backend,omitempty" jsonschema:"backing SQL engine identifier"`
	ProviderRegistry string `json:"provider_registry,omitempty" jsonschema:"provider registry URL or path"`
	Mode             string `json:"mode,omitempty" jsonschema:"server mode (read_only, safe, delete_safe, full_access)"`
	ReadOnly         bool   `json:"is_read_only" jsonschema:"true when mode is read_only (back-compat with PR1)"`
}

// ServerInfoDTO is the client-facing server info payload, mirrors ServerInfoOutput.
type ServerInfoDTO struct {
	Version          string `json:"version,omitempty"`
	Commit           string `json:"commit,omitempty"`
	BuildDate        string `json:"build_date,omitempty"`
	Platform         string `json:"platform,omitempty"`
	Transport        string `json:"transport,omitempty"`
	SQLBackend       string `json:"sql_backend,omitempty"`
	ProviderRegistry string `json:"provider_registry,omitempty"`
	Mode             string `json:"mode,omitempty"`
	ReadOnly         bool   `json:"is_read_only"`
}

// QueryJSONInput is the input shape for SELECT / mutation / lifecycle tools.
type QueryJSONInput struct {
	SQL      string `json:"sql" yaml:"sql"`
	RowLimit int    `json:"row_limit,omitempty" yaml:"row_limit,omitempty"`
	Format   string `json:"format,omitempty" yaml:"format,omitempty" jsonschema:"text content render format: markdown (default) or json"` //nolint:lll // schema doc
}

// RegistryInput is the shared input shape for list_registry and pull_provider.
// list_registry treats Provider as optional (when empty, lists all available
// providers); pull_provider requires Provider and treats Version as optional
// (when empty, the latest published version is pulled).
type RegistryInput struct {
	Provider string `json:"provider,omitempty" yaml:"provider,omitempty"`
	Version  string `json:"version,omitempty" yaml:"version,omitempty"`
	Format   string `json:"format,omitempty" yaml:"format,omitempty" jsonschema:"text content render format: markdown (default) or json"` //nolint:lll // schema doc
}

// CredentialsReloadInput is the input shape for reload_credentials.
// Provider optionally scopes the status report to one provider; the env file
// sourcing itself is always process-wide.
type CredentialsReloadInput struct {
	Provider string `json:"provider,omitempty" yaml:"provider,omitempty"`
	Format   string `json:"format,omitempty" yaml:"format,omitempty" jsonschema:"text content render format: markdown (default) or json"` //nolint:lll // schema doc
}

// ProviderCredentialStatusDTO reports one provider's credential resolution
// outcome.  Variable names and file paths only; never secret values.
type ProviderCredentialStatusDTO struct {
	Provider    string `json:"provider"`
	AuthType    string `json:"auth_type,omitempty"`
	SourcedFrom string `json:"sourced_from,omitempty" jsonschema:"where credentials are read from, eg env:VAR_NAME or file:/path"`
	Status      string `json:"status" jsonschema:"ok, unresolved or not_checked"`
	Detail      string `json:"detail,omitempty" jsonschema:"resolution error detail when status is unresolved"`
}

// CredentialsReloadDTO is the result of reload_credentials (issue #688).
type CredentialsReloadDTO struct {
	EnvFile        string                        `json:"env_file,omitempty" jsonschema:"configured dotenv file path, empty when none configured"`
	EnvFileSourced bool                          `json:"env_file_sourced" jsonschema:"true when the env file was found and sourced on this call"`
	SourcedVars    []string                      `json:"sourced_vars,omitempty" jsonschema:"names of environment variables set from the env file (values are never returned)"` //nolint:lll // schema doc
	Providers      []ProviderCredentialStatusDTO `json:"providers"`
}

// QueryResultDTO is the typed structured payload returned alongside the rendered text.
type QueryResultDTO struct {
	Rows []map[string]any `json:"rows"`
}

// ValidationResultDTO is the result of validate_select_query.
type ValidationResultDTO struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}
