package constants

const (
	AnalyticsPrefix                    string = "stackql_analytics"
	AuthTypeDelimiter                  string = "::"
	AuthTypeSQLDataSourcePrefix        string = "sql_data_source"
	GoogleV1DiscoveryDoc               string = "https://www.googleapis.com/discovery/v1/apis"
	GoogleV1OperationURLPropertyString string = "selfLink"
	GoogleV1ProviderCacheName          string = "google_provider_v_0_3_7"
	stackqlKeyTmplStr                  string = "__KEY_TEMPLATE__"
	stackqlPathKey                     string = "name"
	ServiceAccountRevokeErrStr         string = `[INFO] Only interactive login credentials can be revoked, to authenticate with a different service account change the credentialsfilepath in the .stackqlrc file or reauthenticate with a different service account using the AUTH command`
	ServiceAccountPathErrStr           string = `[ERROR] credentialsfilepath not supplied or key file does not exist.`
	OAuthInteractiveAuthErrStr         string = `[INFO] Interactive credentials must be revoked before logging in with a different user, use the AUTH REVOKE command before attempting to authenticate again.`
	NotAuthenticatedShowStr            string = `[INFO] Not authenticated, use the AUTH command to authenticate to a provider.`
	JsonStr                            string = "json"
	TableStr                           string = "table"
	CSVStr                             string = "csv"
	TextStr                            string = "text"
	PostgresIDMaxWidth                 int    = 63
	PrettyTextStr                      string = "pptext"
	DbEngineSQLite3Embedded            string = "sqlite3_embedded"
	DbEnginePostgresTCP                string = "postgres_tcp"
	DbEngineSnowflakeTCP               string = "snowflake_tcp"
	DbEngineDefault                    string = DbEngineSQLite3Embedded
	SQLDialectSQLite3                  string = "sqlite3"
	SQLDialectPostgres                 string = "postgres"
	SQLDialectSnowflake                string = "snowflake"
	SQLDbNameSnowflake                 string = "snowflake"
	SQLDialectDefault                  string = SQLDialectSQLite3
	SQLFuncJSONExtractSQLite           string = "json_extract"
	SQLFuncJSONExtractPostgres         string = "json_extract_path_text"
	SQLFuncJSONExtractConformed        string = SQLFuncJSONExtractSQLite
	SQLFuncGroupConcatSQLite           string = "group_concat"
	SQLFuncGroupConcatPostgres         string = "string_agg"
	SQLFuncGroupConcatConformed        string = SQLFuncGroupConcatSQLite
	DefaulHttpBodyFormat               string = JsonStr
	RequestBodyKeyPrefix               string = "data"
	RequestBodyKeyDelimiter            string = "__"
	RequestBodyBaseKey                 string = RequestBodyKeyPrefix + RequestBodyKeyDelimiter
	DefaultPrettyPrintBaseIndent       int    = 2
	DefaultPrettyPrintIndent           int    = 2
	DefaultQueryCacheSize              int    = 10000
	MaxDigits32BitUnsigned             int    = 10
	DefaultAnalyticsTemplateString     string = "stackql_analytics_{{ .objectName }}"
	DefaultViewsTemplateString         string = "stackql_views.{{ .objectName }}"
	DefaultAnalyticsRegexpString       string = `^stackql_analytics_(?P<objectName>.*)$`
	DefaultViewsRegexpString           string = `^stackql_views\.(?P<objectName>.*)$`
)

const (
	SQLDataSourceSchemaPostgresInfo string = "pgi"
	SQLDataSourceSchemaDefault      string = SQLDataSourceSchemaPostgresInfo
)

const (
	LimitsIndirectMaxChainLength int = 1
)

type GCStatus int

const (
	GCWhite GCStatus = iota
	GCBlack
	GCGrey
)

type BackendQueryType int

const (
	BackendExec BackendQueryType = iota
	BackendQuery
	BackendNop
	BackendTableObject
)
