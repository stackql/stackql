package sqldialect

import _ "embed"

//go:embed sql/sqlite/sqlengine-setup.ddl
var sqlEngineSetupDDL string
