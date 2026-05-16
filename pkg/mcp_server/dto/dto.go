package dto

// HierarchyInput identifies a point in the provider/service/resource/method hierarchy.
type HierarchyInput struct {
	Provider string `json:"provider,omitempty" yaml:"provider,omitempty"`
	Service  string `json:"service,omitempty" yaml:"service,omitempty"`
	Resource string `json:"resource,omitempty" yaml:"resource,omitempty"`
	Method   string `json:"method,omitempty" yaml:"method,omitempty"`
	RowLimit int    `json:"row_limit,omitempty" yaml:"row_limit,omitempty"`
}

// ServerInfoOutput is the backend-facing server info payload. The JSON tag
// for ReadOnly stays `is_read_only` so existing robot tests keep passing.
type ServerInfoOutput struct {
	Version          string `json:"version,omitempty" jsonschema:"stackql semver"`
	Commit           string `json:"commit,omitempty" jsonschema:"git short commit SHA"`
	BuildDate        string `json:"build_date,omitempty" jsonschema:"build timestamp"`
	Platform         string `json:"platform,omitempty" jsonschema:"build platform"`
	Transport        string `json:"transport,omitempty" jsonschema:"MCP transport (http, stdio, reverse_proxy)"`
	SQLBackend       string `json:"sql_backend,omitempty" jsonschema:"backing SQL engine identifier"`
	ProviderRegistry string `json:"provider_registry,omitempty" jsonschema:"provider registry URL or path"`
	ReadOnly         bool   `json:"is_read_only" jsonschema:"is the server read-only"`
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
