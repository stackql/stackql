package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/stackql/any-sdk/pkg/constants"
	"github.com/stackql/any-sdk/pkg/db/sqlcontrol"
	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/any-sdk/pkg/jsonpath"
	"github.com/stackql/any-sdk/pkg/netutils"
	"github.com/stackql/any-sdk/pkg/nomenclature"
	"github.com/stackql/any-sdk/public/formulation"
	"github.com/stackql/any-sdk/public/sqlengine"
	"github.com/stackql/stackql/internal/stackql/acid/tsm"
	"github.com/stackql/stackql/internal/stackql/acid/txn_context"
	"github.com/stackql/stackql/internal/stackql/bundle"
	"github.com/stackql/stackql/internal/stackql/datasource/sql_datasource"
	"github.com/stackql/stackql/internal/stackql/dbmsinternal"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/garbagecollector"
	"github.com/stackql/stackql/internal/stackql/kstore"
	"github.com/stackql/stackql/internal/stackql/provider"
	"github.com/stackql/stackql/internal/stackql/sql_system"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
	"github.com/stackql/stackql/internal/stackql/typing"
	"github.com/stackql/stackql/internal/stackql/writer"
	"github.com/stackql/stackql/pkg/txncounter"

	lrucache "github.com/stackql/stackql-parser/go/cache"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"gopkg.in/yaml.v2"
)

var (
	_ HandlerContext = &standardHandlerContext{}
)

type HandlerContext interface { //nolint:revive // don't mind stuttering this one
	Clone() HandlerContext
	//
	GetStacqklSemver() string
	GetASTFormatter() sqlparser.NodeFormatter
	GetAuthContext(providerName string) (*dto.AuthCtx, error)
	GetDBMSInternalRouter() dbmsinternal.Router
	GetProvider(providerName string) (provider.IProvider, error)
	DeleteProvider(providerName string) error
	GetSupportedProviders(extended bool) (map[string]map[string]interface{}, error)
	LogHTTPResponseMap(target interface{})
	//
	GetRawQuery() string
	GetQuery() string
	GetRuntimeContext() dto.RuntimeCtx
	GetProviders() map[string]provider.IProvider
	GetControlAttributes() sqlcontrol.ControlAttributes
	GetCurrentProvider() string
	GetAuthContexts() dto.AuthContexts
	GetRegistry() formulation.RegistryAPI
	GetErrorPresentation() string
	GetOutfile() io.Writer
	GetOutErrFile() io.Writer
	GetLRUCache() *lrucache.LRUCache
	GetSQLDataSource(name string) (sql_datasource.SQLDataSource, bool)
	GetSQLEngine() sqlengine.SQLEngine
	GetSQLSystem() sql_system.SQLSystem
	GetPersistenceSystem() formulation.PersistenceSystem
	GetGarbageCollector() garbagecollector.GarbageCollector
	GetDrmConfig() drm.Config
	SetTxnCounterMgr(txncounter.Manager)
	GetTxnCounterMgr() txncounter.Manager
	GetTxnStore() kstore.KStore
	GetNamespaceCollection() tablenamespace.Collection
	GetFormatter() sqlparser.NodeFormatter
	GetPGInternalRouter() dbmsinternal.Router
	//
	SetCurrentProvider(string)
	SetQuery(string)
	SetRawQuery(string)
	//
	SetOutErrFile(io.Writer)
	//
	GetTxnCoordinatorCtx() txn_context.ITransactionCoordinatorContext

	GetTypingConfig() typing.Config
	GetIsolationLevel() constants.IsolationLevel
	UpdateIsolationLevel(isolationLevelStr string) error

	GetRollbackType() constants.RollbackType
	UpdateRollbackType(rollbackTypeStr string) error

	GetTSM() (tsm.TSM, bool)
	SetTSM(tsm.TSM)

	SetExportNamespace(string)
	GetExportNamespace() string

	GetDataFlowCfg() dto.DataFlowCfg

	//
	SetConfigAtPath(path string, rhs interface{}, scope string) error

	// for testing only
	SetDefaultHTTPClient(client *http.Client)
}

