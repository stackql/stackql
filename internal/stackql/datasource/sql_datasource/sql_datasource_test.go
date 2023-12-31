package sql_datasource //nolint:testpackage,stylecheck // test package

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/datasource/sqltable"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type SQLDataSourceMock struct {
	mock.Mock
}

func (m *SQLDataSourceMock) Begin() (*sql.Tx, error) {
	args := m.Called()
	return args.Get(0).(*sql.Tx), args.Error(1)
}

func (m *SQLDataSourceMock) Exec(query string, args ...interface{}) (sql.Result, error) {
	args1 := m.Called(query, args)
	return args1.Get(0).(sql.Result), args1.Error(1)
}

func (m *SQLDataSourceMock) Query(query string, args ...interface{}) (*sql.Rows, error) {
	args1 := m.Called(query, args)
	return args1.Get(0).(*sql.Rows), args1.Error(1)
}

func (m *SQLDataSourceMock) QueryRow(query string, args ...interface{}) *sql.Row {
	args1 := m.Called(query, args)
	return args1.Get(0).(*sql.Row)
}

func (m *SQLDataSourceMock) GetTableMetadata(args ...string) (sqltable.SQLTable, error) {
	args1 := m.Called(args)
	return args1.Get(0).(sqltable.SQLTable), args1.Error(1)
}

func (m *SQLDataSourceMock) GetSchemaType() string {
	args := m.Called()
	return args.String(0)
}

func (m *SQLDataSourceMock) GetDBName() string {
	args := m.Called()
	return args.String(0)
}

func genericSQLDataSourceFuncMock(authCtx *dto.AuthCtx, driverName, dbName string) (SQLDataSource, error) {
	// Mock implementation
	return &SQLDataSourceMock{}, nil
}

func TestNewDataSource(t *testing.T) {
	t.Run("authCtx is nil", func(t *testing.T) {
		ds, err := NewDataSource(nil, nil)
		assert.Error(t, err)
		assert.Nil(t, ds)
		assert.Equal(t, "cannot create sql data source from nil auth context", err.Error())
	})

	t.Run("authCtx is not supported", func(t *testing.T) {
		authCtx := &dto.AuthCtx{Type: "not_supported"}
		ds, err := NewDataSource(authCtx, nil)
		assert.Error(t, err)
		assert.Nil(t, ds)
		assert.Equal(t, fmt.Sprintf("sql data source of type '%s' not supported", authCtx.Type), err.Error())
	})

	t.Run("authCtx.Type is snowflake", func(t *testing.T) {
		authCtx := &dto.AuthCtx{Type: fmt.Sprintf(
			"%s%s%s",
			constants.AuthTypeSQLDataSourcePrefix,
			constants.AuthTypeDelimiter,
			"snowflake",
		)}
		ds, err := NewDataSource(authCtx, genericSQLDataSourceFuncMock)
		assert.NotNil(t, ds)
		assert.Nil(t, err)
	})

	t.Run("authCtx.Type is postgres", func(t *testing.T) {
		authCtx := &dto.AuthCtx{Type: fmt.Sprintf(
			"%s%s%s",
			constants.AuthTypeSQLDataSourcePrefix,
			constants.AuthTypeDelimiter,
			"postgres",
		)}
		ds, err := NewDataSource(authCtx, genericSQLDataSourceFuncMock)
		assert.NotNil(t, ds)
		assert.Nil(t, err)
	})
}
