package dto

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/jeroenrinzema/psql-wire/pkg/sqldata"
)

const (
	AuthApiKeyStr                   string = "api_key"
	AuthAWSSigningv4Str             string = "aws_signing_v4"
	AuthBasicStr                    string = "basic"
	AuthInteractiveStr              string = "interactive"
	AuthServiceAccountStr           string = "service_account"
	AuthNullStr                     string = "null_auth"
	DarkColorScheme                 string = "dark"
	LightColorScheme                string = "light"
	NullColorScheme                 string = "null"
	DefaultColorScheme              string = DarkColorScheme
	DefaultWindowsColorScheme       string = NullColorScheme
	DryRunFlagKey                   string = "dryrun"
	AuthCtxKey                      string = "auth"
	APIRequestTimeoutKey            string = "apirequesttimeout"
	CacheKeyCountKey                string = "cachekeycount"
	CacheTTLKey                     string = "metadatattl"
	ColorSchemeKey                  string = "colorscheme"
	ConfigFilePathKey               string = "configfile"
	CPUProfileKey                   string = "cpuprofile"
	CSVHeadersDisableKey            string = "hideheaders"
	DbEngineKey                     string = "dbengine"
	DbGenerationIdKey               string = "dbgenerationid"
	DbFilePathKey                   string = "dbfilepath"
	DbInitFilePathKey               string = "dbinitfilepath"
	DelimiterKey                    string = "delimiter"
	ErrorPresentationKey            string = "errorpresentation"
	HTTPLogEnabledKey               string = "http.log.enabled"
	HTTPMaxResultsKey               string = "http.response.maxResults"
	HTTPProxyHostKey                string = "http.proxy.host"
	HTTPProxyPasswordKey            string = "http.proxy.password"
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
	ReinitKey                       string = "reinit"
	TemplateCtxFilePathKey          string = "iqldata"
	TestWithoutApiCallsKey          string = "testwithoutapicalls"
	UseNonPreferredAPIsKEy          string = "usenonpreferredapis"
	VerboseFlagKey                  string = "verbose"
	ViperCfgFileNameKey             string = "viperconfigfilename"
	WorkOfflineKey                  string = "offline"
)

type KeyVal struct {
	K string
	V []byte
}

type BackendMessages struct {
	WorkingMessages []string
}

type HTTPElementType int

const (
	QueryParam HTTPElementType = iota
	PathParam
	Header
	BodyAttribute
)

type HTTPElement struct {
	Type HTTPElementType
	Name string
}

type AuthCtx struct {
	Scopes      []string `json:"scopes,omitempty" yaml:"scopes,omitempty"`
	Type        string   `json:"type" yaml:"type"`
	ValuePrefix string   `json:"valuePrefix" yaml:"valuePrefix"`
	ID          string   `json:"-" yaml:"-"`
	KeyID       string   `json:"keyID" yaml:"keyID"`
	KeyFilePath string   `json:"credentialsfilepath" yaml:"credentialsfilepath"`
	KeyEnvVar   string   `json:"credentialsenvvar" yaml:"credentialsenvvar"`
	Active      bool     `json:"-" yaml:"-"`
}

func (ac *AuthCtx) HasKey() bool {
	if ac.KeyFilePath != "" || ac.KeyEnvVar != "" {
		return true
	}
	return false
}

func (ac *AuthCtx) InferAuthType(authTypeRequested string) string {
	ft := strings.ToLower(authTypeRequested)
	switch ft {
	case AuthApiKeyStr:
		return AuthApiKeyStr
	case AuthServiceAccountStr:
		return AuthServiceAccountStr
	case AuthInteractiveStr:
		return AuthInteractiveStr
	}
	if ac.KeyFilePath != "" || ac.KeyEnvVar != "" {
		return AuthServiceAccountStr
	}
	return AuthInteractiveStr
}

func (ac *AuthCtx) GetCredentialsBytes() ([]byte, error) {
	if ac.KeyEnvVar != "" {
		rv := os.Getenv(ac.KeyEnvVar)
		if rv == "" {
			return nil, fmt.Errorf("credentialsenvvar references empty string")
		}
		return []byte(rv), nil
	}
	credentialFile := ac.KeyFilePath
	return ioutil.ReadFile(credentialFile)
}

func (ac *AuthCtx) GetCredentialsSourceDescriptorString() string {
	if ac.KeyEnvVar != "" {
		return fmt.Sprintf("credentialsenvvar:%s", ac.KeyEnvVar)
	}
	return fmt.Sprintf("credentialsfilepath:%s", ac.KeyFilePath)
}

type ExecPayload struct {
	Payload    []byte
	Header     map[string][]string
	PayloadMap map[string]interface{}
}

func inferKeyFileType(keyFileType string) string {
	if keyFileType == "" {
		return AuthServiceAccountStr
	}
	return keyFileType
}

