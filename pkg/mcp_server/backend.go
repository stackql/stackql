package mcp_server //nolint:revive // fine for now

import (
	"context"
	"database/sql/driver"

	"github.com/stackql/stackql/pkg/mcp_server/dto"
)

type Backend interface {

	// Ping verifies the backend connection is active.
	Ping(ctx context.Context) error

	// Close gracefully shuts down the backend connection.
	Close() error

	// ServerInfo returns server identity and runtime metadata.
	ServerInfo(ctx context.Context, args any) (dto.ServerInfoOutput, error)

	// ExecQuery executes a non-row-returning SQL statement (mutations, EXEC).
	ExecQuery(ctx context.Context, query string) (map[string]any, error)

	// ValidateQuery parses and plans a SELECT without executing it.
	ValidateQuery(ctx context.Context, query string) ([]map[string]any, error)

	// RunQueryJSON executes a SELECT and returns the rows.
	RunQueryJSON(ctx context.Context, input dto.QueryJSONInput) ([]map[string]interface{}, error)

	// ListProviders lists available providers.
	ListProviders(ctx context.Context) ([]map[string]any, error)

	// ListServices lists services under a provider.
	ListServices(ctx context.Context, hI dto.HierarchyInput) ([]map[string]any, error)

	// ListResources lists resources under a provider/service.
	ListResources(ctx context.Context, hI dto.HierarchyInput) ([]map[string]any, error)

	// ListMethods lists access methods for a resource.
	ListMethods(ctx context.Context, hI dto.HierarchyInput) ([]map[string]any, error)

	// DescribeResource returns the output fields for a resource's primary read method.
	DescribeResource(ctx context.Context, hI dto.HierarchyInput) ([]map[string]any, error)

	// DescribeMethod returns the full I/O contract for one method.
	DescribeMethod(ctx context.Context, hI dto.HierarchyInput) ([]map[string]any, error)

	// ListRegistry lists providers (and optionally their versions) available
	// in the provider registry. Distinct from ListProviders, which lists
	// providers already pulled into the local approot cache.
	ListRegistry(ctx context.Context, rI dto.RegistryInput) ([]map[string]any, error)

	// PullProvider installs a provider (optionally pinned to a version) into
	// the approot cache so subsequent queries can resolve it. Writes only
	// local state, no cloud control/data-plane effect.
	PullProvider(ctx context.Context, rI dto.RegistryInput) (map[string]any, error)
}

// QueryResult represents the result of a query execution.
type QueryResult interface {
	// GetColumns returns metadata about each column in the result set.
	GetColumns() []ColumnInfo

	// GetRows returns the actual data returned by the query.
	GetRows() [][]interface{}

	// GetRowsAffected returns the number of rows affected by DML operations.
	GetRowsAffected() int64

	// GetExecutionTime returns the time taken to execute the query in milliseconds.
	GetExecutionTime() int64
}

// ColumnInfo provides metadata about a result column.
type ColumnInfo interface {
	// GetName returns the column name as returned by the query.
	GetName() string

	// GetType returns the data type of the column (e.g., "string", "int64", "float64").
	GetType() string

	// IsNullable indicates whether the column can contain null values.
	IsNullable() bool
}

// SchemaProvider represents the metadata structure of available resources.
type SchemaProvider interface {
	// GetProviders returns all available providers (e.g., aws, google, azure).
	GetProviders() []Provider
}

// Provider represents a StackQL provider with its services and resources.
type Provider interface {
	// GetName returns the provider identifier (e.g., "aws", "google").
	GetName() string

	// GetVersion returns the provider version.
	GetVersion() string

	// GetServices returns all services available in this provider.
	GetServices() []Service
}

// Service represents a service within a provider.
type Service interface {
	// GetName returns the service identifier (e.g., "ec2", "compute").
	GetName() string

	// GetResources returns all resources available in this service.
	GetResources() []Resource
}

// Resource represents a queryable resource.
type Resource interface {
	// GetName returns the resource identifier (e.g., "instances", "buckets").
	GetName() string

	// GetMethods returns the available operations for this resource.
	GetMethods() []string

	// GetFields returns the available fields in this resource.
	GetFields() []Field
}

