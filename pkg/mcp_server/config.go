package mcp_server //nolint:revive,stylecheck,mnd // fine for now

import (
	"encoding/json"
	"fmt"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/stackql/stackql/pkg/mcp_server/audit"
	"github.com/stackql/stackql/pkg/mcp_server/policy"
)

// Config represents the complete configuration for the MCP server.
type Config struct {
	// Server contains server-specific configuration.
	Server ServerConfig `json:"server" yaml:"server"`

	// Backend contains backend-specific configuration.
	Backend BackendConfig `json:"backend" yaml:"backend"`

	// EnabledTools restricts which MCP tools the server publishes.
	// When nil or empty, every built-in tool is published (default behavior).
	// When populated, only the named tools are registered.
	EnabledTools []string `json:"enabled_tools,omitempty" yaml:"enabled_tools,omitempty"`

	// EnabledPrompts restricts which MCP prompts the server publishes.
	// Same semantics as EnabledTools: nil or empty means all registered prompts are published.
	EnabledPrompts []string `json:"enabled_prompts,omitempty" yaml:"enabled_prompts,omitempty"`
}

// IsToolEnabled reports whether the named tool should be published.
// Empty/nil EnabledTools means all tools are enabled.
func (c *Config) IsToolEnabled(name string) bool {
	if len(c.EnabledTools) == 0 {
		return true
	}
	for _, n := range c.EnabledTools {
		if n == name {
			return true
		}
	}
	return false
}

// IsPromptEnabled reports whether the named prompt should be published.
// Empty/nil EnabledPrompts means all prompts are enabled.
func (c *Config) IsPromptEnabled(name string) bool {
	if len(c.EnabledPrompts) == 0 {
		return true
	}
	for _, n := range c.EnabledPrompts {
		if n == name {
			return true
		}
	}
	return false
}

func (c *Config) GetServerTransport() string {
	if c.Server.Transport == "" {
		return DefaultConfig().Server.Transport
	}
	return c.Server.Transport
}

func (c *Config) GetServerAddress() string {
	if c.Server.Address == "" {
		return DefaultConfig().Server.Address
	}
	return c.Server.Address
}

func (c *Config) IsTcpBackend() bool {
	return c.Backend.Type == "tcp"
}

func (c *Config) GetBackendConnectionString() string {
	if c.Backend.ConnectionString == "" {
		return DefaultConfig().Backend.ConnectionString
	}
	return c.Backend.ConnectionString
}

// GetMode returns the server's effective mode.  Empty string is mapped to
// the safe default.
func (c *Config) GetMode() string {
	if c == nil || c.Server.Mode == "" {
		return policy.ModeSafe
	}
	return c.Server.Mode
}

// IsAuditEnabled reports whether audit logging should run.  Audit is on by
// default unless explicitly disabled.
func (c *Config) IsAuditEnabled() bool {
	if c == nil || c.Server.Audit.Disabled {
		return false
	}
	return true
}

// ServerConfig contains configuration for the MCP server itself.
type ServerConfig struct {
	// Name is the server name advertised to clients.
	Name string `json:"name" yaml:"name"`

	// Transport specifies the transport configuration for the server.
	Transport string `json:"transport" yaml:"transport"`

	// Address is the server Address advertised to clients.
	Address string `json:"address" yaml:"address"`

	// Scheme is the protocol scheme used by the server.
	Scheme string `json:"scheme" yaml:"scheme"`

	// Version is the server version advertised to clients.
	Version string `json:"version" yaml:"version"`

	TLSCertFile string `json:"tls_cert_file,omitempty" yaml:"tls_cert_file,omitempty"`
	TLSKeyFile  string `json:"tls_key_file,omitempty" yaml:"tls_key_file,omitempty"`

	TransportCfg map[string]any `json:"transport_cfg,omitempty" yaml:"transport_cfg,omitempty"`

	// Description is a human-readable description of the server.
	Description string `json:"description" yaml:"description"`

	// MaxConcurrentRequests limits the number of concurrent client requests.
	MaxConcurrentRequests int `json:"max_concurrent_requests" yaml:"max_concurrent_requests"`

	// RequestTimeout specifies the timeout for individual requests.
	RequestTimeout Duration `json:"request_timeout" yaml:"request_timeout"`

	// Mode controls the safety contract for query / mutation / lifecycle tools.
	// Legal values: "read_only", "safe", "delete_safe", "full_access".
	// Empty string is treated as "safe".
	//
	// For back-compat with PR1, the JSON/YAML key `read_only: true` is also
	// accepted and is equivalent to Mode = "read_only".  When both `mode`
	// and `read_only` are set, `mode` wins.
	Mode string `json:"mode,omitempty" yaml:"mode,omitempty"`

	// Audit configures the audit subsystem.  Audit is enabled by default
	// (Disabled is false) and writes to a file sink.
	Audit AuditConfig `json:"audit,omitempty" yaml:"audit,omitempty"`
}

