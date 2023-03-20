package dto

const (
	AuthAPIKeyStr                   string = "api_key"
	AuthAWSSigningv4Str             string = "aws_signing_v4"
	AuthAzureDefaultStr             string = "azure_default"
	AuthBasicStr                    string = "basic"
	AuthBearerStr                   string = "bearer"
	AuthInteractiveStr              string = "interactive"
	AuthServiceAccountStr           string = "service_account"
	AuthNullStr                     string = "null_auth"
	DarkColorScheme                 string = "dark"
	LightColorScheme                string = "light"
	NullColorScheme                 string = "null"
	DefaultColorScheme              string = DarkColorScheme
	DefaultWindowsColorScheme       string = NullColorScheme
	DryRunFlagKey                   string = "dryrun"
	ExecutionConcurrencyLimitKey    string = "execution.concurrency.limit"
	AuthCtxKey                      string = "auth"
	APIRequestTimeoutKey            string = "apirequesttimeout"
	CacheKeyCountKey                string = "cachekeycount"
	CacheTTLKey                     string = "metadatattl"
	ColorSchemeKey                  string = "colorscheme"
	ConfigFilePathKey               string = "configfile"
	CPUProfileKey                   string = "cpuprofile"
	CSVHeadersDisableKey            string = "hideheaders"
	DelimiterKey                    string = "delimiter"
	ErrorPresentationKey            string = "errorpresentation"
	IndirectDepthMaxKey             string = "indirect.depth.max"
	DataflowDependencyMaxKey        string = "dataflow.dependency.max"
	HTTPLogEnabledKey               string = "http.log.enabled"
	HTTPMaxResultsKey               string = "http.response.maxResults"
	HTTPPAgeLimitKey                string = "http.response.pageLimit"
	HTTPProxyHostKey                string = "http.proxy.host"
	HTTPProxyPasswordKey            string = "http.proxy.password" //nolint:gosec // no hardcoded credentials
	HTTPProxyPortKey                string = "http.proxy.port"
	HTTPProxySchemeKey              string = "http.proxy.scheme"
	HTTPProxyUserKey                string = "http.proxy.user"
	CABundleKey                     string = "tls.CABundle"
	AllowInsecureKey                string = "tls.allowInsecure"
	InfilePathKey                   string = "infile"
	LogLevelStrKey                  string = "loglevel"
	OutfilePathKey                  string = "outfile"
	OutputFormatKey                 string = "output"
	ApplicationFilesRootPathKey     string = "approot"
	ApplicationFilesRootPathModeKey string = "approotfilemode"
	PgSrvAddressKey                 string = "pgsrv.address"
	PgSrvLogLevelKey                string = "pgsrv.loglevel"
	PgSrvPortKey                    string = "pgsrv.port"
	PgSrvRawTLSCfgKey               string = "pgsrv.tls"
	ProviderStrKey                  string = "provider"
	QueryCacheSizeKey               string = "querycachesize"
	RegistryRawKey                  string = "registry"
	GCCfgRawKey                     string = "gc"
	NamespaceCfgRawKey              string = "namespaces"
	SQLBackendCfgRawKey             string = "sqlBackend"
	DBInternalCfgRawKey             string = "dbInternal"
	StoreTxnCfgRawKey               string = "store.txn"
	TemplateCtxFilePathKey          string = "iqldata"
	TestWithoutAPICallsKey          string = "TestWithoutAPICalls"
	UseNonPreferredAPIsKEy          string = "usenonpreferredapis"
	VerboseFlagKey                  string = "verbose"
	ViperCfgFileNameKey             string = "viperconfigfilename"
	WorkOfflineKey                  string = "offline"
)

func inferKeyFileType(keyFileType string) string {
	if keyFileType == "" {
		return AuthServiceAccountStr
	}
	return keyFileType
}
