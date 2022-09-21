package sqlengine

import _ "embed"

//go:embed sql/sqlite/sqlengine-setup.ddl
var sqlEngineSetupDDL string

//go:embed sql/sqlite/unreachable-tables-query-gen.sql
var unreachableTablesQuery string

//go:embed sql/sqlite/remove-obsolete-query-gen.sql
var cleanupObsoleteQuery string

//go:embed sql/sqlite/remove-obsolete-qualified-query-gen.sql
var cleanupObsoleteQualifiedQuery string
