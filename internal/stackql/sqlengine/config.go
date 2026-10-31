package sqlengine

// Config is the host's resolved database connection configuration. Environment
// variables are resolved by the caller before constructing it.
type Config struct {
	RequireForeignKeys    bool   `json:"-"`
	DBEngine              string `json:"dbEngine"`
	DSN                   string `json:"dsn"`
	DbInitFilePath        string `json:"dbInitFilepath"`
	InitMaxRetries        int    `json:"initMaxRetries"`
	InitRetryInitialDelay int    `json:"initRetryInitialDelay"`
}