type standardHandlerContext struct {
	// mutex to protect auth context map
	// must be a pointer dure to clonability
	authMapMutex        *sync.Mutex
	sessionCtxMutex     *sync.Mutex
	providersMapMutex   *sync.Mutex
	rawQuery            string
	query               string
	runtimeContext      dto.RuntimeCtx
	providers           map[string]provider.IProvider
	controlAttributes   sqlcontrol.ControlAttributes
	currentProvider     string
	authContexts        dto.AuthContexts
	sqlDataSources      map[string]sql_datasource.SQLDataSource
	registry            formulation.RegistryAPI
	errorPresentation   string
	outfile             io.Writer
	outErrFile          io.Writer
	lRUCache            *lrucache.LRUCache
	sqlEngine           sqlengine.SQLEngine
	sqlSystem           sql_system.SQLSystem
	persistenceSystem   formulation.PersistenceSystem
	garbageCollector    garbagecollector.GarbageCollector
	drmConfig           drm.Config
	txnCounterMgr       txncounter.Manager
	txnStore            kstore.KStore
	namespaceCollection tablenamespace.Collection
	formatter           sqlparser.NodeFormatter
	pgInternalRouter    dbmsinternal.Router
	txnCoordinatorCtx   txn_context.ITransactionCoordinatorContext
	typCfg              typing.Config
	sessionContext      dto.SessionContext
	walInstance         tsm.TSM
	exportNamespace     string
	stackqlSemver       string
	defaultHTTPClient   *http.Client
}

// for testing only.
func (hc *standardHandlerContext) SetDefaultHTTPClient(client *http.Client) {
	hc.defaultHTTPClient = client
}

func (hc *standardHandlerContext) GetDataFlowCfg() dto.DataFlowCfg {
	return dto.NewDataFlowCfg(
		hc.runtimeContext.DataflowDependencyMax,
		hc.runtimeContext.DataflowComponentsMax,
	)
}

func (hc *standardHandlerContext) GetStacqklSemver() string {
	return hc.stackqlSemver
}

func (hc *standardHandlerContext) SetExportNamespace(exportNamespace string) {
	hc.exportNamespace = exportNamespace
}

func (hc *standardHandlerContext) GetExportNamespace() string {
	return hc.exportNamespace
}

func (hc *standardHandlerContext) GetTSM() (tsm.TSM, bool) {
	return hc.walInstance, hc.walInstance != nil
}

func (hc *standardHandlerContext) SetTSM(w tsm.TSM) {
	hc.walInstance = w
}

func (hc *standardHandlerContext) GetTxnCoordinatorCtx() txn_context.ITransactionCoordinatorContext {
	return hc.txnCoordinatorCtx
}

func (hc *standardHandlerContext) GetTypingConfig() typing.Config {
	return hc.typCfg
}

func (hc *standardHandlerContext) SetCurrentProvider(p string) {
	hc.currentProvider = p
}

func (hc *standardHandlerContext) SetRawQuery(rq string) {
	hc.rawQuery = rq
}

func (hc *standardHandlerContext) SetQuery(q string) {
	hc.query = q
}

func (hc *standardHandlerContext) SetTxnCounterMgr(mgr txncounter.Manager) {
	hc.txnCounterMgr = mgr
}

func (hc *standardHandlerContext) GetRawQuery() string                         { return hc.rawQuery }
func (hc *standardHandlerContext) GetQuery() string                            { return hc.query }
func (hc *standardHandlerContext) GetRuntimeContext() dto.RuntimeCtx           { return hc.runtimeContext }
func (hc *standardHandlerContext) GetProviders() map[string]provider.IProvider { return hc.providers }
func (hc *standardHandlerContext) GetControlAttributes() sqlcontrol.ControlAttributes {
	return hc.controlAttributes
}
func (hc *standardHandlerContext) GetCurrentProvider() string { return hc.currentProvider }

