// Package omnistaging owns RDBMS staging for omnisdk-backed queries.
//
// Each top-level query receives one collision-free query ID. When remaining
// SQL work requires materialization, the final Omni relation for that query
// is staged in exactly one query-owned table. Child query IDs exist only for
// genuine pre-analysis splits and are released with their parent.
//
// See docs/technical/omnisdk_staging.md.
package omnistaging

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"gopkg.in/yaml.v2"
)

// DefaultQuerySchema is the PostgreSQL schema holding query tables when
// none is configured.
const DefaultQuerySchema = "stackql_queries"

// Executor is the subset of database access required for staging.
// *sql.DB and *sql.Tx satisfy it.
type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// QueryID identifies one query and its staging namespace.
type QueryID interface {
	Value() int64
	String() string
}

// Column describes one column of a staged relation.
type Column interface {
	Name() string
	RelationalType() string
}

// BatchSource yields batches of rows in cursor order. Next returns io.EOF
// once the source is exhausted. It is pulled only as fast as batches are
// written, so the producer is back-pressured by staging.
type BatchSource interface {
	Next(ctx context.Context) ([][]any, error)
}

// Config carries staging configuration.
type Config interface {
	QuerySchema() string
}

type queryID struct {
	value int64
}

func newQueryID(value int64) QueryID {
	return &queryID{value: value}
}

func (q *queryID) Value() int64 {
	return q.value
}

func (q *queryID) String() string {
	return strconv.FormatInt(q.value, 10)
}

type column struct {
	name           string
	relationalType string
}

// NewColumn constructs a staged relation column.
func NewColumn(name, relationalType string) Column {
	return &column{name: name, relationalType: relationalType}
}

func (c *column) Name() string {
	return c.name
}

func (c *column) RelationalType() string {
	return c.relationalType
}

type config struct {
	querySchema string
}

// NewConfig constructs staging configuration; an empty querySchema selects
// DefaultQuerySchema.
func NewConfig(querySchema string) Config {
	if querySchema == "" {
		querySchema = DefaultQuerySchema
	}
	return &config{querySchema: querySchema}
}

// NewConfigFromSQLBackend reads staging configuration from the raw
// --sqlBackend string (JSON or YAML), key schemata.querySchema.
func NewConfigFromSQLBackend(raw string) (Config, error) {
	var parsed struct {
		Schemata struct {
			QuerySchema string `yaml:"querySchema"`
		} `yaml:"schemata"`
	}
	if err := yaml.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("omnistaging: cannot parse sql backend config: %w", err)
	}
	return NewConfig(parsed.Schemata.QuerySchema), nil
}

func (c *config) QuerySchema() string {
	return c.querySchema
}
