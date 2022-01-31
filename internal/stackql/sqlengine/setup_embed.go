package sqlengine

import _ "embed"

//go:embed sql/sqlengine-setup.ddl
var sqlEngineSetupDDL string

//go:embed sql/unreachable-tables-query-gen.sql
var unreachableTablesQuery string

//go:embed sql/remove-obsolete-query-gen.sql
var cleanupObsoleteQuery string

//go:embed sql/remove-obsolete-qualified-query-gen.sql
var cleanupObsoleteQualifiedQuery string