func (hc *standardHandlerContext) GetAuthContexts() dto.AuthContexts {
	hc.authMapMutex.Lock()
	defer hc.authMapMutex.Unlock()
	return hc.authContexts
}

func (hc *standardHandlerContext) GetRegistry() formulation.RegistryAPI { return hc.registry }
func (hc *standardHandlerContext) GetErrorPresentation() string         { return hc.errorPresentation }
func (hc *standardHandlerContext) GetOutfile() io.Writer                { return hc.outfile }
func (hc *standardHandlerContext) GetOutErrFile() io.Writer {
	defer hc.sessionCtxMutex.Unlock()
	hc.sessionCtxMutex.Lock()
	return hc.outErrFile
}
func (hc *standardHandlerContext) SetOutErrFile(w io.Writer) {
	defer hc.sessionCtxMutex.Unlock()
	hc.sessionCtxMutex.Lock()
	hc.outErrFile = w
}
func (hc *standardHandlerContext) GetLRUCache() *lrucache.LRUCache    { return hc.lRUCache }
func (hc *standardHandlerContext) GetSQLEngine() sqlengine.SQLEngine  { return hc.sqlEngine }
func (hc *standardHandlerContext) GetSQLSystem() sql_system.SQLSystem { return hc.sqlSystem }
func (hc *standardHandlerContext) GetPersistenceSystem() formulation.PersistenceSystem {
	return hc.persistenceSystem
}
func (hc *standardHandlerContext) GetGarbageCollector() garbagecollector.GarbageCollector {
	return hc.garbageCollector
}
func (hc *standardHandlerContext) GetDrmConfig() drm.Config { return hc.drmConfig }
func (hc *standardHandlerContext) GetTxnCounterMgr() txncounter.Manager {
	return hc.txnCounterMgr
}
func (hc *standardHandlerContext) GetTxnStore() kstore.KStore { return hc.txnStore }

//	func (hc *standardHandlerContext) GetNamespaceCollection() tablenamespace.Collection {
//		return hc.namespaceCollection
//	}
func (hc *standardHandlerContext) GetFormatter() sqlparser.NodeFormatter { return hc.formatter }
func (hc *standardHandlerContext) GetPGInternalRouter() dbmsinternal.Router {
	return hc.pgInternalRouter
}

func getProviderMap(providerName string, providerDesc formulation.ProviderDescription) map[string]interface{} {
	latestVersion, err := providerDesc.GetLatestVersion()
	if err != nil {
		//nolint:lll // long message
		latestVersion = fmt.Sprintf("could not infer latest version due to error.  Suggested action is that you wipe the local provider directory.  Error =  '%s'", err.Error())
	}
	googleMap := map[string]interface{}{
		"name":    providerName,
		"version": latestVersion,
	}
	return googleMap
}

func getProviderMapExtended(
	providerName string,
	providerDesc formulation.ProviderDescription,
) map[string]interface{} {
	return getProviderMap(providerName, providerDesc)
}

func (hc *standardHandlerContext) GetSQLDataSource(name string) (sql_datasource.SQLDataSource, bool) {
	ac, ok := hc.sqlDataSources[name]
	return ac, ok
}

func (hc *standardHandlerContext) GetSupportedProviders(extended bool) (map[string]map[string]interface{}, error) {
	retVal := make(map[string]map[string]interface{})
	provs := hc.registry.ListLocallyAvailableProviders()
	// Supporting SQL data sources
	// These will be overwritten by any documented providers with the same name
	for k := range hc.sqlDataSources {
		pn := k
		retVal[pn] = map[string]interface{}{
			"name": pn,
		}
	}
	for k, pd := range provs {
		pn := k
		if pn == "googleapis.com" {
			pn = "google"
		}
		if extended {
			retVal[pn] = getProviderMapExtended(pn, pd)
		} else {
			retVal[pn] = getProviderMap(pn, pd)
		}
	}
	return retVal, nil
}

