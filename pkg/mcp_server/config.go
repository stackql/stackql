package mcp_server //nolint:revive,stylecheck,mnd // fine for now

import (
	"encoding/json"
	"fmt"
	"time"

	"gopkg.in/yaml.v2"
)

// Config represents the complete configuration for the MCP server.
type Config struct {
	// Server contains server-specific configuration.
	Server ServerConfig `json:"server" yaml:"server"`

	// Backend contains backend-specific configuration.
	Backend BackendConfig `json:"backend" yaml:"backend"`
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

	// Description is a human-readable description of the server.
	Description string `json:"description" yaml:"description"`

	// MaxConcurrentRequests limits the number of concurrent client requests.
	MaxConcurrentRequests int `json:"max_concurrent_requests" yaml:"max_concurrent_requests"`

	// RequestTimeout specifies the timeout for individual requests.
	RequestTimeout Duration `json:"request_timeout" yaml:"request_timeout"`

	IsReadOnly *bool `json:"read_only,omitempty" yaml:"read_only,omitempty"`
}

// BackendConfig contains configuration for the backend connection.
type BackendConfig struct {
	// Type specifies the backend type ("stackql", "tcp", "memory").
	Type string `json:"type" yaml:"type"`

	// ConnectionString contains the connection details for the backend.
	// Format depends on the backend type.
	ConnectionString string `json:"connection_string" yaml:"connection_string"`

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
//
//nolint:gocognit // simple validation logic
func (c *Config) Validate() error {
	// if c.Server.Name == "" {
	// 	return fmt.Errorf("server.name is required")
	// }
	// if c.Server.Version == "" {
	// 	return fmt.Errorf("server.version is required")
	// }
	// if c.Server.MaxConcurrentRequests <= 0 {
	// 	return fmt.Errorf("server.max_concurrent_requests must be greater than 0")
	// }
	// if c.Backend.Type == "" {
	// 	return fmt.Errorf("backend.type is required")
	// }
	// if c.Backend.MaxConnections <= 0 {
	// 	return fmt.Errorf("backend.max_connections must be greater than 0")
	// }

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