func GetAuthCtx(scopes []string, keyFilePath string, keyFileType string) *AuthCtx {
	var authType string
	if keyFilePath == "" {
		authType = AuthInteractiveStr
	} else {
		authType = inferKeyFileType(keyFileType)
	}
	return &AuthCtx{
		Scopes:      scopes,
		Type:        authType,
		KeyFilePath: keyFilePath,
		Active:      false,
	}
}

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
	DbEngine                     string
	DbFilePath                   string
	DbGenerationId               int
	DbInitFilePath               string
	Delimiter                    string
	DryRunFlag                   bool
	ErrorPresentation            string
	HTTPLogEnabled               bool
	HTTPMaxResults               int
	HTTPProxyHost                string
	HTTPProxyPassword            string
	HTTPProxyPort                int
	HTTPProxyScheme              string
	HTTPProxyUser                string
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
	Reinit                       bool
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
	case DbEngineKey:
		rc.DbEngine = val
	case DbFilePathKey:
		rc.DbFilePath = val
	case DbGenerationIdKey:
		retVal = setInt(&rc.DbGenerationId, val)
	case DbInitFilePathKey:
		rc.DbInitFilePath = val
	case DelimiterKey:
		rc.Delimiter = val
	case DryRunFlagKey:
		retVal = setBool(&rc.DryRunFlag, val)
	case ErrorPresentationKey:
		rc.ErrorPresentation = val
	case HTTPLogEnabledKey:
		retVal = setBool(&rc.HTTPLogEnabled, val)
	case HTTPMaxResultsKey:
		retVal = setInt(&rc.HTTPMaxResults, val)
	case HTTPProxyHostKey:
		rc.HTTPProxyHost = val
	case HTTPProxyPasswordKey:
		rc.HTTPProxyPassword = val
	case HTTPProxyPortKey:
		retVal = setInt(&rc.HTTPProxyPort, val)
	case HTTPProxySchemeKey:
		rc.HTTPProxyScheme = val
	case HTTPProxyUserKey:
		rc.HTTPProxyUser = val
	case InfilePathKey:
		rc.InfilePath = val
	case LogLevelStrKey:
		rc.LogLevelStr = val
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
	case ReinitKey:
		retVal = setBool(&rc.Reinit, val)
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

type RowsDTO struct {
	RowMap      map[string]map[string]interface{}
	ColumnOrder []string
	Err         error
	RowSort     func(map[string]map[string]interface{}) []string
}

type OutputContext struct {
	RuntimeContext RuntimeCtx
	Result         sqldata.ISQLResultStream
}

type PrepareResultSetDTO struct {
	OutputBody  map[string]interface{}
	Msg         *BackendMessages
	RawRows     map[int]map[int]interface{}
	RowMap      map[string]map[string]interface{}
	ColumnOrder []string
	RowSort     func(map[string]map[string]interface{}) []string
	Err         error
}

func NewPrepareResultSetDTO(
	body map[string]interface{},
	rowMap map[string]map[string]interface{},
	columnOrder []string,
	rowSort func(map[string]map[string]interface{}) []string,
	err error,
	msg *BackendMessages,
) PrepareResultSetDTO {
	return PrepareResultSetDTO{
		OutputBody:  body,
		RowMap:      rowMap,
		ColumnOrder: columnOrder,
		RowSort:     rowSort,
		Err:         err,
		Msg:         msg,
		RawRows:     map[int]map[int]interface{}{},
	}
}

func NewPrepareResultSetPlusRawDTO(
	body map[string]interface{},
	rowMap map[string]map[string]interface{},
	columnOrder []string,
	rowSort func(map[string]map[string]interface{}) []string,
	err error,
	msg *BackendMessages,
	rawRows map[int]map[int]interface{},
) PrepareResultSetDTO {
	return PrepareResultSetDTO{
		OutputBody:  body,
		RowMap:      rowMap,
		ColumnOrder: columnOrder,
		RowSort:     rowSort,
		Err:         err,
		Msg:         msg,
		RawRows:     rawRows,
	}
}

type ExecutorOutput struct {
	GetSQLResult  func() sqldata.ISQLResultStream
	GetRawResult  func() map[int]map[int]interface{}
	GetOutputBody func() map[string]interface{}
	Msg           *BackendMessages
	Err           error
}

func (ex ExecutorOutput) ResultToMap() (map[int]map[int]interface{}, error) {
	return ex.GetRawResult(), nil
}

func NewExecutorOutput(result sqldata.ISQLResultStream, body map[string]interface{}, rawResult map[int]map[int]interface{}, msg *BackendMessages, err error) ExecutorOutput {
	return newExecutorOutput(result, body, rawResult, msg, err)
}

func newExecutorOutput(result sqldata.ISQLResultStream, body map[string]interface{}, rawResult map[int]map[int]interface{}, msg *BackendMessages, err error) ExecutorOutput {
	return ExecutorOutput{
		GetSQLResult: func() sqldata.ISQLResultStream { return result },
		GetRawResult: func() map[int]map[int]interface{} {
			if rawResult == nil {
				return make(map[int]map[int]interface{})
			}
			return rawResult
		},
		GetOutputBody: func() map[string]interface{} { return body },
		Msg:           msg,
		Err:           err,
	}
}

func NewErroneousExecutorOutput(err error) ExecutorOutput {
	return newExecutorOutput(nil, nil, nil, nil, err)
}

type BasicPrimitiveContext struct {
	body      map[string]interface{}
	authCtx   func(string) (*AuthCtx, error)
	writer    io.Writer
	errWriter io.Writer
}

func NewBasicPrimitiveContext(authCtx func(string) (*AuthCtx, error), writer io.Writer, errWriter io.Writer) *BasicPrimitiveContext {
	return &BasicPrimitiveContext{
		authCtx:   authCtx,
		writer:    writer,
		errWriter: errWriter,
	}
}

func (bpp *BasicPrimitiveContext) GetAuthContext(prov string) (*AuthCtx, error) {
	return bpp.authCtx(prov)
}

func (bpp *BasicPrimitiveContext) GetWriter() io.Writer {
	return bpp.writer
}

func (bpp *BasicPrimitiveContext) GetErrWriter() io.Writer {
	return bpp.errWriter
}