func (hc *standardHandlerContext) GetASTFormatter() sqlparser.NodeFormatter {
	return hc.formatter
}

func (hc *standardHandlerContext) DeleteProvider(providerName string) error {
	hc.providersMapMutex.Lock()
	defer hc.providersMapMutex.Unlock()
	if providerName == "googleapis.com" {
		providerName = "google"
	}
	if providerName == "" {
		return fmt.Errorf("provider name cannot be empty")
	}
	_, ok := hc.providers[providerName]
	if len(hc.providers) == 0 || !ok {
		return fmt.Errorf("provider '%s' not found", providerName)
	}
	delete(hc.providers, providerName)
	return nil
}

func (hc *standardHandlerContext) GetProvider(providerName string) (provider.IProvider, error) {
	var err error
	if providerName == "" {
		providerName = hc.runtimeContext.ProviderStr
	}
	if hc.namespaceCollection.GetAnalyticsCacheTableNamespaceConfigurator().IsAllowed(providerName) {
		providerName = hc.namespaceCollection.GetAnalyticsCacheTableNamespaceConfigurator().GetObjectName(providerName)
	}
	ds, err := nomenclature.ExtractProviderDesignation(providerName)
	if err != nil {
		return nil, err
	}
	// NOTE: this method **must** be the only place the providers map is accessed or mutated
	hc.providersMapMutex.Lock()
	defer hc.providersMapMutex.Unlock()
	prov, ok := hc.providers[providerName]
	//nolint:nestif // TODO: review
	if !ok {
		prov, err = provider.GenerateProvider(
			hc.runtimeContext, ds.Name, ds.Tag,
			hc.registry, hc.sqlSystem, hc.persistenceSystem,
			hc.defaultHTTPClient,
		)
		if err == nil {
			hc.providers[providerName] = prov
			// update auth info with provider default if auth not already present
			pr, prErr := prov.GetProvider()
			if prErr != nil {
				return prov, prErr
			}
			authDTO, exists := pr.GetAuth()
			if exists {
				auth := transformOpenapiStackqlAuthToLocal(authDTO)
				hc.updateAuthContextIfNotExists(providerName, auth)
			}
			return prov, nil
		}
		err = fmt.Errorf("cannot find provider = '%s': %w", providerName, err)
	}
	return prov, err
}

func (hc *standardHandlerContext) LogHTTPResponseMap(target interface{}) {
	if target == nil {
		//nolint:errcheck // ignore error on output stream
		hc.outErrFile.Write(
			[]byte("processed http response body not present\n"))
		return
	}
	//nolint:nestif // ignore nested if
	if hc.runtimeContext.HTTPLogEnabled {
		switch target := target.(type) {
		case []map[string]interface{}, map[string]interface{}, []interface{}:
			b, err := json.MarshalIndent(target, "", "  ")
			if err != nil {
				//nolint:errcheck // ignore error on output stream
				hc.outErrFile.Write(
					[]byte(
						fmt.Sprintf(
							"processed http response body map '%v' colud not be marshalled; error: %s\n",
							target, err.Error())))
				return
			}
			if target != nil { //nolint:govet // TODO: review
				//nolint:errcheck // ignore error on output stream
				hc.outErrFile.Write(
					[]byte(fmt.Sprintf("processed http response body object: %s\n", string(b))))
			} else {
				//nolint:errcheck // ignore error on output stream
				hc.outErrFile.Write(
					[]byte("processed http response body not present\n"))
			}
		default:
			if target != nil { //nolint:govet // TODO: review
				//nolint:errcheck // ignore error on output stream
				hc.outErrFile.Write(
					[]byte(
						fmt.Sprintf("processed http response body object: %v\n", target)))
			} else {
				//nolint:errcheck // ignore error on output stream
				hc.outErrFile.Write(
					[]byte("processed http response body not present\n"))
			}
		}
	}
}