// Field represents a field within a resource.
type Field interface {
	// GetName returns the field identifier.
	GetName() string

	// GetType returns the field data type.
	GetType() string

	// IsRequired indicates if this field is mandatory for certain operations.
	IsRequired() bool

	// GetDescription returns human-readable documentation for the field.
	GetDescription() string
}

// BackendError represents an error that occurred in the backend.
type BackendError struct {
	// Code is a machine-readable error code.
	Code string `json:"code"`

	// Message is a human-readable error message.
	Message string `json:"message"`

	// Details contains additional context about the error.
	Details map[string]interface{} `json:"details,omitempty"`
}

func (e *BackendError) Error() string {
	return e.Message
}

// Ensure BackendError implements the driver.Valuer interface for database compatibility.
func (e *BackendError) Value() (driver.Value, error) {
	return e.Message, nil
}

// Private implementations of interfaces

type queryResult struct {
	Columns       []ColumnInfo    `json:"columns"`
	Rows          [][]interface{} `json:"rows"`
	RowsAffected  int64           `json:"rows_affected"`
	ExecutionTime int64           `json:"execution_time_ms"`
}

func (qr *queryResult) GetColumns() []ColumnInfo { return qr.Columns }
func (qr *queryResult) GetRows() [][]interface{} { return qr.Rows }
func (qr *queryResult) GetRowsAffected() int64   { return qr.RowsAffected }
func (qr *queryResult) GetExecutionTime() int64  { return qr.ExecutionTime }

type columnInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

func (ci *columnInfo) GetName() string  { return ci.Name }
func (ci *columnInfo) GetType() string  { return ci.Type }
func (ci *columnInfo) IsNullable() bool { return ci.Nullable }

type schemaProvider struct {
	Providers []Provider `json:"providers"`
}

func (sp *schemaProvider) GetProviders() []Provider { return sp.Providers }

type provider struct {
	Name     string    `json:"name"`
	Version  string    `json:"version"`
	Services []Service `json:"services"`
}

func (p *provider) GetName() string        { return p.Name }
func (p *provider) GetVersion() string     { return p.Version }
func (p *provider) GetServices() []Service { return p.Services }

type service struct {
	Name      string     `json:"name"`
	Resources []Resource `json:"resources"`
}

func (s *service) GetName() string          { return s.Name }
func (s *service) GetResources() []Resource { return s.Resources }

type resource struct {
	Name    string   `json:"name"`
	Methods []string `json:"methods"`
	Fields  []Field  `json:"fields"`
}

func (r *resource) GetName() string      { return r.Name }
func (r *resource) GetMethods() []string { return r.Methods }
func (r *resource) GetFields() []Field   { return r.Fields }

type field struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

func (f *field) GetName() string        { return f.Name }
func (f *field) GetType() string        { return f.Type }
func (f *field) IsRequired() bool       { return f.Required }
func (f *field) GetDescription() string { return f.Description }

// Factory functions

// NewQueryResult creates a new QueryResult instance.
func NewQueryResult(columns []ColumnInfo, rows [][]interface{}, rowsAffected, executionTime int64) QueryResult {
	return &queryResult{
		Columns:       columns,
		Rows:          rows,
		RowsAffected:  rowsAffected,
		ExecutionTime: executionTime,
	}
}

// NewColumnInfo creates a new ColumnInfo instance.
func NewColumnInfo(name, colType string, nullable bool) ColumnInfo {
	return &columnInfo{
		Name:     name,
		Type:     colType,
		Nullable: nullable,
	}
}

// NewSchemaProvider creates a new SchemaProvider instance.
func NewSchemaProvider(providers []Provider) SchemaProvider {
	return &schemaProvider{
		Providers: providers,
	}
}

// NewProvider creates a new Provider instance.
func NewProvider(name, version string, services []Service) Provider {
	return &provider{
		Name:     name,
		Version:  version,
		Services: services,
	}
}

// NewService creates a new Service instance.
func NewService(name string, resources []Resource) Service {
	return &service{
		Name:      name,
		Resources: resources,
	}
}

// NewResource creates a new Resource instance.
func NewResource(name string, methods []string, fields []Field) Resource {
	return &resource{
		Name:    name,
		Methods: methods,
		Fields:  fields,
	}
}

// NewField creates a new Field instance.
func NewField(name, fieldType string, required bool, description string) Field {
	return &field{
		Name:        name,
		Type:        fieldType,
		Required:    required,
		Description: description,
	}
}
