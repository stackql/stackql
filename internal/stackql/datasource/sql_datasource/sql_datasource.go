package sql_datasource //nolint:revive,stylecheck // package name is helpful

import (
	"database/sql"
	"fmt"

	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/datasource/sqltable"
	"github.com/stackql/stackql/internal/stackql/dto"
)

type SQLDataSource interface {
	Begin() (*sql.Tx, error)
	Exec(string, ...interface{}) (sql.Result, error)
	Query(string, ...interface{}) (*sql.Rows, error)
	QueryRow(string, ...any) *sql.Row
	GetTableMetadata(...string) (sqltable.SQLTable, error)
	GetSchemaType() string
	GetDBName() string
}

func NewDataSource(authCtx *dto.AuthCtx) (SQLDataSource, error) {
	if authCtx == nil {
		return nil, fmt.Errorf("cannot create sql data source from nil auth context")
	}
	if authCtx.Type == fmt.Sprintf(
		"%s%s%s",
		constants.AuthTypeSQLDataSourcePrefix,
		constants.AuthTypeDelimiter,
		"snowflake",
	) {
		return newGenericSQLDataSource(authCtx, "snowflake", "snowflake")
	}
	if authCtx.Type == fmt.Sprintf(
		"%s%s%s",
		constants.AuthTypeSQLDataSourcePrefix,
		constants.AuthTypeDelimiter,
		"postgres",
	) {
		return newGenericSQLDataSource(authCtx, "pgx", "postgres")
	}
	return nil, fmt.Errorf("sql data source of type '%s' not supported", authCtx.Type)
}