func (hc *standardHandlerContext) GetAuthContext(providerName string) (*dto.AuthCtx, error) {
	var err error
	if providerName == "" {
		providerName = hc.runtimeContext.ProviderStr
	}
	hc.authMapMutex.Lock()
	defer hc.authMapMutex.Unlock()
	authCtx, ok := hc.authContexts[providerName]
	if !ok {
		err = fmt.Errorf("cannot find AUTH context for provider = '%s'", providerName)
	}
	return authCtx, err
}

func (hc *standardHandlerContext) UpdateIsolationLevel(isolationLevelStr string) error {
	hc.sessionCtxMutex.Lock()
	defer hc.sessionCtxMutex.Unlock()
	return hc.sessionContext.UpdateIsolationLevel(isolationLevelStr)
}

func (hc *standardHandlerContext) GetIsolationLevel() constants.IsolationLevel {
	hc.sessionCtxMutex.Lock()
	defer hc.sessionCtxMutex.Unlock()
	return hc.sessionContext.GetIsolationLevel()
}

func (hc *standardHandlerContext) UpdateRollbackType(rollbackTypeStr string) error {
	hc.sessionCtxMutex.Lock()
	defer hc.sessionCtxMutex.Unlock()
	return hc.sessionContext.UpdateIsolationLevel(rollbackTypeStr)
}

func (hc *standardHandlerContext) GetRollbackType() constants.RollbackType {
	hc.sessionCtxMutex.Lock()
	defer hc.sessionCtxMutex.Unlock()
	return hc.sessionContext.GetRollbackType()
}

func (hc *standardHandlerContext) updateAuthContextIfNotExists(providerName string, authCtx *dto.AuthCtx) {
	hc.authMapMutex.Lock()
	defer hc.authMapMutex.Unlock()
	_, alreadyExists := hc.authContexts[providerName]
	if alreadyExists {
		return
	}
	hc.authContexts[providerName] = authCtx
}

func (hc *standardHandlerContext) SetConfigAtPath(path string, rhs interface{}, scope string) error {
	return hc.setConfigAtPath(path, rhs, scope)
}

func (hc *standardHandlerContext) setConfigAtPath(path string, rhs interface{}, scope string) error {
	searchPath, searchPathErr := composeSystemSearchPath(path)
	if searchPathErr != nil {
		return searchPathErr
	}
	system := searchPath.GetSystem()
	remainder := searchPath.GetRemainder()
	switch system {
	case dto.AuthCtxKey:
		return hc.setAuthContextAtPath(remainder, rhs, scope)
	default:
		return fmt.Errorf("system '%s' not supported", system)
	}
}

func (hc *standardHandlerContext) setAuthContextAtPath(path string, rhs interface{}, scope string) error {
	hc.authMapMutex.Lock()
	defer hc.authMapMutex.Unlock()
	// TODO: review
	if scope == "local" {
		hc.authContexts = hc.authContexts.Clone()
	}
	searchPath, searchPathErr := composeSystemSearchPath(path)
	if searchPathErr != nil {
		return searchPathErr
	}
	authCtx := hc.authContexts[searchPath.GetSystem()]
	rv := jsonpath.Set(authCtx, searchPath.GetRemainder(), rhs)
	return rv
}

func (hc *standardHandlerContext) GetNamespaceCollection() tablenamespace.Collection {
	return hc.namespaceCollection
}

func (hc *standardHandlerContext) GetDBMSInternalRouter() dbmsinternal.Router {
	return hc.pgInternalRouter
}

func GetRegistry(runtimeCtx dto.RuntimeCtx) (formulation.RegistryAPI, error) {
	return getRegistry(runtimeCtx)
}

