package sql_system //nolint:revive,stylecheck // package name is meaningful and readable

import (
	"github.com/stackql/any-sdk/public/sqlengine"
)

//nolint:checknoglobals,gochecknoglobals // fine with this
var (
	sqLiteEngineSetupDDL, _   = sqlengine.GetSQLEngineSetupDDL("sqlite")
	postgresEngineSetupDDL, _ = sqlengine.GetSQLEngineSetupDDL("postgres")
)
