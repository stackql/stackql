package sql_datasource //nolint:stylecheck,revive // package name is helpful

import (
	"database/sql"
	"fmt"

	"github.com/stackql/any-sdk/pkg/constants"
	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/stackql/internal/stackql/datasource/sqltable"
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

type genericSQLDataSourceFunc func(*dto.AuthCtx, string, string) (SQLDataSource, error)

func NewDataSource(authCtx *dto.AuthCtx, genericSQL genericSQLDataSourceFunc) (SQLDataSource, error) {
	if authCtx == nil {
		return nil, fmt.Errorf("cannot create sql data source from nil auth context")
	}
	if authCtx.Type == fmt.Sprintf(
		"%s%s%s",
		constants.AuthTypeSQLDataSourcePrefix,
		constants.AuthTypeDelimiter,
		"snowflake",
	) {
		return genericSQL(authCtx, "snowflake", "snowflake")
	}
	if authCtx.Type == fmt.Sprintf(
		"%s%s%s",
		constants.AuthTypeSQLDataSourcePrefix,
		constants.AuthTypeDelimiter,
		"postgres",
	) {
		return genericSQL(authCtx, "pgx", "postgres")
	}
	return nil, fmt.Errorf("sql data source of type '%s' not supported", authCtx.Type)
}
