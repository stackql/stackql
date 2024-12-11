package dto

import "strconv"

type RuntimeCtx struct {
	APIRequestTimeout            int
	AuthRaw                      string
	CABundle                     string
	AllowInsecure                bool
	CacheKeyCount                int
	CacheTTL                     int
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
	DataflowComponentsMax        int
	DataflowDependencyMax        int
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
	ExportAlias                  string
	ProviderStr                  string
	RegistryRaw                  string
	SessionCtxRaw                string
	SQLBackendCfgRaw             string
	DBInternalCfgRaw             string
	NamespaceCfgRaw              string
	StoreTxnCfgRaw               string
	GCCfgRaw                     string
	ACIDCfgRaw                   string
	QueryCacheSize               int
	TemplateCtxFilePath          string
	TestWithoutAPICalls          bool
	UseNonPreferredAPIs          bool
	VarList                      []string
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

//nolint:funlen,gocyclo,cyclop // pretty much boileplate
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
	case ColorSchemeKey: // deprecated
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
	case DataflowComponentsMaxKey:
		retVal = setInt(&rc.DataflowComponentsMax, val)
	case DataflowDependencyMaxKey:
		retVal = setInt(&rc.DataflowDependencyMax, val)
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
	case ACIDCfgRawKey:
		rc.ACIDCfgRaw = val
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
	case ExportAliasKey:
		rc.ExportAlias = val
	case PgSrvPortKey:
		retVal = setInt(&rc.PGSrvPort, val)
	case PgSrvRawTLSCfgKey:
		rc.PGSrvRawTLSCfg = val
	case QueryCacheSizeKey:
		retVal = setInt(&rc.QueryCacheSize, val)
	case RegistryRawKey:
		rc.RegistryRaw = val
	case SessionCtxKey:
		rc.SessionCtxRaw = val
	case TemplateCtxFilePathKey:
		rc.TemplateCtxFilePath = val
	case TestWithoutAPICallsKey:
		retVal = setBool(&rc.TestWithoutAPICalls, val)
	case UseNonPreferredAPIsKEy:
		retVal = setBool(&rc.UseNonPreferredAPIs, val)
	case VarListKey:
		rc.VarList = append(rc.VarList, val)
	case VerboseFlagKey:
		retVal = setBool(&rc.VerboseFlag, val)
	case ViperCfgFileNameKey:
		rc.ViperCfgFileName = val
	case WorkOfflineKey:
		retVal = setBool(&rc.WorkOffline, val)
	}
	return retVal
}

func (rc RuntimeCtx) Copy() RuntimeCtx {
	return RuntimeCtx{
		APIRequestTimeout:            rc.APIRequestTimeout,
		AuthRaw:                      rc.AuthRaw,
		CABundle:                     rc.CABundle,
		AllowInsecure:                rc.AllowInsecure,
		CacheKeyCount:                rc.CacheKeyCount,
		CacheTTL:                     rc.CacheTTL,
		ConfigFilePath:               rc.ConfigFilePath,
		CPUProfile:                   rc.CPUProfile,
		CSVHeadersDisable:            rc.CSVHeadersDisable,
		Delimiter:                    rc.Delimiter,
		DryRunFlag:                   rc.DryRunFlag,
		ErrorPresentation:            rc.ErrorPresentation,
		ExecutionConcurrencyLimit:    rc.ExecutionConcurrencyLimit,
		HTTPLogEnabled:               rc.HTTPLogEnabled,
		HTTPMaxResults:               rc.HTTPMaxResults,
		HTTPPageLimit:                rc.HTTPPageLimit,
		HTTPProxyHost:                rc.HTTPProxyHost,
		HTTPProxyPassword:            rc.HTTPProxyPassword,
		HTTPProxyPort:                rc.HTTPProxyPort,
		HTTPProxyScheme:              rc.HTTPProxyScheme,
		HTTPProxyUser:                rc.HTTPProxyUser,
		IndirectDepthMax:             rc.IndirectDepthMax,
		DataflowComponentsMax:        rc.DataflowComponentsMax,
		DataflowDependencyMax:        rc.DataflowDependencyMax,
		InfilePath:                   rc.InfilePath,
		LogLevelStr:                  rc.LogLevelStr,
		OutfilePath:                  rc.OutfilePath,
		OutputFormat:                 rc.OutputFormat,
		ApplicationFilesRootPath:     rc.ApplicationFilesRootPath,
		ApplicationFilesRootPathMode: rc.ApplicationFilesRootPathMode,
		PGSrvAddress:                 rc.PGSrvAddress,
		PGSrvLogLevel:                rc.PGSrvLogLevel,
		PGSrvPort:                    rc.PGSrvPort,
		PGSrvRawTLSCfg:               rc.PGSrvRawTLSCfg,
		ExportAlias:                  rc.ExportAlias,
		ProviderStr:                  rc.ProviderStr,
		RegistryRaw:                  rc.RegistryRaw,
		SessionCtxRaw:                rc.SessionCtxRaw,
		SQLBackendCfgRaw:             rc.SQLBackendCfgRaw,
		DBInternalCfgRaw:             rc.DBInternalCfgRaw,
		NamespaceCfgRaw:              rc.NamespaceCfgRaw,
		StoreTxnCfgRaw:               rc.StoreTxnCfgRaw,
		GCCfgRaw:                     rc.GCCfgRaw,
		ACIDCfgRaw:                   rc.ACIDCfgRaw,
		QueryCacheSize:               rc.QueryCacheSize,
		TemplateCtxFilePath:          rc.TemplateCtxFilePath,
		TestWithoutAPICalls:          rc.TestWithoutAPICalls,
		UseNonPreferredAPIs:          rc.UseNonPreferredAPIs,
		VarList:                      rc.VarList,
		VerboseFlag:                  rc.VerboseFlag,
		ViperCfgFileName:             rc.ViperCfgFileName,
		WorkOffline:                  rc.WorkOffline,
	}
}
