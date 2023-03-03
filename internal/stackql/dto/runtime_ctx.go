package dto

import "strconv"

type RuntimeCtx struct {
	APIRequestTimeout            int
	AuthRaw                      string
	CABundle                     string
	AllowInsecure                bool
	CacheKeyCount                int
	CacheTTL                     int
	ColorScheme                  string
	ConfigFilePath               string
	CPUProfile                   string
	CSVHeadersDisable            bool
	Delimiter                    string
	DryRunFlag                   bool
	ErrorPresentation            string
	ExecutionConcurrencyLimit    int
	HTTPLogEnabled               bool
	HTTPMaxResults               int
	HTTPPageLimit                int
	HTTPProxyHost                string
	HTTPProxyPassword            string
	HTTPProxyPort                int
	HTTPProxyScheme              string
	HTTPProxyUser                string
	IndirectDepthMax             int
	InfilePath                   string
	LogLevelStr                  string
	OutfilePath                  string
	OutputFormat                 string
	ApplicationFilesRootPath     string
	ApplicationFilesRootPathMode uint32
	PGSrvAddress                 string
	PGSrvLogLevel                string
	PGSrvPort                    int
	PGSrvRawTLSCfg               string
	ProviderStr                  string
	RegistryRaw                  string
	SQLBackendCfgRaw             string
	DBInternalCfgRaw             string
	NamespaceCfgRaw              string
	StoreTxnCfgRaw               string
	GCCfgRaw                     string
	QueryCacheSize               int
	TemplateCtxFilePath          string
	TestWithoutApiCalls          bool
	UseNonPreferredAPIs          bool
	VerboseFlag                  bool
	ViperCfgFileName             string
	WorkOffline                  bool
}

func setInt(iPtr *int, val string) error {
	i, err := strconv.Atoi(val)
	if err == nil {
		*iPtr = i
	}
	return err
}

func setUint32(uPtr *uint32, val string) error {
	ui, err := strconv.ParseUint(val, 10, 32)
	if err == nil {
		*uPtr = uint32(ui)
	}
	return err
}

func setBool(bPtr *bool, val string) error {
	b, err := strconv.ParseBool(val)
	if err == nil {
		*bPtr = b
	}
	return err
}

func (rc *RuntimeCtx) Set(key string, val string) error {
	var retVal error
	switch key {
	case APIRequestTimeoutKey:
		retVal = setInt(&rc.APIRequestTimeout, val)
	case AuthCtxKey:
		rc.AuthRaw = val
	case CABundleKey:
		rc.CABundle = val
	case CacheKeyCountKey:
		retVal = setInt(&rc.CacheKeyCount, val)
	case CacheTTLKey:
		retVal = setInt(&rc.CacheTTL, val)
	case ColorSchemeKey:
		rc.ColorScheme = val
	case ConfigFilePathKey:
		rc.ConfigFilePath = val
	case CPUProfileKey:
		rc.CPUProfile = val
	case CSVHeadersDisableKey:
		retVal = setBool(&rc.CSVHeadersDisable, val)
	case SQLBackendCfgRawKey:
		rc.SQLBackendCfgRaw = val
	case DBInternalCfgRawKey:
		rc.DBInternalCfgRaw = val
	case DelimiterKey:
		rc.Delimiter = val
	case DryRunFlagKey:
		retVal = setBool(&rc.DryRunFlag, val)
	case ErrorPresentationKey:
		rc.ErrorPresentation = val
	case ExecutionConcurrencyLimitKey:
		retVal = setInt(&rc.ExecutionConcurrencyLimit, val)
	case HTTPLogEnabledKey:
		retVal = setBool(&rc.HTTPLogEnabled, val)
	case HTTPMaxResultsKey:
		retVal = setInt(&rc.HTTPMaxResults, val)
	case HTTPPAgeLimitKey:
		retVal = setInt(&rc.HTTPPageLimit, val)
	case HTTPProxyHostKey:
		rc.HTTPProxyHost = val
	case HTTPProxyPasswordKey:
		rc.HTTPProxyPassword = val
	case HTTPProxyPortKey:
		retVal = setInt(&rc.HTTPProxyPort, val)
	case IndirectDepthMaxKey:
		retVal = setInt(&rc.IndirectDepthMax, val)
	case HTTPProxySchemeKey:
		rc.HTTPProxyScheme = val
	case HTTPProxyUserKey:
		rc.HTTPProxyUser = val
	case InfilePathKey:
		rc.InfilePath = val
	case LogLevelStrKey:
		rc.LogLevelStr = val
	case NamespaceCfgRawKey:
		rc.NamespaceCfgRaw = val
	case StoreTxnCfgRawKey:
		rc.StoreTxnCfgRaw = val
	case GCCfgRawKey:
		rc.GCCfgRaw = val
	case OutfilePathKey:
		rc.OutfilePath = val
	case OutputFormatKey:
		rc.OutputFormat = val
	case ApplicationFilesRootPathKey:
		rc.ApplicationFilesRootPath = val
	case ApplicationFilesRootPathModeKey:
		retVal = setUint32(&rc.ApplicationFilesRootPathMode, val)
	case PgSrvAddressKey:
		rc.PGSrvAddress = val
	case PgSrvLogLevelKey:
		rc.PGSrvLogLevel = val
	case PgSrvPortKey:
		retVal = setInt(&rc.PGSrvPort, val)
	case PgSrvRawTLSCfgKey:
		rc.PGSrvRawTLSCfg = val
	case QueryCacheSizeKey:
		retVal = setInt(&rc.QueryCacheSize, val)
	case RegistryRawKey:
		rc.RegistryRaw = val
	case TemplateCtxFilePathKey:
		rc.TemplateCtxFilePath = val
	case TestWithoutApiCallsKey:
		retVal = setBool(&rc.TestWithoutApiCalls, val)
	case UseNonPreferredAPIsKEy:
		retVal = setBool(&rc.UseNonPreferredAPIs, val)
	case VerboseFlagKey:
		retVal = setBool(&rc.VerboseFlag, val)
	case ViperCfgFileNameKey:
		rc.ViperCfgFileName = val
	case WorkOfflineKey:
		retVal = setBool(&rc.WorkOffline, val)
	}
	return retVal
}
