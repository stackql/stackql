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
	// Server and environment info
	ServerInfo(ctx context.Context, args any) (dto.ServerInfoOutput, error)

	// Current DB identity details
	DBIdentity(ctx context.Context, args any) (map[string]any, error)

	Greet(ctx context.Context, args dto.GreetInput) (string, error)

	// Execute a SQL query with typed input (preferred)
	RunQuery(ctx context.Context, args dto.QueryInput) (string, error)

	// Execute a SQL query and return JSON rows with typed input (preferred)
	RunQueryJSON(ctx context.Context, input dto.QueryJSONInput) ([]map[string]interface{}, error)

	// List resource URIs for tables in a schema
	// ListTableResources(ctx context.Context, hI HierarchyInput) ([]string, error)

	// Read rows from a table resource
	// ReadTableResource(ctx context.Context, hI HierarchyInput) ([]map[string]interface{}, error)

	// Prompt: guidelines for writing safe SELECT queries
	PromptWriteSafeSelectTool(ctx context.Context, args dto.HierarchyInput) (string, error)

	// Prompt: tips for reading EXPLAIN ANALYZE output
	// PromptExplainPlanTipsTool(ctx context.Context) (string, error)

	// List tables in a schema with optional filters and return JSON rows
	ListTablesJSON(ctx context.Context, input dto.ListTablesInput) ([]map[string]interface{}, error)

	// List tables with pagination and filters
	ListTablesJSONPage(ctx context.Context, input dto.ListTablesPageInput) (map[string]interface{}, error)

	// List all schemas in the database
	ListProviders(ctx context.Context) (string, error)

	ListServices(ctx context.Context, hI dto.HierarchyInput) (string, error)

	ListResources(ctx context.Context, hI dto.HierarchyInput) (string, error)

	ListMethods(ctx context.Context, hI dto.HierarchyInput) (string, error)

	// List all tables in a specific schema
	// ListTables(ctx context.Context, hI HierarchyInput) (string, error)

	// Get detailed information about a table
	DescribeTable(ctx context.Context, hI dto.HierarchyInput) (string, error)

	// Get foreign key information for a table
	GetForeignKeys(ctx context.Context, hI dto.HierarchyInput) (string, error)

	// Find both explicit and implied relationships for a table
	FindRelationships(ctx context.Context, hI dto.HierarchyInput) (string, error)
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
