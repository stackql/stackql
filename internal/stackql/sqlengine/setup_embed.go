package sqlengine

import (
	_ "embed"
	"fmt"
)

//go:embed sql/sqlite/sqlengine-setup.ddl
var sqLiteEngineSetupDDL string

//go:embed sql/postgres/sqlengine-setup.ddl
var postgresEngineSetupDDL string

func GetSQLEngineSetupDDL(engineType string) (string, error) {
	switch engineType {
	case "sqlite":
		return sqLiteEngineSetupDDL, nil
	case "postgres":
		return postgresEngineSetupDDL, nil
	default:
		return "", fmt.Errorf("unsupported engine type: %s", engineType)
	}
}
