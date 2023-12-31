package sql_datasource //nolint:testpackage,stylecheck // test package

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockDBUtil struct {
	mock.Mock
}

func (m *MockDBUtil) GetDB(driverName, dbName string, sqlCfg *dto.SQLBackendCfg) (*sql.DB, error) {
	args := m.Called(driverName, dbName, sqlCfg)
	return args.Get(0).(*sql.DB), args.Error(1)
}

func TestNewGenericSQLDataSource(t *testing.T) {
	dbName := "test"
	driverName := "test"
	authCtx := &dto.AuthCtx{}
	authCtx.SQLCfg = &dto.SQLBackendCfg{
		DBEngine: "test",
	}

	db, mock, stubErr := sqlmock.New()
	if stubErr != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", stubErr)
	}
	defer db.Close()

	//nolint:unparam // patch function
	getDBFuncPatch := func(string, string, dto.SQLBackendCfg) (*sql.DB, error) {
		return db, nil
	}

	t.Run("check if sqlcfg is nil", func(t *testing.T) {
		testAuthCtx := &dto.AuthCtx{}
		_, err := NewGenericSQLDataSource(testAuthCtx, driverName, dbName)
		assert.Equal(t, fmt.Sprintf("cannot init %s data source with empty SQL config", dbName), err.Error())
	})

	t.Run("get db name", func(t *testing.T) {
		getDBFunc = getDBFuncPatch
		res, _ := NewGenericSQLDataSource(authCtx, driverName, dbName)
		assert.Equal(t, dbName, res.GetDBName())
	})

	t.Run("check if schematype value is default", func(t *testing.T) {
		getDBFunc = getDBFuncPatch
		res, _ := NewGenericSQLDataSource(authCtx, driverName, dbName)
		assert.Equal(t, constants.SQLDataSourceSchemaDefault, res.GetSchemaType())
	})

	t.Run("test exec", func(t *testing.T) {
		getDBFunc = getDBFuncPatch
		res, _ := NewGenericSQLDataSource(authCtx, driverName, dbName)
		mock.ExpectExec("^INSERT INTO test VALUES \\(\\)$").WillReturnResult(sqlmock.NewResult(1, 1))
		_, err := res.Exec("INSERT INTO test VALUES ()")
		assert.NoError(t, err)
	})

	t.Run("test query", func(t *testing.T) {
		getDBFunc = getDBFuncPatch
		res, _ := NewGenericSQLDataSource(authCtx, driverName, dbName)
		mock.ExpectQuery("^SELECT \\* FROM test$").WillReturnRows(sqlmock.NewRows([]string{"id"}))

		//nolint:rowserrcheck // row will be closed
		row, err := res.Query("SELECT * FROM test")
		assert.NoError(t, err)
		defer row.Close()
	})

	t.Run("test query row", func(t *testing.T) {
		getDBFunc = getDBFuncPatch
		res, _ := NewGenericSQLDataSource(authCtx, driverName, dbName)
		mock.ExpectQuery("^SELECT \\* FROM test$").WillReturnRows(sqlmock.NewRows([]string{"id"}))
		row := res.QueryRow("SELECT * FROM test")
		assert.NotNil(t, row)
	})

	t.Run("test begin", func(t *testing.T) {
		getDBFunc = getDBFuncPatch
		res, _ := NewGenericSQLDataSource(authCtx, driverName, dbName)
		mock.ExpectBegin()
		tx, err := res.Begin()
		assert.NoError(t, err)
		assert.NotNil(t, tx)
	})

	t.Run("test get table metadata", func(t *testing.T) {
		getDBFunc = getDBFuncPatch
		res, _ := NewGenericSQLDataSource(authCtx, driverName, dbName)
		args := []string{"arg1", "arg2"}
		_, err := res.GetTableMetadata(args...)
		assert.Equal(t, fmt.Sprintf("could not obtain sql data source table metadata for args = '%v'", args), err.Error())
	})
}
