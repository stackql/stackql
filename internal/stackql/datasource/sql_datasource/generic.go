package sql_datasource //nolint:revive,stylecheck // package name is helpful

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"     //nolint:revive,nolintlint // this is a DB driver pattern
	_ "github.com/snowflakedb/gosnowflake" //nolint:revive,nolintlint // this is a DB driver pattern

	"github.com/stackql/any-sdk/pkg/constants"
	"github.com/stackql/any-sdk/pkg/db/db_util"
	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/stackql/internal/stackql/datasource/sqltable"
)

var (
	_         SQLDataSource = &genericSQLDataSource{}
	getDBFunc               = db_util.GetDB //nolint:gochecknoglobals // patching variable
)

func NewGenericSQLDataSource(authCtx *dto.AuthCtx, driverName string, dbName string) (SQLDataSource, error) {
	sqlCfg, hasSQLCfg := authCtx.GetSQLCfg()
	if !hasSQLCfg {
		return nil, fmt.Errorf("cannot init %s data source with empty SQL config", dbName)
	}
	db, err := getDBFunc(driverName, dbName, sqlCfg)
	if err != nil {
		return nil, err
	}
	schemaType := sqlCfg.GetSchemaType()
	if schemaType == "" {
		schemaType = constants.SQLDataSourceSchemaDefault
	}
	return &genericSQLDataSource{
		db:         db,
		dbName:     dbName,
		schemaType: schemaType,
	}, nil
}

type genericSQLDataSource struct {
	db         *sql.DB
	dbName     string
	schemaType string
}

func (ds *genericSQLDataSource) GetSchemaType() string {
	return ds.schemaType
}

func (ds *genericSQLDataSource) GetDBName() string {
	return ds.dbName
}

func (ds *genericSQLDataSource) Exec(query string, args ...interface{}) (sql.Result, error) {
	return ds.db.Exec(query, args...)
}

func (ds *genericSQLDataSource) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return ds.db.Query(query, args...)
}

func (ds *genericSQLDataSource) QueryRow(query string, args ...interface{}) *sql.Row {
	return ds.db.QueryRow(query, args...)
}

func (ds *genericSQLDataSource) Begin() (*sql.Tx, error) {
	return ds.db.Begin()
}

func (ds *genericSQLDataSource) GetTableMetadata(args ...string) (sqltable.SQLTable, error) {
	return nil, fmt.Errorf("could not obtain sql data source table metadata for args = '%v'", args)
}

// func (ds *genericSQLDataSource) getPostgresTableMetadata(schemaName, tableName string) (sqltable.SQLTable, error) {
// 	queryTemplate := `
// 	SELECT
// 		column_name,
// 		data_type
// 	FROM
// 		information_schema.columns
// 	WHERE
// 		table_name = ?
// 		AND
// 		table_schema = ?;
// 	`
// 	return nil, fmt.Errorf("could not obtain sql data source table metadata for table = '%s'", tableName)
// }