func getRegistry(runtimeCtx dto.RuntimeCtx) (formulation.RegistryAPI, error) {
	var rc formulation.RegistryConfig
	err := yaml.Unmarshal([]byte(runtimeCtx.RegistryRaw), &rc)
	if err != nil {
		return nil, err
	}
	if rc.LocalDocRoot == "" {
		if strings.HasPrefix(rc.RegistryURL, "file:") {
			rc.LocalDocRoot = path.Clean(path.Join(strings.TrimPrefix(rc.RegistryURL, "file:"), ".."))
		} else {
			rc.LocalDocRoot = runtimeCtx.ApplicationFilesRootPath
		}
	}
	rt := netutils.GetRoundTripper(runtimeCtx, nil)
	return formulation.NewRegistry(rc, rt)
}

func (hc *standardHandlerContext) Clone() HandlerContext {
	hc.sessionCtxMutex.Lock()
	defer hc.sessionCtxMutex.Unlock()
	rv := standardHandlerContext{
		authMapMutex:        hc.authMapMutex,
		sessionCtxMutex:     hc.sessionCtxMutex,
		providersMapMutex:   hc.providersMapMutex,
		drmConfig:           hc.drmConfig,
		rawQuery:            hc.rawQuery,
		runtimeContext:      hc.runtimeContext,
		currentProvider:     hc.currentProvider,
		providers:           hc.providers,
		authContexts:        hc.authContexts,
		registry:            hc.registry,
		controlAttributes:   hc.controlAttributes,
		errorPresentation:   hc.errorPresentation,
		lRUCache:            hc.lRUCache,
		sqlEngine:           hc.sqlEngine,
		sqlDataSources:      hc.sqlDataSources,
		sqlSystem:           hc.sqlSystem,
		persistenceSystem:   hc.persistenceSystem,
		garbageCollector:    hc.garbageCollector,
		outErrFile:          hc.outErrFile,
		outfile:             hc.outfile,
		txnCounterMgr:       hc.txnCounterMgr,
		txnStore:            hc.txnStore,
		namespaceCollection: hc.namespaceCollection,
		formatter:           hc.formatter,
		pgInternalRouter:    hc.pgInternalRouter,
		txnCoordinatorCtx:   hc.txnCoordinatorCtx,
		typCfg:              hc.typCfg,
		sessionContext:      hc.sessionContext,
		exportNamespace:     hc.exportNamespace,
		stackqlSemver:       hc.stackqlSemver,
		defaultHTTPClient:   hc.defaultHTTPClient,
	}
	return &rv
}

func getOutputFile(filename string) (*os.File, error) {
	switch filename {
	case "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	default:
		return os.Create(filename)
	}
}