// serverConfigWire mirrors ServerConfig with the legacy `read_only` flag
// included as a transient field.  Used only for unmarshalling.
type serverConfigWire struct {
	Name                  string         `json:"name" yaml:"name"`
	Transport             string         `json:"transport" yaml:"transport"`
	Address               string         `json:"address" yaml:"address"`
	Scheme                string         `json:"scheme" yaml:"scheme"`
	Version               string         `json:"version" yaml:"version"`
	TLSCertFile           string         `json:"tls_cert_file,omitempty" yaml:"tls_cert_file,omitempty"`
	TLSKeyFile            string         `json:"tls_key_file,omitempty" yaml:"tls_key_file,omitempty"`
	TransportCfg          map[string]any `json:"transport_cfg,omitempty" yaml:"transport_cfg,omitempty"`
	Description           string         `json:"description" yaml:"description"`
	MaxConcurrentRequests int            `json:"max_concurrent_requests" yaml:"max_concurrent_requests"`
	RequestTimeout        Duration       `json:"request_timeout" yaml:"request_timeout"`
	Mode                  string         `json:"mode,omitempty" yaml:"mode,omitempty"`
	Audit                 AuditConfig    `json:"audit,omitempty" yaml:"audit,omitempty"`
	// LegacyReadOnly preserves the PR1 `read_only: true` wire form.
	LegacyReadOnly *bool `json:"read_only,omitempty" yaml:"read_only,omitempty"`
}

func (s *ServerConfig) fromWire(w serverConfigWire) {
	s.Name = w.Name
	s.Transport = w.Transport
	s.Address = w.Address
	s.Scheme = w.Scheme
	s.Version = w.Version
	s.TLSCertFile = w.TLSCertFile
	s.TLSKeyFile = w.TLSKeyFile
	s.TransportCfg = w.TransportCfg
	s.Description = w.Description
	s.MaxConcurrentRequests = w.MaxConcurrentRequests
	s.RequestTimeout = w.RequestTimeout
	s.Mode = w.Mode
	s.Audit = w.Audit
	// Legacy: `read_only: true` with no `mode` -> Mode = "read_only".
	// `mode` always wins.
	if s.Mode == "" && w.LegacyReadOnly != nil && *w.LegacyReadOnly {
		s.Mode = policy.ModeReadOnly
	}
}

// UnmarshalJSON honours the legacy `read_only: true` shim.
func (s *ServerConfig) UnmarshalJSON(data []byte) error {
	var w serverConfigWire
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	s.fromWire(w)
	return nil
}

// UnmarshalYAML honours the legacy `read_only: true` shim.
func (s *ServerConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var w serverConfigWire
	if err := unmarshal(&w); err != nil {
		return err
	}
	s.fromWire(w)
	return nil
}

// AuditConfig configures the audit subsystem.
type AuditConfig struct {
	// Disabled turns the audit subsystem off entirely.  Default false
	// (audit is on by default).
	Disabled bool `json:"disabled,omitempty" yaml:"disabled,omitempty"`

	// FailureMode controls what happens when the sink returns an error.
	// Legal values: "strict" (default), "strict_mutations", "best_effort".
	FailureMode string `json:"failure_mode,omitempty" yaml:"failure_mode,omitempty"`

	// Sink selects the destination kind.  Currently only "file" is
	// implemented; other values are reserved.  Empty defaults to "file".
	Sink string `json:"sink,omitempty" yaml:"sink,omitempty"`

	// File holds file-sink-specific options.  Only consulted when Sink is
	// "file" (the default).
	File audit.FileConfig `json:"file,omitempty" yaml:"file,omitempty"`
}

