package sql_system //nolint:revive,stylecheck // package name is meaningful and readable

import _ "embed"

//go:embed sql/sqlite/sqlengine-setup.ddl
var sqLiteEngineSetupDDL string

//go:embed sql/postgres/sqlengine-setup.ddl
var postgresEngineSetupDDL string