func NewHandlerCtx(
	cmdString string,
	runtimeCtx dto.RuntimeCtx,
	lruCache *lrucache.LRUCache,
	inputBundle bundle.Bundle,
	stackqlSemver string,
) (HandlerContext, error) {
	reg, err := getRegistry(runtimeCtx)
	if err != nil {
		return nil, err
	}
	providers := make(map[string]provider.IProvider)
	controlAttributes := inputBundle.GetControlAttributes()
	sqlEngine := inputBundle.GetSQLEngine()
	outWriter, isOutwriter := inputBundle.GetStdOut()
	outErrWriter, isOutErrWriter := inputBundle.GetStdErr()
	var fileErr error
	if !isOutwriter {
		outFilePath := runtimeCtx.OutfilePath
		if outFilePath == "" {
			outFilePath = "stdout"
		}
		outWriter, fileErr = getOutputFile(outFilePath)
		if fileErr != nil {
			return nil, fileErr
		}
	}
	if !isOutErrWriter {
		outErrPath := writer.StdErrStr
		outErrWriter, fileErr = getOutputFile(outErrPath)
		if fileErr != nil {
			return nil, fileErr
		}
	}
	rv := standardHandlerContext{
		authMapMutex:        &sync.Mutex{},
		sessionCtxMutex:     &sync.Mutex{},
		providersMapMutex:   &sync.Mutex{},
		rawQuery:            cmdString,
		runtimeContext:      runtimeCtx.Copy(),
		providers:           providers,
		authContexts:        inputBundle.GetAuthContexts(),
		registry:            reg,
		controlAttributes:   controlAttributes,
		errorPresentation:   runtimeCtx.ErrorPresentation,
		lRUCache:            lruCache,
		sqlEngine:           sqlEngine,
		sqlDataSources:      inputBundle.GetSQLDataSources(),
		sqlSystem:           inputBundle.GetSQLSystem(),
		persistenceSystem:   inputBundle.GetPersistenceSystem(),
		garbageCollector:    inputBundle.GetGC(),
		txnCounterMgr:       inputBundle.GetTxnCounterManager(),
		txnStore:            inputBundle.GetTxnStore(),
		namespaceCollection: inputBundle.GetNamespaceCollection(),
		formatter:           inputBundle.GetSQLSystem().GetASTFormatter(),
		pgInternalRouter:    inputBundle.GetDBMSInternalRouter(),
		currentProvider:     runtimeCtx.ProviderStr,
		txnCoordinatorCtx:   inputBundle.GetTxnCoordinatorContext(),
		typCfg:              inputBundle.GetTypingConfig(),
		sessionContext:      inputBundle.GetSessionContext(),
		exportNamespace:     runtimeCtx.ExportAlias,
		outfile:             outWriter,
		outErrFile:          outErrWriter,
		stackqlSemver:       stackqlSemver,
	}
	drmCfg, err := drm.GenerateDRMConfig(
		inputBundle.GetSQLSystem(),
		inputBundle.GetPersistenceSystem(),
		inputBundle.GetTypingConfig(),
		rv.namespaceCollection,
		controlAttributes,
	)
	if err != nil {
		return nil, err
	}
	rv.drmConfig = drmCfg
	return &rv, nil
}

func transformOpenapiStackqlAuthToLocal(authDTO formulation.AuthDTO) *dto.AuthCtx {
	rv := &dto.AuthCtx{
		Scopes:                  authDTO.GetScopes(),
		Subject:                 authDTO.GetSubject(),
		Type:                    authDTO.GetType(),
		ValuePrefix:             authDTO.GetValuePrefix(),
		KeyID:                   authDTO.GetKeyID(),
		KeyIDEnvVar:             authDTO.GetKeyIDEnvVar(),
		KeyFilePath:             authDTO.GetKeyFilePath(),
		KeyFilePathEnvVar:       authDTO.GetKeyFilePathEnvVar(),
		KeyEnvVar:               authDTO.GetKeyEnvVar(),
		EnvVarAPIKeyStr:         authDTO.GetEnvVarAPIKeyStr(),
		EnvVarAPISecretStr:      authDTO.GetEnvVarAPISecretStr(),
		EnvVarUsername:          authDTO.GetEnvVarUsername(),
		EnvVarPassword:          authDTO.GetEnvVarPassword(),
		EncodedBasicCredentials: authDTO.GetInlineBasicCredentials(),
		Location:                authDTO.GetLocation(),
		Name:                    authDTO.GetName(),
		TokenURL:                authDTO.GetTokenURL(),
		GrantType:               authDTO.GetGrantType(),
		ClientID:                authDTO.GetClientID(),
		ClientSecret:            authDTO.GetClientSecret(),
		ClientIDEnvVar:          authDTO.GetClientIDEnvVar(),
		ClientSecretEnvVar:      authDTO.GetClientSecretEnvVar(),
		Values:                  authDTO.GetValues(),
		AuthStyle:               authDTO.GetAuthStyle(),
		AccountID:               authDTO.GetAccountID(),
		AccoountIDEnvVar:        authDTO.GetAccountIDEnvVar(),
	}
	successor, successorExists := authDTO.GetSuccessor()
	currentParent := rv
	for {
		if successorExists {
			transformedSuccessor := transformOpenapiStackqlAuthToLocal(successor)
			currentParent.Successor = transformedSuccessor
			currentParent = transformedSuccessor
			successor, successorExists = successor.GetSuccessor()
		} else {
			break
		}
	}
	return rv
}
