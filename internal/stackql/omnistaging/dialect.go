package omnistaging

import (
	"fmt"
	"strings"

	"github.com/stackql/any-sdk/pkg/constants"
)

const (
	sqliteQueryTablePrefix  = "__iql__.queries."
	sqliteQueryIDSeqTable   = "__iql__.query_id_seq"
	postgresQueryIDSequence = "query_id_seq"
	// Per-statement bind parameter limits.
	sqliteMaxBindParameters   = 32766
	postgresMaxBindParameters = 65535
)

// Dialect renders the backend-specific SQL for query staging.
type Dialect interface {
	SetupStatements() []string
	NextQueryIDStatement() string
	TableName(id QueryID) string
	CreateTableStatement(id QueryID, columns []Column) (string, error)
	InsertStatement(id QueryID, columns []Column, rowCount int) (string, error)
	DropTableStatement(id QueryID) string
	MaxBindParameters() int
}

// NewDialect selects a dialect from an any-sdk SQL dialect name.
func NewDialect(sqlDialect string, cfg Config) (Dialect, error) {
	switch sqlDialect {
	case constants.SQLDialectSQLite3:
		return newSQLiteDialect(), nil
	case constants.SQLDialectPostgres:
		return newPostgresDialect(cfg), nil
	default:
		return nil, fmt.Errorf("omnistaging: unsupported sql dialect %q", sqlDialect)
	}
}

func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func quoteLiteral(s string) string {
	return `'` + strings.ReplaceAll(s, `'`, `''`) + `'`
}

func createTableStatement(tableName string, columns []Column) (string, error) {
	if len(columns) == 0 {
		return "", fmt.Errorf("omnistaging: cannot create %s without columns", tableName)
	}
	defs := make([]string, 0, len(columns))
	for _, col := range columns {
		if col.Name() == "" || col.RelationalType() == "" {
			return "", fmt.Errorf("omnistaging: column requires name and relational type")
		}
		defs = append(defs, quoteIdentifier(col.Name())+" "+col.RelationalType())
	}
	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", tableName, strings.Join(defs, ", ")), nil
}

func insertStatement(
	tableName string,
	columns []Column,
	rowCount int,
	maxBindParameters int,
	placeholder func(int) string,
) (string, error) {
	if len(columns) == 0 || rowCount < 1 {
		return "", fmt.Errorf("omnistaging: insert into %s requires columns and rows", tableName)
	}
	if len(columns)*rowCount > maxBindParameters {
		return "", fmt.Errorf("omnistaging: insert into %s exceeds %d bind parameters", tableName, maxBindParameters)
	}
	names := make([]string, 0, len(columns))
	for _, col := range columns {
		names = append(names, quoteIdentifier(col.Name()))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "INSERT INTO %s (%s) VALUES ", tableName, strings.Join(names, ", "))
	ordinal := 1
	for r := 0; r < rowCount; r++ {
		if r > 0 {
			b.WriteString(", ")
		}
		b.WriteByte('(')
		for c := range columns {
			if c > 0 {
				b.WriteString(", ")
			}
			b.WriteString(placeholder(ordinal))
			ordinal++
		}
		b.WriteByte(')')
	}
	return b.String(), nil
}

type sqliteDialect struct{}

func newSQLiteDialect() Dialect {
	return &sqliteDialect{}
}

func (d *sqliteDialect) SetupStatements() []string {
	return []string{
		fmt.Sprintf(
			"CREATE TABLE IF NOT EXISTS %s (id INTEGER PRIMARY KEY AUTOINCREMENT)",
			quoteIdentifier(sqliteQueryIDSeqTable),
		),
	}
}

// AUTOINCREMENT never reuses a rowid, so IDs stay unique across processes
// sharing one database file.
func (d *sqliteDialect) NextQueryIDStatement() string {
	return fmt.Sprintf("INSERT INTO %s DEFAULT VALUES RETURNING id", quoteIdentifier(sqliteQueryIDSeqTable))
}

func (d *sqliteDialect) TableName(id QueryID) string {
	return quoteIdentifier(sqliteQueryTablePrefix + id.String())
}

func (d *sqliteDialect) CreateTableStatement(id QueryID, columns []Column) (string, error) {
	return createTableStatement(d.TableName(id), columns)
}

func (d *sqliteDialect) InsertStatement(id QueryID, columns []Column, rowCount int) (string, error) {
	return insertStatement(d.TableName(id), columns, rowCount, d.MaxBindParameters(), func(int) string { return "?" })
}

func (d *sqliteDialect) DropTableStatement(id QueryID) string {
	return "DROP TABLE IF EXISTS " + d.TableName(id)
}

func (d *sqliteDialect) MaxBindParameters() int {
	return sqliteMaxBindParameters
}

type postgresDialect struct {
	querySchema string
}

func newPostgresDialect(cfg Config) Dialect {
	return &postgresDialect{querySchema: cfg.QuerySchema()}
}

func (d *postgresDialect) sequenceName() string {
	return quoteIdentifier(d.querySchema) + "." + quoteIdentifier(postgresQueryIDSequence)
}

func (d *postgresDialect) SetupStatements() []string {
	return []string{
		"CREATE SCHEMA IF NOT EXISTS " + quoteIdentifier(d.querySchema),
		"CREATE SEQUENCE IF NOT EXISTS " + d.sequenceName(),
	}
}

func (d *postgresDialect) NextQueryIDStatement() string {
	return fmt.Sprintf("SELECT nextval(%s)", quoteLiteral(d.sequenceName()))
}

func (d *postgresDialect) TableName(id QueryID) string {
	return quoteIdentifier(d.querySchema) + "." + quoteIdentifier(id.String())
}

func (d *postgresDialect) CreateTableStatement(id QueryID, columns []Column) (string, error) {
	return createTableStatement(d.TableName(id), columns)
}

func (d *postgresDialect) InsertStatement(id QueryID, columns []Column, rowCount int) (string, error) {
	return insertStatement(
		d.TableName(id), columns, rowCount, d.MaxBindParameters(),
		func(ordinal int) string { return fmt.Sprintf("$%d", ordinal) },
	)
}

func (d *postgresDialect) DropTableStatement(id QueryID) string {
	return "DROP TABLE IF EXISTS " + d.TableName(id)
}

func (d *postgresDialect) MaxBindParameters() int {
	return postgresMaxBindParameters
}