// GetFailureMode returns the effective failure-mode string with the default
// substituted for empty input.
func (a AuditConfig) GetFailureMode() string {
	if a.FailureMode == "" {
		return audit.FailureModeStrict
	}
	return a.FailureMode
}

// BackendConfig contains configuration for the backend connection.
type BackendConfig struct {
	// Type specifies the backend type ("tcp", "memory").
	Type string `json:"type" yaml:"type"`

	// AppName is an optional application name describing the backend.
	// In the first instance, this is stackql.
	// **Possible** future use case for the backing db (e.g., "postgres", "mysql", etc).
	AppName string `json:"app_name" yaml:"app_name"`

	// ConnectionString contains the connection details for the backend.
	// Format depends on the backend type.
	ConnectionString string `json:"dsn" yaml:"dsn"`

	// MaxConnections limits the number of backend connections.
	MaxConnections int `json:"max_connections" yaml:"max_connections"`

	// ConnectionTimeout specifies the timeout for backend connections.
	ConnectionTimeout Duration `json:"connection_timeout" yaml:"connection_timeout"`

	// QueryTimeout specifies the timeout for individual queries.
	QueryTimeout Duration `json:"query_timeout" yaml:"query_timeout"`
}

// Duration is a wrapper around time.Duration that can be marshaled to/from JSON and YAML.
type Duration time.Duration

// MarshalJSON implements json.Marshaler.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON implements json.Unmarshaler.
func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	duration, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(duration)
	return nil
}

// MarshalYAML implements yaml.Marshaler.
func (d Duration) MarshalYAML() (interface{}, error) {
	return time.Duration(d).String(), nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	duration, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(duration)
	return nil
}

// DefaultConfig returns a configuration with sensible defaults.
func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Name:                  "StackQL MCP Server",
			Version:               "0.1.0",
			Description:           "Model Context Protocol server for StackQL",
			MaxConcurrentRequests: 100,
			Transport:             serverTransportStdIO,
			Address:               DefaultHTTPServerAddress,
			RequestTimeout:        Duration(30 * time.Second),
			Mode:                  policy.ModeSafe,
		},
		Backend: BackendConfig{
			Type:              "stackql",
			ConnectionString:  "stackql://localhost",
			MaxConnections:    10,
			ConnectionTimeout: Duration(10 * time.Second),
			QueryTimeout:      Duration(30 * time.Second),
		},
	}
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	rv := defaultConfig()
	return rv
}

func DefaultHTTPConfig() *Config {
	rv := defaultConfig()
	rv.Server.Transport = serverTransportHTTP
	return rv
}

func DefaultSSEConfig() *Config {
	rv := defaultConfig()
	rv.Server.Transport = serverTransportSSE
	return rv
}

// Validate validates the configuration and returns an error if invalid.
func (c *Config) Validate() error {
	if !policy.IsLegalMode(c.Server.Mode) {
		return fmt.Errorf("invalid server.mode %q (legal: read_only, safe, delete_safe, full_access)", c.Server.Mode)
	}
	switch c.Server.Audit.GetFailureMode() {
	case audit.FailureModeStrict, audit.FailureModeStrictMutations, audit.FailureModeBestEffort:
	default:
		return fmt.Errorf("invalid server.audit.failure_mode %q (legal: strict, strict_mutations, best_effort)",
			c.Server.Audit.FailureMode)
	}
	return nil
}

// LoadFromJSON loads configuration from JSON data.
func LoadFromJSON(data []byte) (*Config, error) {
	config := &Config{}
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON config: %w", err)
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return config, nil
}

// LoadFromYAML loads configuration from YAML data.
func LoadFromYAML(data []byte) (*Config, error) {
	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return config, nil
}
