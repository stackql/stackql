package dto

// HierarchyInput identifies a point in the provider/service/resource/method hierarchy.
type HierarchyInput struct {
	Provider string `json:"provider,omitempty" yaml:"provider,omitempty"`
	Service  string `json:"service,omitempty" yaml:"service,omitempty"`
	Resource string `json:"resource,omitempty" yaml:"resource,omitempty"`
	Method   string `json:"method,omitempty" yaml:"method,omitempty"`
	RowLimit int    `json:"row_limit,omitempty" yaml:"row_limit,omitempty"`
}

// RegistryInput identifies a provider (and optionally a version) within the
// provider registry.  Used by both list_registry (provider optional) and
// pull_provider (provider required).
type RegistryInput struct {
	Provider string `json:"provider,omitempty" yaml:"provider,omitempty"`
	Version  string `json:"version,omitempty" yaml:"version,omitempty"`
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
