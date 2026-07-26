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
	Format   string `json:"format,omitempty" yaml:"format,omitempty" jsonschema:"text content render format: markdown (default) or json"`                          //nolint:lll // schema doc
	Source   string `json:"source,omitempty" yaml:"source,omitempty" jsonschema:"query library id when the SQL originated from a library entry; attribution only"` //nolint:lll // schema doc
}

// QueryLibrarySearchInput is the input shape for query_library_search.
type QueryLibrarySearchInput struct {
	Intent           string   `json:"intent" yaml:"intent" jsonschema:"natural language description of what the user wants"`
	Provider         string   `json:"provider,omitempty" yaml:"provider,omitempty" jsonschema:"filter by provider, eg aws"`
	Service          string   `json:"service,omitempty" yaml:"service,omitempty" jsonschema:"filter by service, eg ec2"`
	Tags             []string `json:"tags,omitempty" yaml:"tags,omitempty" jsonschema:"filter by tags"`
	IncludeMutations bool     `json:"include_mutations,omitempty" yaml:"include_mutations,omitempty"`
	Limit            int      `json:"limit,omitempty" yaml:"limit,omitempty" jsonschema:"max hits, default 5, max 20"`
	Format           string   `json:"format,omitempty" yaml:"format,omitempty" jsonschema:"text content render format: markdown (default) or json"` //nolint:lll // schema doc
}

// QueryLibraryGetInput is the input shape for query_library_get.
type QueryLibraryGetInput struct {
	ID     string         `json:"id" yaml:"id" jsonschema:"query id from search results, eg aws/ec2/regions-enabled"`
	Params map[string]any `json:"params,omitempty" yaml:"params,omitempty" jsonschema:"parameter values; when supplied the server validates and returns rendered SQL"` //nolint:lll // schema doc
	Format string         `json:"format,omitempty" yaml:"format,omitempty" jsonschema:"text content render format: markdown (default) or json"`                        //nolint:lll // schema doc
}

// QueryLibraryHitDTO is one ranked search hit; deliberately compact.
type QueryLibraryHitDTO struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Description    string   `json:"description,omitempty"`
	Providers      []string `json:"providers,omitempty"`
	Services       []string `json:"services,omitempty"`
	Mutation       bool     `json:"mutation"`
	RequiredParams []string `json:"required_params,omitempty"`
	Score          float64  `json:"score"`
}

// QueryLibrarySearchDTO is the result of query_library_search.
type QueryLibrarySearchDTO struct {
	Hits         []QueryLibraryHitDTO `json:"hits"`
	Miss         bool                 `json:"miss" jsonschema:"true when no hit cleared the score threshold"`
	DialectGuide string               `json:"dialect_guide,omitempty" jsonschema:"compact dialect guide returned on the miss path"`
	SourceTier   string               `json:"source_tier,omitempty" jsonschema:"primary, fallback or snapshot"`
	SourceURL    string               `json:"source_url,omitempty" jsonschema:"citation: URL or embedded path the catalogue was loaded from"`
	Stale        bool                 `json:"stale,omitempty" jsonschema:"true when served from cache or snapshot after fetch failure"`
}

// QueryLibraryParamDTO describes one declared template parameter.
type QueryLibraryParamDTO struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Default     string   `json:"default,omitempty"`
	Description string   `json:"description,omitempty"`
	Example     string   `json:"example,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Pattern     string   `json:"pattern,omitempty"`
}

// QueryLibraryCostDTO carries pre-execution cost hints: fan_out (none |
// region | project | account | subscription) lets an agent warn the user
// before running something that iterates a whole scope.
type QueryLibraryCostDTO struct {
	FanOut    string `json:"fan_out,omitempty"`
	Expensive bool   `json:"expensive,omitempty"`
	Notes     string `json:"notes,omitempty"`
}

// QueryLibraryOutputDTO describes one column the template returns.
type QueryLibraryOutputDTO struct {
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}

// QueryLibraryGetDTO is the result of query_library_get.  Without params it is
// the teaching surface (template with placeholders intact); with params the
// Rendered/SQL/Valid fields carry the outcome.
type QueryLibraryGetDTO struct {
	ID            string                  `json:"id"`
	Title         string                  `json:"title,omitempty"`
	Description   string                  `json:"description,omitempty"`
	Mutation      bool                    `json:"mutation"`
	Verb          string                  `json:"verb,omitempty" jsonschema:"select, mutation or lifecycle"`
	Cost          *QueryLibraryCostDTO    `json:"cost,omitempty"`
	Outputs       []QueryLibraryOutputDTO `json:"outputs,omitempty"`
	Auth          []string                `json:"auth,omitempty" jsonschema:"env vars the template's providers need resolved"`
	Related       []string                `json:"related,omitempty" jsonschema:"related library entry ids"`
	NextTool      string                  `json:"next_tool" jsonschema:"run_select_query, run_mutation_query or run_lifecycle_operation"` //nolint:lll // schema doc
	Template      string                  `json:"template,omitempty" jsonschema:"raw template with {{param}} placeholders"`
	Notes         string                  `json:"notes,omitempty"`
	DocURL        string                  `json:"doc_url,omitempty"`
	Params        []QueryLibraryParamDTO  `json:"params,omitempty"`
	Rendered      bool                    `json:"rendered"`
	Valid         bool                    `json:"valid"`
	SQL           string                  `json:"sql,omitempty" jsonschema:"rendered SQL, present when rendered and valid"`
	MissingParams []QueryLibraryParamDTO  `json:"missing_params,omitempty"`
	UnknownParams []string                `json:"unknown_params,omitempty"`
	Errors        []string                `json:"errors,omitempty"`
	SourceTier    string                  `json:"source_tier,omitempty"`
	SourceURL     string                  `json:"source_url,omitempty" jsonschema:"citation: URL or embedded path the document was loaded from"`
	Stale         bool                    `json:"stale,omitempty"`
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

// CredentialsReloadInput is the input shape for reload_credentials; Provider
// optionally scopes the status report (sourcing is always process-wide).
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
