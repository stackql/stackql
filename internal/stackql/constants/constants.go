package constants

const (
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
	PrettyTextStr                      string = "pptext"
	DefaulHttpBodyFormat               string = JsonStr
	RequestBodyKeyPrefix               string = "data"
	RequestBodyKeyDelimiter            string = "__"
	RequestBodyBaseKey                 string = RequestBodyKeyPrefix + RequestBodyKeyDelimiter
	DefaultPrettyPrintBaseIndent       int    = 2
	DefaultPrettyPrintIndent           int    = 2
	DefaultQueryCacheSize              int    = 10000
)
