package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/stackql/any-sdk/pkg/client"
	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/any-sdk/pkg/httpelement"
	"github.com/stackql/any-sdk/pkg/local_template_executor"
	"github.com/stackql/any-sdk/pkg/logging"
	"github.com/stackql/any-sdk/pkg/stream_transform"
	"github.com/stackql/any-sdk/pkg/streaming"

	"github.com/stackql/any-sdk/pkg/providerinvoker"
	"github.com/stackql/any-sdk/public/providerinvokers/anysdkhttp"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/acid/binlog"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/tableinsertioncontainer"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"github.com/stackql/stackql/internal/stackql/util"

	sdk_internal_dto "github.com/stackql/any-sdk/pkg/internaldto"
	"github.com/stackql/any-sdk/public/formulation"
)

var (
	MonitorPollIntervalSeconds int = 10 //nolint:revive,gochecknoglobals // TODO: global vars refactor
)

//nolint:gochecknoglobals // TODO: global vars refactor
var (
	_                  MonoValentExecutorFactory = (*monoValentExecution)(nil)
	nilElisionFunction                           = func(string, ...any) bool {
		return false
	}
)

type MonoValentExecutorFactory interface {
	GetExecutor() (func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput, error)
}

type MonitorMonoValentExecutorFactory interface {
	GetMonitorExecutor(
		provider formulation.Provider,
		op formulation.OperationStore,
		precursor primitive.IPrimitive,
		initialCtx primitive.IPrimitiveCtx,
		comments sqlparser.CommentDirectives,
	) (func(pc primitive.IPrimitiveCtx, bd interface{}) internaldto.ExecutorOutput, error)
}

// monoValentExecution implements the Builder interface
// and represents the action of acquiring data from an endpoint
// and then persisting that data into a table.
// This data would then subsequently be queried by later execution phases.

//nolint:unused // TODO: refactor
type monoValentExecution struct {
	invoker                    providerinvoker.Invoker
	graphHolder                primitivegraph.PrimitiveGraphHolder
	handlerCtx                 handler.HandlerContext
	tableMeta                  tablemetadata.ExtendedTableMetadata
	addressSpace               formulation.AddressSpace
	drmCfg                     drm.Config
	insertPreparedStatementCtx drm.PreparedStatementCtx
	insertionContainer         tableinsertioncontainer.TableInsertionContainer
	txnCtrlCtr                 internaldto.TxnControlCounters
	rowSort                    func(map[string]map[string]interface{}) []string
	root                       primitivegraph.PrimitiveNode
	stream                     streaming.MapStream
	isReadOnly                 bool //nolint:unused // TODO: build out
	isSkipResponse             bool
	isMutation                 bool
	isAwait                    bool
	defaultHTTPClient          *http.Client // for testing purposes only
}

func NewMonoValentExecutorFactory(
	graphHolder primitivegraph.PrimitiveGraphHolder,
	handlerCtx handler.HandlerContext,
	tableMeta tablemetadata.ExtendedTableMetadata,
	insertCtx drm.PreparedStatementCtx,
	insertionContainer tableinsertioncontainer.TableInsertionContainer,
	rowSort func(map[string]map[string]interface{}) []string,
	stream streaming.MapStream,
	isSkipResponse bool,
	isMutation bool,
	isAwait bool,
) MonoValentExecutorFactory {
	var tcc internaldto.TxnControlCounters
	if insertCtx != nil {
		tcc = insertCtx.GetGCCtrlCtrs()
	}
	if stream == nil {
		stream = streaming.NewNopMapStream()
	}
	var defaultHTTPClient *http.Client
	if tableMeta != nil {
		prov, provErr := tableMeta.GetProvider()
		if provErr == nil && prov != nil {
			defaultHTTPClient = prov.GetDefaultHTTPClient()
		}
	}
	return &monoValentExecution{
		graphHolder:                graphHolder,
		handlerCtx:                 handlerCtx,
		tableMeta:                  tableMeta,
		rowSort:                    rowSort,
		drmCfg:                     handlerCtx.GetDrmConfig(),
		insertPreparedStatementCtx: insertCtx,
		insertionContainer:         insertionContainer,
		txnCtrlCtr:                 tcc,
		stream:                     stream,
		isSkipResponse:             isSkipResponse,
		isMutation:                 isMutation,
		isAwait:                    isAwait,
		defaultHTTPClient:          defaultHTTPClient,
		invoker:                    anysdkhttp.New(),
	}
}

type standardMethodElider struct {
	elisionFunc func(string, ...any) bool
}

func (sme *standardMethodElider) IsElide(reqEncoding string, argz ...any) bool {
	return sme.elisionFunc(reqEncoding, argz...)
}

func NewNilMethodElider() methodElider {
	return &standardMethodElider{
		elisionFunc: nilElisionFunction,
	}
}

func NewStandardMethodElider(elisionFunc func(string, ...any) bool) methodElider {
	return &standardMethodElider{
		elisionFunc: elisionFunc,
	}
}

//nolint:lll // chaining
func (mv *monoValentExecution) elideActionIfPossible(
	currentTcc internaldto.TxnControlCounters,
	tableName string,
	_ string, // request endocidng placeholder
) methodElider {
	elisionFunc := func(reqEncoding string, _ ...any) bool {
		olderTcc, isMatch := mv.handlerCtx.GetNamespaceCollection().GetAnalyticsCacheTableNamespaceConfigurator().Match(
			tableName,
			reqEncoding,
			mv.drmCfg.GetControlAttributes().GetControlLatestUpdateColumnName(), mv.drmCfg.GetControlAttributes().GetControlInsertEncodedIDColumnName())
		if isMatch {
			nonControlColumns := mv.insertPreparedStatementCtx.GetNonControlColumns()
			var nonControlColumnNames []string
			for _, c := range nonControlColumns {
				nonControlColumnNames = append(nonControlColumnNames, c.GetName())
			}
			//nolint:errcheck // TODO: fix
			mv.handlerCtx.GetGarbageCollector().Update(
				tableName,
				olderTcc.Clone(),
				currentTcc,
			)
			//nolint:errcheck // TODO: fix
			mv.insertionContainer.SetTableTxnCounters(tableName, olderTcc)
			mv.insertPreparedStatementCtx.SetGCCtrlCtrs(olderTcc)
			r, sqlErr := mv.handlerCtx.GetNamespaceCollection().GetAnalyticsCacheTableNamespaceConfigurator().Read(
				tableName, reqEncoding,
				mv.drmCfg.GetControlAttributes().GetControlInsertEncodedIDColumnName(),
				nonControlColumnNames)
			if sqlErr != nil {
				internaldto.NewErroneousExecutorOutput(sqlErr)
			}
			mv.drmCfg.ExtractObjectFromSQLRows(r, nonControlColumns, mv.stream)
			return true
		}
		return false
	}
	return NewStandardMethodElider(elisionFunc)
}

type methodElider interface {
	IsElide(string, ...any) bool
}

type actionInsertResult struct {
	err                error
	isHousekeepingDone bool
}

type ActionInsertResult interface {
	GetError() (error, bool)
	IsHousekeepingDone() bool
}

//nolint:revive // no idea why this is a thing
func (air *actionInsertResult) GetError() (error, bool) {
	return air.err, air.err != nil
}

func (air *actionInsertResult) IsHousekeepingDone() bool {
	return air.isHousekeepingDone
}

func newActionInsertResult(isHousekeepingDone bool, err error) ActionInsertResult {
	return &actionInsertResult{
		err:                err,
		isHousekeepingDone: isHousekeepingDone,
	}
}

type itemsDTO struct {
	items             interface{}
	ok                bool
	isNilPayload      bool
	singletonResponse map[string]interface{}
}

type ItemisationResult interface {
	GetItems() (interface{}, bool)
	GetSingltetonResponse() (map[string]interface{}, bool)
	IsOk() bool
	IsNilPayload() bool
}

func (id *itemsDTO) GetItems() (interface{}, bool) {
	return id.items, id.items != nil
}

func (id *itemsDTO) GetSingltetonResponse() (map[string]interface{}, bool) {
	return id.singletonResponse, id.singletonResponse != nil
}

func (id *itemsDTO) IsOk() bool {
	return id.ok
}

func (id *itemsDTO) IsNilPayload() bool {
	return id.isNilPayload
}

func newItemisationResult(
	items interface{},
	ok bool,
	isNilPayload bool,
	singletonResponse map[string]interface{},
) ItemisationResult {
	return &itemsDTO{
		items:             items,
		ok:                ok,
		isNilPayload:      isNilPayload,
		singletonResponse: singletonResponse,
	}
}

//nolint:nestif // apathy
func itemise(
	target interface{},
	resErr error,
	selectItemsKey string,
) ItemisationResult {
	var items interface{}
	var ok bool
	var singletonResponse map[string]interface{}
	logging.GetLogger().Infoln(fmt.Sprintf("monoValentExecution.Execute() target = %v", target))
	switch pl := target.(type) {
	// add case for xml object,
	case map[string]interface{}:
		singletonResponse = pl
		if selectItemsKey != "" && selectItemsKey != "/*" {
			items, ok = pl[selectItemsKey]
			if !ok {
				if resErr != nil {
					items = []interface{}{}
					ok = true
				} else {
					items = []interface{}{
						pl,
					}
					ok = true
				}
			}
		} else {
			items = []interface{}{
				pl,
			}
			ok = true
		}
	case []interface{}:
		items = pl
		ok = true
	case []map[string]interface{}:
		items = pl
		ok = true
	case nil:
		return newItemisationResult(nil, false, true, singletonResponse)
	}
	return newItemisationResult(items, ok, false, singletonResponse)
}

func inferNextPageResponseElement(provider formulation.Provider, method formulation.OperationStore) sdk_internal_dto.HTTPElement {
	st, ok := method.GetPaginationResponseTokenSemantic()
	if ok {
		if tp, err := sdk_internal_dto.ExtractHTTPElement(st.GetLocation()); err == nil {
			rv := sdk_internal_dto.NewHTTPElement(
				tp,
				st.GetKey(),
			)
			transformer, tErr := st.GetTransformer()
			if tErr == nil && transformer != nil {
				rv.SetTransformer(transformer)
			}
			return rv
		}
	}
	providerStr := provider.GetName()
	switch providerStr {
	case "github", "okta":
		rv := sdk_internal_dto.NewHTTPElement(
			sdk_internal_dto.Header,
			"Link",
		)
		rv.SetTransformer(formulation.DefaultLinkHeaderTransformer)
		return rv
	default:
		return sdk_internal_dto.NewHTTPElement(
			sdk_internal_dto.BodyAttribute,
			"nextPageToken",
		)
	}
}

func inferNextPageRequestElement(provider formulation.Provider, method formulation.OperationStore) sdk_internal_dto.HTTPElement {
	st, ok := method.GetPaginationRequestTokenSemantic()
	if ok {
		if tp, err := sdk_internal_dto.ExtractHTTPElement(st.GetLocation()); err == nil {
			rv := sdk_internal_dto.NewHTTPElement(
				tp,
				st.GetKey(),
			)
			transformer, tErr := st.GetTransformer()
			if tErr == nil && transformer != nil {
				rv.SetTransformer(transformer)
			}
			return rv
		}
	}
	providerStr := provider.GetName()
	switch providerStr {
	case "github", "okta":
		return sdk_internal_dto.NewHTTPElement(
			sdk_internal_dto.RequestString,
			"",
		)
	default:
		return sdk_internal_dto.NewHTTPElement(
			sdk_internal_dto.QueryParam,
			"pageToken",
		)
	}
}

type PagingState interface {
	GetPageCount() int
	IsFinished() bool
	GetHTTPResponse() (*http.Response, error)
	GetAPIError() error
}

type httpPagingState struct {
	pageCount  int
	isFinished bool
	response   client.AnySdkResponse
	apiErr     error
}

func (hps *httpPagingState) GetPageCount() int {
	return hps.pageCount
}

func (hps *httpPagingState) IsFinished() bool {
	return hps.isFinished
}

func (hps *httpPagingState) GetHTTPResponse() (*http.Response, error) {
	if hps.response != nil {
		return hps.response.GetHttpResponse()
	}
	return nil, fmt.Errorf("nil http response in paging state")
}

func (hps *httpPagingState) GetAPIError() error {
	return hps.apiErr
}

func newPagingState(
	pageCount int,
	isFinished bool,
	response client.AnySdkResponse,
	apiErr error,
) PagingState {
	return &httpPagingState{
		pageCount:  pageCount,
		isFinished: isFinished,
		response:   response,
		apiErr:     apiErr,
	}
}

func page(
	res formulation.Response,
	method formulation.OperationStore,
	provider formulation.Provider,
	reqCtx formulation.HTTPArmouryParameters,
	pageCount int,
	rtCtx dto.RuntimeCtx,
	authCtx *dto.AuthCtx,
	outErrFile io.Writer,
	defaultHTTPClient *http.Client,
) PagingState {
	npt := inferNextPageResponseElement(provider, method)
	nptRequest := inferNextPageRequestElement(provider, method)
	if npt == nil || nptRequest == nil {
		return newPagingState(pageCount, true, nil, nil)
	}
	tk := extractNextPageToken(res, npt)
	if tk == "" || tk == "<nil>" || tk == "[]" || (rtCtx.HTTPPageLimit > 0 && pageCount >= rtCtx.HTTPPageLimit) {
		return newPagingState(pageCount, true, nil, nil)
	}
	pageCount++
	req, reqErr := reqCtx.SetNextPage(method, tk, nptRequest)
	if reqErr != nil {
		return newPagingState(pageCount, true, nil, reqErr)
	}
	cc := formulation.NewAnySdkClientConfigurator(rtCtx, provider.GetName(), defaultHTTPClient)
	response, apiErr := formulation.CallFromSignature(
		cc, rtCtx, authCtx, authCtx.Type, false, outErrFile, provider,
		formulation.NewAnySdkOpStoreDesignation(method),
		formulation.NewwHTTPAnySdkArgList(req), // TODO: abstract
	)
	return newPagingState(pageCount, false, response, apiErr)
}

type ActionInsertPayload interface {
	GetItemisationResult() formulation.ItemisationResult
	IsHousekeepingDone() bool
	GetTableName() string
	GetParamsUsed() map[string]interface{}
	GetReqEncoding() string
}

func newHTTPActionInsertPayload(
	itemisationResult formulation.ItemisationResult,
	housekeepingDone bool,
	tableName string,
	paramsUsed map[string]interface{},
	reqEncoding string,
) formulation.ActionInsertPayload {
	return formulation.NewActionInsertPayload(
		itemisationResult,
		housekeepingDone,
		tableName,
		paramsUsed,
		reqEncoding,
	)
}

//nolint:nestif,gocognit // acceptable for now
func (mv *monoValentExecution) ActionInsertPreparation(
	payload formulation.ActionInsertPayload,
) formulation.ActionInsertResult {
	itemisationResult := payload.GetItemisationResult()
	housekeepingDone := payload.IsHousekeepingDone()
	tableName := payload.GetTableName()
	paramsUsed := payload.GetParamsUsed()
	reqEncoding := payload.GetReqEncoding()

	items, _ := itemisationResult.GetItems()
	keys := make(map[string]map[string]interface{})
	iArr, iErr := castItemsArray(items)
	if iErr != nil {
		return newActionInsertResult(housekeepingDone, iErr)
	}
	streamErr := mv.stream.Write(iArr)
	if streamErr != nil {
		return newActionInsertResult(housekeepingDone, streamErr)
	}
	if len(iArr) > 0 {
		if !housekeepingDone && mv.insertPreparedStatementCtx != nil {
			_, execErr := mv.handlerCtx.GetSQLEngine().Exec(mv.insertPreparedStatementCtx.GetGCHousekeepingQueries())
			tcc := mv.insertPreparedStatementCtx.GetGCCtrlCtrs()
			tcc.SetTableName(tableName)
			//nolint:errcheck // TODO: fix
			mv.insertionContainer.SetTableTxnCounters(tableName, tcc)
			housekeepingDone = true
			if execErr != nil {
				return newActionInsertResult(housekeepingDone, execErr)
			}
		}

		for i, item := range iArr {
			if item != nil {
				if len(paramsUsed) > 0 {
					for k, v := range paramsUsed {
						if _, itemOk := item[k]; !itemOk {
							item[k] = v
						}
					}
				}

				logging.GetLogger().Infoln(
					fmt.Sprintf(
						"running insert with query = '''%s''', control parameters: %v",
						mv.insertPreparedStatementCtx.GetQuery(),
						mv.insertPreparedStatementCtx.GetGCCtrlCtrs(),
					),
				)
				r, rErr := mv.drmCfg.ExecuteInsertDML(
					mv.handlerCtx.GetSQLEngine(),
					mv.insertPreparedStatementCtx,
					item,
					reqEncoding,
				)
				logging.GetLogger().Infoln(
					fmt.Sprintf(
						"insert result = %v, error = %v",
						r,
						rErr,
					),
				)
				if rErr != nil {
					expandedErr := fmt.Errorf(
						"sql insert error: '%w' from query: %s",
						rErr,
						mv.insertPreparedStatementCtx.GetQuery(),
					)
					return newActionInsertResult(housekeepingDone, expandedErr)
				}
				keys[strconv.Itoa(i)] = item
			}
		}
	}

	return newActionInsertResult(housekeepingDone, nil)
}

type AgnosticatePayload interface {
	GetArmoury() (formulation.HTTPArmoury, error)
	GetProvider() formulation.Provider
	GetMethod() formulation.OperationStore
	GetTableName() string
	GetAuthContext() *dto.AuthCtx
	GetRuntimeCtx() dto.RuntimeCtx
	GetOutErrFile() io.Writer
	GetMaxResultsElement() sdk_internal_dto.HTTPElement
	GetElider() methodElider
	IsNilResponseAcceptable() bool
	GetPolyHandler() PolyHandler
	GetSelectItemsKey() string
	GetInsertPreparator() formulation.InsertPreparator
	IsSkipResponse() bool
	IsMutation() bool
	IsAwait() bool
	GetDefaultHTTPClient() *http.Client // testing purposes only
}

type httpAgnosticatePayload struct {
	tableMeta               tablemetadata.ExtendedTableMetadata
	provider                formulation.Provider
	method                  formulation.OperationStore
	tableName               string
	authCtx                 *dto.AuthCtx
	rtCtx                   dto.RuntimeCtx
	outErrFile              io.Writer
	maxResultsElement       sdk_internal_dto.HTTPElement
	elider                  methodElider
	isNilResponseAcceptable bool
	polyHandler             PolyHandler
	selectItemsKey          string
	insertPreparator        formulation.InsertPreparator
	isSkipResponse          bool
	isMutation              bool
	isAwait                 bool
	defaultHTTPClient       *http.Client // testing purposes only
}

func newHTTPAgnosticatePayload(
	tableMeta tablemetadata.ExtendedTableMetadata,
	provider formulation.Provider,
	method formulation.OperationStore,
	tableName string,
	authCtx *dto.AuthCtx,
	rtCtx dto.RuntimeCtx,
	outErrFile io.Writer,
	maxResultsElement sdk_internal_dto.HTTPElement,
	elider methodElider,
	isNilResponseAcceptable bool,
	polyHandler PolyHandler,
	selectItemsKey string,
	insertPreparator formulation.InsertPreparator,
	isSkipResponse bool,
	isMutation bool,
	isAwait bool,
	defaultHTTPClient *http.Client,
) AgnosticatePayload {
	return &httpAgnosticatePayload{
		tableMeta:               tableMeta,
		provider:                provider,
		method:                  method,
		tableName:               tableName,
		authCtx:                 authCtx,
		rtCtx:                   rtCtx,
		outErrFile:              outErrFile,
		maxResultsElement:       maxResultsElement,
		elider:                  elider,
		isNilResponseAcceptable: isNilResponseAcceptable,
		polyHandler:             polyHandler,
		selectItemsKey:          selectItemsKey,
		insertPreparator:        insertPreparator,
		isSkipResponse:          isSkipResponse,
		isMutation:              isMutation,
		isAwait:                 isAwait,
		defaultHTTPClient:       defaultHTTPClient,
	}
}

func (ap *httpAgnosticatePayload) GetPolyHandler() PolyHandler {
	return ap.polyHandler
}

func (ap *httpAgnosticatePayload) GetDefaultHTTPClient() *http.Client {
	return ap.defaultHTTPClient
}

func (ap *httpAgnosticatePayload) IsAwait() bool {
	return ap.isAwait
}

func (ap *httpAgnosticatePayload) GetInsertPreparator() formulation.InsertPreparator {
	return ap.insertPreparator
}

func (ap *httpAgnosticatePayload) GetSelectItemsKey() string {
	return ap.selectItemsKey
}

func (ap *httpAgnosticatePayload) GetArmoury() (formulation.HTTPArmoury, error) {
	return ap.tableMeta.GetHTTPArmoury()
}

func (ap *httpAgnosticatePayload) GetProvider() formulation.Provider {
	return ap.provider
}

func (ap *httpAgnosticatePayload) IsMutation() bool {
	return ap.isMutation
}

func (ap *httpAgnosticatePayload) IsSkipResponse() bool {
	return ap.isSkipResponse
}

func (ap *httpAgnosticatePayload) GetMethod() formulation.OperationStore {
	return ap.method
}

func (ap *httpAgnosticatePayload) GetTableName() string {
	return ap.tableName
}

func (ap *httpAgnosticatePayload) GetAuthContext() *dto.AuthCtx {
	return ap.authCtx
}

func (ap *httpAgnosticatePayload) GetRuntimeCtx() dto.RuntimeCtx {
	return ap.rtCtx
}

func (ap *httpAgnosticatePayload) GetOutErrFile() io.Writer {
	return ap.outErrFile
}

func (ap *httpAgnosticatePayload) GetMaxResultsElement() sdk_internal_dto.HTTPElement {
	return ap.maxResultsElement
}

func (ap *httpAgnosticatePayload) GetElider() methodElider {
	return ap.elider
}

func (ap *httpAgnosticatePayload) IsNilResponseAcceptable() bool {
	return ap.isNilResponseAcceptable
}

type PolyHandler interface {
	LogHTTPResponseMap(target interface{})
	MessageHandler([]string)
	GetMessages() []string
}

type standardPolyHandler struct {
	handlerCtx handler.HandlerContext
	messages   []string
}

func (sph *standardPolyHandler) LogHTTPResponseMap(target interface{}) {
	sph.handlerCtx.LogHTTPResponseMap(target)
}

func (sph *standardPolyHandler) MessageHandler(messages []string) {
	sph.messages = append(sph.messages, messages...)
}

func (sph *standardPolyHandler) GetMessages() []string {
	return sph.messages
}

func NewStandardPolyHandler(handlerCtx handler.HandlerContext) PolyHandler {
	return &standardPolyHandler{
		handlerCtx: handlerCtx,
		messages:   []string{},
	}
}

func agnosticate(
	agPayload AgnosticatePayload,
) (processorResponse, error) {
	outErrFile := agPayload.GetOutErrFile()
	runtimeCtx := agPayload.GetRuntimeCtx()
	provider := agPayload.GetProvider()
	tableName := agPayload.GetTableName()
	authCtx := agPayload.GetAuthContext()
	method := agPayload.GetMethod()
	mr := agPayload.GetMaxResultsElement()
	elider := agPayload.GetElider()
	polyHandler := agPayload.GetPolyHandler()
	selectItemsKey := agPayload.GetSelectItemsKey()
	insertPreparator := agPayload.GetInsertPreparator()
	isSkipResponse := agPayload.IsNilResponseAcceptable()
	isMutation := agPayload.IsMutation()
	isAwait := agPayload.IsAwait()
	defaultHTTPClient := agPayload.GetDefaultHTTPClient()
	// TODO: TCC setup
	armoury, armouryErr := agPayload.GetArmoury()
	if armouryErr != nil {
		//nolint:errcheck // TODO: fix
		outErrFile.Write([]byte(
			fmt.Sprintf(
				"error assembling http aspects for resource '%s': %s\n",
				method.GetResource().GetID(),
				armouryErr.Error(),
			),
		),
		)
		return nil, armouryErr
	}
	if mr != nil {
		// TODO: infer param position and act accordingly
		ok := true
		if ok && runtimeCtx.HTTPMaxResults > 0 {
			passOverParams := armoury.GetRequestParams()
			for i, p := range passOverParams {
				param := p
				q := param.GetQuery()
				q.Set("maxResults", strconv.Itoa(runtimeCtx.HTTPMaxResults))
				param.SetRawQuery(q.Encode())
				passOverParams[i] = param
			}
			armoury.SetRequestParams(passOverParams)
		}
	}
	reqParams := armoury.GetRequestParams()
	logging.GetLogger().Infof("monoValentExecution.Execute() req param count = %d", len(reqParams))
	var processorResponse processorResponse
	for _, rc := range reqParams {
		rq := rc
		processor := NewProcessor(
			NewProcessorPayload(
				rq,
				elider,
				provider,
				method,
				tableName,
				runtimeCtx,
				authCtx,
				outErrFile,
				polyHandler,
				selectItemsKey,
				insertPreparator,
				isSkipResponse,
				false,
				isAwait,
				false,
				isMutation,
				"",
				defaultHTTPClient,
			),
		)
		processorResponse = processor.Process()
		if processorResponse != nil && processorResponse.GetError() != nil {
			return processorResponse, processorResponse.GetError()
		}
	}
	return processorResponse, nil
}

type ProcessorPayload interface {
	GetArmouryParams() formulation.HTTPArmouryParameters
	GetElider() methodElider
	GetProvider() formulation.Provider
	GetMethod() formulation.OperationStore
	GetTableName() string
	GetRuntimeCtx() dto.RuntimeCtx
	GetAuthCtx() *dto.AuthCtx
	GetOutErrFile() io.Writer
	GetPolyHandler() PolyHandler
	GetSelectItemsKey() string
	GetInsertPreparator() formulation.InsertPreparator
	IsSkipResponse() bool
	IsMutation() bool
	IsMaterialiseResponse() bool
	IsAwait() bool
	IsReverseRequired() bool
	GetVerb() string
	GetDefaultHTTPClient() *http.Client // testing purposes only
}

func NewProcessorPayload(
	armouryParams formulation.HTTPArmouryParameters,
	elider methodElider,
	provider formulation.Provider,
	method formulation.OperationStore,
	tableName string,
	runtimeCtx dto.RuntimeCtx,
	authCtx *dto.AuthCtx,
	outErrFile io.Writer,
	polyHandler PolyHandler,
	selectItemsKey string,
	insertPreparator formulation.InsertPreparator,
	isSkipResponse bool,
	isMaterialiseResponse bool,
	isAwait bool,
	isReverseRequired bool,
	isMutation bool,
	verb string,
	defaultHTTPClient *http.Client,
) ProcessorPayload {
	return &standardProcessorPayload{
		armouryParams:         armouryParams,
		elider:                elider,
		provider:              provider,
		method:                method,
		tableName:             tableName,
		runtimeCtx:            runtimeCtx,
		authCtx:               authCtx,
		outErrFile:            outErrFile,
		polyHandler:           polyHandler,
		selectItemsKey:        selectItemsKey,
		insertPreparator:      insertPreparator,
		isSkipResponse:        isSkipResponse,
		isMaterialiseResponse: isMaterialiseResponse,
		isAwait:               isAwait,
		isReverseRequired:     isReverseRequired,
		isMutation:            isMutation,
		verb:                  verb,
		defaultHTTPClient:     defaultHTTPClient,
	}
}

type standardProcessorPayload struct {
	armouryParams         formulation.HTTPArmouryParameters
	elider                methodElider
	provider              formulation.Provider
	method                formulation.OperationStore
	tableName             string
	runtimeCtx            dto.RuntimeCtx
	authCtx               *dto.AuthCtx
	outErrFile            io.Writer
	polyHandler           PolyHandler
	selectItemsKey        string
	insertPreparator      formulation.InsertPreparator
	isSkipResponse        bool
	isMaterialiseResponse bool
	isAwait               bool
	isReverseRequired     bool
	isMutation            bool
	verb                  string
	defaultHTTPClient     *http.Client // testing purposes only
}

func (pp *standardProcessorPayload) GetArmouryParams() formulation.HTTPArmouryParameters {
	return pp.armouryParams
}

func (pp *standardProcessorPayload) IsSkipResponse() bool {
	return pp.isSkipResponse
}

func (pp *standardProcessorPayload) GetDefaultHTTPClient() *http.Client {
	return pp.defaultHTTPClient
}

func (pp *standardProcessorPayload) IsAwait() bool {
	return pp.isAwait
}

func (pp *standardProcessorPayload) IsMutation() bool {
	return pp.isMutation
}

func (pp *standardProcessorPayload) IsReverseRequired() bool {
	return pp.isReverseRequired
}

func (pp *standardProcessorPayload) IsMaterialiseResponse() bool {
	return pp.isMaterialiseResponse
}

func (pp *standardProcessorPayload) GetElider() methodElider {
	return pp.elider
}

func (pp *standardProcessorPayload) GetVerb() string {
	if pp.verb == "" {
		return "insert"
	}
	return pp.verb
}

func (pp *standardProcessorPayload) GetProvider() formulation.Provider {
	return pp.provider
}

func (pp *standardProcessorPayload) GetMethod() formulation.OperationStore {
	return pp.method
}

func (pp *standardProcessorPayload) GetTableName() string {
	return pp.tableName
}

func (pp *standardProcessorPayload) GetRuntimeCtx() dto.RuntimeCtx {
	return pp.runtimeCtx
}

func (pp *standardProcessorPayload) GetAuthCtx() *dto.AuthCtx {
	return pp.authCtx
}

func (pp *standardProcessorPayload) GetOutErrFile() io.Writer {
	return pp.outErrFile
}

func (pp *standardProcessorPayload) GetPolyHandler() PolyHandler {
	return pp.polyHandler
}

func (pp *standardProcessorPayload) GetSelectItemsKey() string {
	return pp.selectItemsKey
}

func (pp *standardProcessorPayload) GetInsertPreparator() formulation.InsertPreparator {
	return pp.insertPreparator
}

type processorResponse interface {
	GetError() error
	GetSingletonBody() map[string]interface{}
	WithSuccessMessages([]string) processorResponse
	GetSuccessMessages() []string
	AppendReversal(rev formulation.HTTPPreparator)
	GetReversalStream() formulation.HttpPreparatorStream
	IsFailed() bool
	GetFailedMessage() string
}

type httpProcessorResponse struct {
	body            map[string]interface{}
	err             error
	successMessages []string
	reversalStream  formulation.HttpPreparatorStream
	isFailed        bool
	failedMessage   string
}

func (hpr *httpProcessorResponse) IsFailed() bool {
	return hpr.isFailed
}

func (hpr *httpProcessorResponse) GetFailedMessage() string {
	return hpr.failedMessage
}

func (hpr *httpProcessorResponse) WithSuccessMessages(messages []string) processorResponse {
	hpr.successMessages = messages
	return hpr
}

//nolint:errcheck // acceptable for now
func (hpr *httpProcessorResponse) AppendReversal(rev formulation.HTTPPreparator) {
	hpr.reversalStream.Write(rev)
}

func (hpr *httpProcessorResponse) GetReversalStream() formulation.HttpPreparatorStream {
	return hpr.reversalStream
}

func (hpr *httpProcessorResponse) GetSuccessMessages() []string {
	return hpr.successMessages
}

func (hpr *httpProcessorResponse) GetError() error {
	return hpr.err
}

func (hpr *httpProcessorResponse) GetSingletonBody() map[string]interface{} {
	return hpr.body
}

func newHTTPProcessorResponse(
	body map[string]interface{},
	reversalStream formulation.HttpPreparatorStream,
	isFailed bool,
	err error,
) processorResponse {
	return &httpProcessorResponse{
		body:           body,
		err:            err,
		reversalStream: reversalStream,
		isFailed:       isFailed,
	}
}

type Processor interface {
	Process() processorResponse
}

type standardProcessor struct {
	payload           ProcessorPayload
	defaultHTTPClient *http.Client
}

func NewProcessor(payload ProcessorPayload) Processor {
	return &standardProcessor{
		payload:           payload,
		defaultHTTPClient: payload.GetDefaultHTTPClient(),
	}
}

//nolint:funlen,bodyclose,gocognit,gocyclo,cyclop // acceptable for now
func (sp *standardProcessor) Process() processorResponse {
	processorPayload := sp.payload
	armouryParams := processorPayload.GetArmouryParams()
	elider := processorPayload.GetElider()
	provider := processorPayload.GetProvider()
	method := processorPayload.GetMethod()
	tableName := processorPayload.GetTableName()
	runtimeCtx := processorPayload.GetRuntimeCtx()
	authCtx := processorPayload.GetAuthCtx()
	outErrFile := processorPayload.GetOutErrFile()
	polyHandler := processorPayload.GetPolyHandler()
	selectItemsKey := processorPayload.GetSelectItemsKey()
	insertPreparator := processorPayload.GetInsertPreparator()
	isSkipResponse := processorPayload.IsSkipResponse()
	isMutation := processorPayload.IsMutation()
	isMaterialiseResponse := processorPayload.IsMaterialiseResponse()
	isAwait := processorPayload.IsAwait()
	isReverseRequired := processorPayload.IsReverseRequired()
	verb := processorPayload.GetVerb()

	reversalStream := formulation.NewHttpPreparatorStream()

	reqCtx := armouryParams
	paramsUsed, paramErr := reqCtx.ToFlatMap()
	if paramErr != nil {
		return newHTTPProcessorResponse(nil, reversalStream, false, paramErr)
	}
	reqEncoding := reqCtx.Encode()
	elideOk := elider.IsElide(reqEncoding)
	if elideOk {
		return newHTTPProcessorResponse(nil, reversalStream, false, nil)
	}
	// TODO: fix cloning ops
	cc := formulation.NewAnySdkClientConfigurator(runtimeCtx, provider.GetName(), sp.defaultHTTPClient)
	response, apiErr := formulation.CallFromSignature(
		cc,
		runtimeCtx,
		authCtx,
		authCtx.Type,
		false,
		outErrFile,
		provider,
		formulation.NewAnySdkOpStoreDesignation(method),
		reqCtx.GetArgList(),
	)
	if response == nil {
		if apiErr != nil {
			return newHTTPProcessorResponse(nil, reversalStream, false, apiErr)
		}
		return newHTTPProcessorResponse(nil, reversalStream, false, fmt.Errorf("unacceptable nil response from HTTP call"))
	}
	//nolint:govet // ignore for now
	if isSkipResponse && response == nil {
		return newHTTPProcessorResponse(nil, reversalStream, false, nil)
	}
	httpResponse, httpResponseErr := response.GetHttpResponse()
	if httpResponse != nil && httpResponse.Body != nil {
		defer httpResponse.Body.Close()
	}
	if httpResponse != nil && httpResponse.StatusCode >= 400 && isMaterialiseResponse {
		generatedErr := fmt.Errorf("%s over HTTP error: %s", verb, httpResponse.Status)
		return newHTTPProcessorResponse(nil, reversalStream, true, generatedErr)
	}
	// TODO: refactor into package !!TECH_DEBT!!
	housekeepingDone := false
	nptRequest := inferNextPageRequestElement(provider, method)
	pageCount := 1
	for {
		if apiErr != nil {
			return newHTTPProcessorResponse(nil, reversalStream, false, apiErr)
		}
		if httpResponseErr != nil {
			return newHTTPProcessorResponse(nil, reversalStream, false, httpResponseErr)
		}
		// TODO: add async monitor here
		processed, resErr := method.ProcessResponse(httpResponse)
		if resErr != nil {
			if isSkipResponse && isMutation && httpResponse.StatusCode < 300 {
				return newHTTPProcessorResponse(
					nil, reversalStream, false, nil,
				).WithSuccessMessages([]string{"The operation was despatched successfully"})
			}
			//nolint:errcheck // TODO: fix
			outErrFile.Write(
				[]byte(fmt.Sprintf("error processing response: %s\n", resErr.Error())),
			)
			if processed == nil {
				return newHTTPProcessorResponse(nil, reversalStream, false, resErr)
			}
		}
		reversal, reversalExists := processed.GetReversal()
		if reversalExists {
			reversalAppendErr := reversalStream.Write(reversal)
			if reversalAppendErr != nil {
				return newHTTPProcessorResponse(nil, reversalStream, false, reversalAppendErr)
			}
		}
		if !reversalExists && isReverseRequired {
			return newHTTPProcessorResponse(nil, reversalStream, false, resErr)
		}
		res, respOk := processed.GetResponse()
		if !respOk {
			return newHTTPProcessorResponse(nil, reversalStream, false, fmt.Errorf("response is not a valid response"))
		}
		if res.HasError() {
			polyHandler.MessageHandler([]string{res.Error()})
			return newHTTPProcessorResponse(nil, reversalStream, false, nil)
		}
		polyHandler.LogHTTPResponseMap(res.GetProcessedBody())
		logging.GetLogger().Infoln(fmt.Sprintf("monoValentExecution.Execute() response = %v", res))

		if selectItemsKey == "" {
			selectItemsKey = method.GetSelectItemsKey()
		}

		itemisationResult := itemise(res.GetProcessedBody(), resErr, selectItemsKey)

		if itemisationResult.IsNilPayload() {
			break
		}

		singletonResponse, hasSingletonResponse := itemisationResult.GetSingltetonResponse()
		if isMaterialiseResponse {
			msgs := shallowGenerateSuccessMessagesFromHeirarchy(isAwait)
			//nolint:gomnd,mnd // acceptable for now
			if httpResponse.StatusCode < 300 {
				if hasSingletonResponse {
					return newHTTPProcessorResponse(singletonResponse, reversalStream, false, nil).WithSuccessMessages(msgs)
				}
				return newHTTPProcessorResponse(nil, reversalStream, false, nil).WithSuccessMessages(msgs)
			}
			return newHTTPProcessorResponse(nil, reversalStream, false, nil)
		}
		//nolint:gomnd,mnd // acceptable for now
		if httpResponse.StatusCode >= 300 {
			return newHTTPProcessorResponse(nil, reversalStream, false, nil)
		}

		insertPrepResult := insertPreparator.ActionInsertPreparation(
			formulation.NewActionInsertPayload(
				itemisationResult,
				housekeepingDone,
				tableName,
				paramsUsed,
				reqEncoding,
			),
		)
		housekeepingDone = insertPrepResult.IsHousekeepingDone()
		insertPrepErr, hasInsertPrepErr := insertPrepResult.GetError()
		if !isAwait && isSkipResponse && isMutation && httpResponse.StatusCode < 300 {
			return newHTTPProcessorResponse(
				nil, reversalStream, false, nil,
			).WithSuccessMessages([]string{"The operation was despatched successfully"})
		}
		if hasInsertPrepErr {
			return newHTTPProcessorResponse(nil, reversalStream, false, insertPrepErr)
		}

		pageResult := page(
			res,
			method,
			provider,
			reqCtx,
			pageCount,
			runtimeCtx,
			authCtx,
			outErrFile,
			sp.defaultHTTPClient,
		)
		httpResponse, httpResponseErr = pageResult.GetHTTPResponse()
		// if httpResponse != nil && httpResponse.Body != nil {
		// 	defer httpResponse.Body.Close()
		// }
		if httpResponseErr != nil {
			if hasSingletonResponse { // TODO: fix this horrid hack
				return newHTTPProcessorResponse(singletonResponse, reversalStream, false, nil)
			}
			return newHTTPProcessorResponse(nil, reversalStream, false, nil)
			// return internaldto.NewErroneousExecutorOutput(httpResponseErr)
		}

		if pageResult.IsFinished() {
			return newHTTPProcessorResponse(nil, reversalStream, false, nil)
		}

		pageCount = pageResult.GetPageCount()

		apiErr = pageResult.GetAPIError()
	}
	if reqCtx.GetRequest() != nil {
		q := reqCtx.GetRequest().URL.Query()
		q.Del(nptRequest.GetName())
		reqCtx.SetRawQuery(q.Encode())
	}
	return newHTTPProcessorResponse(nil, reversalStream, false, nil)
}

//nolint:revive,nestif,funlen,gocognit // TODO: investigate
func (mv *monoValentExecution) GetExecutor() (func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput, error) {
	prov, err := mv.tableMeta.GetProvider()
	if err != nil {
		return nil, err
	}
	provider, providerErr := prov.GetProvider()
	if providerErr != nil {
		return nil, providerErr
	}
	m, err := mv.tableMeta.GetMethod()
	if err != nil {
		return nil, err
	}
	tableName, err := mv.tableMeta.GetTableName()
	if err != nil {
		return nil, err
	}
	authCtx, authCtxErr := mv.handlerCtx.GetAuthContext(prov.GetProviderString())
	if authCtxErr != nil {
		return nil, authCtxErr
	}
	ex := func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
		currentTcc := mv.insertPreparedStatementCtx.GetGCCtrlCtrs().Clone()
		mv.graphHolder.AddTxnControlCounters(currentTcc)
		mr := prov.InferMaxResultsElement(m)
		polyHandler := NewStandardPolyHandler(
			mv.handlerCtx,
		)
		protocolType, protocolTypeErr := provider.GetProtocolType()
		if protocolTypeErr != nil {
			return internaldto.NewErroneousExecutorOutput(protocolTypeErr)
		}
		//nolint:exhaustive // acceptable for now
		switch protocolType {
		case client.LocalTemplated:
			inlines := m.GetInline()
			if len(inlines) == 0 {
				return internaldto.NewErroneousExecutorOutput(fmt.Errorf("no inlines found"))
			}
			executor := local_template_executor.NewLocalTemplateExecutor(
				inlines[0],
				inlines[1:],
				nil,
			)
			armoury, armouryErr := mv.tableMeta.GetHTTPArmoury()
			if armouryErr != nil {
				return internaldto.NewErroneousExecutorOutput(armouryErr)
			}
			requestParams := armoury.GetRequestParams()
			logging.GetLogger().Infoln(fmt.Sprintf("requestParams = %v", requestParams))
			flatInlineParams := make(map[string]interface{})
			for _, p := range requestParams {
				foundInlineParams, loopErr := p.GetParameters().GetInlineParameterFlatMap()
				if loopErr == nil {
					flatInlineParams = foundInlineParams
				}
				break //nolint:staticcheck // acceptable for now
			}
			// if mapsErr != nil {
			// 	return internaldto.NewErroneousExecutorOutput(mapsErr)
			// }
			// paramMap := interestingMaps.getParameterMap()
			// params := paramMap[0]
			resp, exErr := executor.Execute(
				map[string]any{"parameters": flatInlineParams},
			)
			if exErr != nil {
				return internaldto.NewErroneousExecutorOutput(exErr)
			}
			var backendMessages []string
			stdOut, stdOutExists := resp.GetStdOut()
			var stdoutStr string
			if stdOutExists {
				stdoutStr = stdOut.String()
				expectedResponse, isExpectedResponse := m.GetResponse()
				if isExpectedResponse {
					responseTransform, responseTransformExists := expectedResponse.GetTransform()
					if responseTransformExists {
						input := stdoutStr
						streamTransformerFactory := stream_transform.NewStreamTransformerFactory(
							responseTransform.GetType(),
							responseTransform.GetBody(),
						)
						if !streamTransformerFactory.IsTransformable() {
							return internaldto.NewErroneousExecutorOutput(
								fmt.Errorf("unsupported template type: %s", responseTransform.GetType()),
							)
						}
						tfm, getTfmErr := streamTransformerFactory.GetTransformer(input)
						if getTfmErr != nil {
							return internaldto.NewErroneousExecutorOutput(
								fmt.Errorf("failed to transform: %w", getTfmErr))
						}
						transformError := tfm.Transform()
						if transformError != nil {
							return internaldto.NewErroneousExecutorOutput(
								fmt.Errorf("failed to transform: %w", transformError))
						}
						outStream := tfm.GetOutStream()
						outputBytes, readErr := io.ReadAll(outStream)
						if readErr != nil {
							return internaldto.NewErroneousExecutorOutput(
								fmt.Errorf("failed to read transformed stream: %w", readErr))
						}
						outputStr := string(outputBytes)
						stdoutStr = outputStr
					}
				}
				var res []map[string]interface{}
				resErr := json.Unmarshal([]byte(stdoutStr), &res)
				itemisationResult := itemise(res, resErr, "")
				insertPrepResult := mv.ActionInsertPreparation(
					newHTTPActionInsertPayload(
						itemisationResult,
						false,
						tableName,
						flatInlineParams,
						"",
					),
				)
				insertErr, hasErr := insertPrepResult.GetError()
				if hasErr {
					return internaldto.NewErroneousExecutorOutput(insertErr)
				}
				// fmt.Fprintf(os.Stdout, "%s", stdoutStr)
			}
			// if stdOutExists {
			// 	backendMessages = append(backendMessages, stdOut.String())
			// }
			stdErr, stdErrExists := resp.GetStdErr()
			if stdErrExists {
				backendMessages = append(backendMessages, stdErr.String())
			}
			backendMessages = append(backendMessages, "OK")
			return internaldto.NewExecutorOutput(
				nil,
				nil,
				nil,
				internaldto.NewBackendMessages(backendMessages),
				nil,
			)
		case client.HTTP:
			invRes, invErr := mv.invoker.Invoke(context.Background(), providerinvoker.Request{
				Payload: formulation.NewPayload(
					mv.tableMeta,
					provider,
					m,
					tableName,
					authCtx,
					mv.handlerCtx.GetRuntimeContext(),
					mv.handlerCtx.GetOutErrFile(),
					mr,
					mv.elideActionIfPossible(
						currentTcc,
						tableName,
						"", // late binding, should remove AOT reference
					),
					true,
					polyHandler,
					mv.tableMeta.GetSelectItemsKey(),
					mv,
					mv.isSkipResponse,
					mv.isMutation,
					mv.isAwait,
					mv.defaultHTTPClient,
					mv.handlerCtx,
				),
			})
			if invErr != nil {
				return internaldto.NewErroneousExecutorOutput(invErr)
			}
			var castMessages internaldto.BackendMessages
			if len(invRes.Messages) > 0 {
				castMessages = internaldto.NewBackendMessages(invRes.Messages)
			}
			if invRes.Body == nil {
				return internaldto.NewExecutorOutput(nil, nil, nil, castMessages, nil)
			}
			bdyMap, bodyMapOk := invRes.Body.(map[string]interface{})
			if !bodyMapOk {
				return internaldto.NewErroneousExecutorOutput(fmt.Errorf("unexpected body type '%T'", invRes.Body))
			}
			return internaldto.NewExecutorOutput(nil, bdyMap, nil, castMessages, err)
		default:
			return internaldto.NewErroneousExecutorOutput(
				fmt.Errorf("unsupported protocol type '%v'", protocolType))
		}
	}
	return ex, nil
}

func shimProcessHTTP(
	url string,
	rtCtx dto.RuntimeCtx,
	authCtx *dto.AuthCtx,
	provider formulation.Provider,
	m formulation.OperationStore,
	outErrFile io.Writer,
) (*http.Response, error) {
	req, monitorReqErr := formulation.GetMonitorRequest(url)
	if monitorReqErr != nil {
		return nil, monitorReqErr
	}
	cc := formulation.NewAnySdkClientConfigurator(rtCtx, provider.GetName(), nil)
	anySdkResponse, apiErr := formulation.CallFromSignature(
		cc, rtCtx, authCtx, authCtx.Type, false, outErrFile, provider, formulation.NewAnySdkOpStoreDesignation(m), req)

	if apiErr != nil {
		return nil, apiErr
	}
	httpResponse, httpResponseErr := anySdkResponse.GetHttpResponse()
	if httpResponseErr != nil {
		return nil, httpResponseErr
	}
	return httpResponse, nil
}

//nolint:funlen,gocognit,errcheck // acceptable for now
func GetMonitorExecutor(
	handlerCtx handler.HandlerContext,
	provider formulation.Provider,
	op formulation.OperationStore,
	precursor primitive.IPrimitive,
	initialCtx primitive.IPrimitiveCtx,
	comments sqlparser.CommentDirectives,
	isReturning bool,
	insertCtx drm.PreparedStatementCtx,
	drmCfg drm.Config,
) (primitive.IPrimitive, error) {
	m := op
	// tableName, err := mv.tableMeta.GetTableName()
	// if err != nil {
	// 	return nil, err
	// }
	// authCtx, authCtxErr := mv.handlerCtx.GetAuthContext(prov.GetProviderString())
	// if authCtxErr != nil {
	// 	return nil, authCtxErr
	// }
	asyncPrim := asyncHTTPMonitorPrimitive{
		handlerCtx:          handlerCtx,
		prov:                provider,
		op:                  op,
		initialCtx:          initialCtx,
		precursor:           precursor,
		elapsedSeconds:      0,
		pollIntervalSeconds: MonitorPollIntervalSeconds,
		comments:            comments,
		insertCtx:           insertCtx,
		drmCfg:              drmCfg,
	}
	if comments != nil {
		asyncPrim.noStatus = comments.IsSet("NOSTATUS")
	}
	rtCtx := handlerCtx.GetRuntimeContext()
	outErrFile := handlerCtx.GetOutErrFile()
	asyncPrim.executor = func(pc primitive.IPrimitiveCtx, bd interface{}) internaldto.ExecutorOutput {
		body, ok := bd.(map[string]interface{})
		if !ok {
			return internaldto.NewExecutorOutput(
				nil,
				nil,
				nil,
				nil,
				fmt.Errorf("cannot execute monitor: response body of type '%T' unreadable", bd),
			)
		}
		if pc == nil {
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, fmt.Errorf("cannot execute monitor: nil plan primitive"))
		}
		if body == nil {
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, fmt.Errorf("cannot execute monitor: no body present"))
		}
		logging.GetLogger().Infoln(fmt.Sprintf("body = %v", body))

		operationDescriptor := getOpDescriptor(body)
		endTime, endTimeOk := body["endTime"]
		prStr := provider.GetName()
		//nolint:nestif // acceptable for now
		if endTimeOk && endTime != "" {
			targetLink, targetLinkOK := body["targetLink"]
			if targetLinkOK && isReturning {
				authCtx, authErr := pc.GetAuthContext(prStr)
				if authErr != nil {
					return internaldto.NewExecutorOutput(nil, nil, nil, nil, authErr)
				}
				if authCtx == nil {
					return internaldto.NewExecutorOutput(nil, nil, nil, nil, fmt.Errorf("cannot execute monitor: no auth context"))
				}
				targetLinkStr, targetLinkStrOk := targetLink.(string)
				if !targetLinkStrOk {
					return internaldto.NewExecutorOutput(
						nil,
						nil,
						nil,
						nil,
						fmt.Errorf("cannot execute monitor: 'targetLink' is not a string"),
					)
				}
				httpResponse, httpResponseErr := shimProcessHTTP(
					targetLinkStr,
					rtCtx,
					authCtx,
					provider,
					m,
					outErrFile,
				)
				if httpResponseErr != nil {
					return internaldto.NewExecutorOutput(nil, nil, nil, nil, httpResponseErr)
				}

				if httpResponse != nil && httpResponse.Body != nil {
					defer httpResponse.Body.Close()
				}
				target, targetErr := m.DeprecatedProcessResponse(httpResponse)
				handlerCtx.LogHTTPResponseMap(target)
				if targetErr != nil {
					return internaldto.NewExecutorOutput(nil, nil, nil, nil, targetErr)
				}
				// TODO: insert into table here
				if isReturning {
					if asyncPrim.insertCtx != nil {
						_, rErr := asyncPrim.drmCfg.ExecuteInsertDML(
							handlerCtx.GetSQLEngine(),
							asyncPrim.insertCtx,
							target,
							"", // TODO: figure out how on earth to compute this encoding
						)
						if rErr != nil {
							return internaldto.NewExecutorOutput(nil, nil, nil, nil, rErr)
						}
					}
				}
				return prepareResultSet(&asyncPrim, pc, target, operationDescriptor)
			}
			return prepareResultSet(&asyncPrim, pc, body, operationDescriptor)
		}
		url, ok := body["selfLink"]
		if !ok {
			return internaldto.NewExecutorOutput(
				nil,
				nil,
				nil,
				nil,
				fmt.Errorf("cannot execute monitor: no 'selfLink' property present"),
			)
		}
		authCtx, authErr := pc.GetAuthContext(prStr)
		if authErr != nil {
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, authErr)
		}
		if authCtx == nil {
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, fmt.Errorf("cannot execute monitor: no auth context"))
		}
		time.Sleep(time.Duration(asyncPrim.pollIntervalSeconds) * time.Second)
		asyncPrim.elapsedSeconds += asyncPrim.pollIntervalSeconds
		if !asyncPrim.noStatus {
			//nolint:errcheck //TODO: handle error
			pc.GetErrWriter().Write(
				[]byte(
					fmt.Sprintf(
						"%s in progress, %d seconds elapsed",
						operationDescriptor,
						asyncPrim.elapsedSeconds,
					) + fmt.Sprintln(""),
				),
			)
		}
		req, monitorReqErr := formulation.GetMonitorRequest(url.(string))
		if monitorReqErr != nil {
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, monitorReqErr)
		}
		cc := formulation.NewAnySdkClientConfigurator(rtCtx, provider.GetName(), nil)
		anySdkResponse, apiErr := formulation.CallFromSignature(
			cc, rtCtx, authCtx, authCtx.Type, false, outErrFile, provider, formulation.NewAnySdkOpStoreDesignation(m), req)

		if apiErr != nil {
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, apiErr)
		}
		httpResponse, httpResponseErr := anySdkResponse.GetHttpResponse()
		if httpResponseErr != nil {
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, httpResponseErr)
		}

		if httpResponse != nil && httpResponse.Body != nil {
			defer httpResponse.Body.Close()
		}
		target, targetErr := m.DeprecatedProcessResponse(httpResponse)
		handlerCtx.LogHTTPResponseMap(target)
		if targetErr != nil {
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, targetErr)
		}
		return asyncPrim.executor(internaldto.NewBasicPrimitiveContext(
			pc.GetAuthContext,
			pc.GetWriter(),
			pc.GetErrWriter(),
		),
			target)
	}
	return &asyncPrim, nil
}

func extractNextPageToken(res formulation.Response, tokenKey sdk_internal_dto.HTTPElement) string {
	//nolint:exhaustive // TODO: review
	switch tokenKey.GetType() {
	case sdk_internal_dto.BodyAttribute:
		return extractNextPageTokenFromBody(res, tokenKey)
	case sdk_internal_dto.Header:
		return extractNextPageTokenFromHeader(res, tokenKey)
	}
	return ""
}

//nolint:bodyclose // acceptable for now
func extractNextPageTokenFromHeader(res formulation.Response, tokenKey sdk_internal_dto.HTTPElement) string {
	r := res.GetHttpResponse()
	if r == nil {
		return ""
	}
	header := r.Header
	if tokenKey.IsTransformerPresent() {
		tf, err := tokenKey.Transformer(header)
		if err != nil {
			return ""
		}
		rv, ok := tf.(string)
		if !ok {
			return ""
		}
		return rv
	}
	vals := header.Values(tokenKey.GetName())
	if len(vals) == 1 {
		return vals[0]
	}
	return ""
}

func extractNextPageTokenFromBody(res formulation.Response, tokenKey sdk_internal_dto.HTTPElement) string {
	elem, err := httpelement.NewHTTPElement(tokenKey.GetName(), "body")
	if err == nil {
		rawVal, rawErr := res.ExtractElement(elem)
		if rawErr == nil {
			switch v := rawVal.(type) {
			case []interface{}:
				if len(v) == 1 {
					return fmt.Sprintf("%v", v[0])
				}
			default:
				return fmt.Sprintf("%v", v)
			}
		}
	}
	body := res.GetProcessedBody()
	switch target := body.(type) { //nolint:gocritic // TODO: review
	case map[string]interface{}:
		tokenName := tokenKey.GetName()
		nextPageToken, ok := target[tokenName]
		if !ok || nextPageToken == "" {
			logging.GetLogger().Infoln("breaking out")
			return ""
		}
		tk, ok := nextPageToken.(string)
		if !ok {
			logging.GetLogger().Infoln("breaking out")
			return ""
		}
		return tk
	}
	return ""
}

type asyncHTTPMonitorPrimitive struct {
	handlerCtx          handler.HandlerContext
	prov                formulation.Provider
	op                  formulation.OperationStore
	initialCtx          primitive.IPrimitiveCtx
	precursor           primitive.IPrimitive
	executor            func(pc primitive.IPrimitiveCtx, initalBody interface{}) internaldto.ExecutorOutput
	elapsedSeconds      int
	pollIntervalSeconds int
	noStatus            bool
	id                  int64
	comments            sqlparser.CommentDirectives
	insertCtx           drm.PreparedStatementCtx
	drmCfg              drm.Config
}

func (pr *asyncHTTPMonitorPrimitive) SetTxnID(_ int) {
}

func (pr *asyncHTTPMonitorPrimitive) IsReadOnly() bool {
	return false
}

func (pr *asyncHTTPMonitorPrimitive) GetRedoLog() (binlog.LogEntry, bool) {
	return nil, false
}

func (pr *asyncHTTPMonitorPrimitive) GetUndoLog() (binlog.LogEntry, bool) {
	return nil, false
}

func (pr *asyncHTTPMonitorPrimitive) WithDebugName(_ string) primitive.IPrimitive {
	return pr
}

func (pr *asyncHTTPMonitorPrimitive) SetUndoLog(_ binlog.LogEntry) {
}

func (pr *asyncHTTPMonitorPrimitive) SetRedoLog(_ binlog.LogEntry) {
}

func (pr *asyncHTTPMonitorPrimitive) IncidentData(fromID int64, input internaldto.ExecutorOutput) error {
	return pr.precursor.IncidentData(fromID, input)
}

func (pr *asyncHTTPMonitorPrimitive) SetInputAlias(alias string, id int64) error {
	return pr.precursor.SetInputAlias(alias, id)
}

func (pr *asyncHTTPMonitorPrimitive) Optimise() error {
	return nil
}

func (pr *asyncHTTPMonitorPrimitive) Execute(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
	if pr.executor != nil {
		if pc == nil {
			pc = pr.initialCtx
		}
		subPr := pr.precursor.Execute(pc)
		if subPr.GetError() != nil || pr.executor == nil {
			return subPr
		}
		prStr := pr.prov.GetName()
		// seems pointless
		_, err := pr.initialCtx.GetAuthContext(prStr)
		if err != nil {
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, err)
		}
		//
		asyP := internaldto.NewBasicPrimitiveContext(
			pr.initialCtx.GetAuthContext,
			pc.GetWriter(),
			pc.GetErrWriter(),
		)
		return pr.executor(asyP, subPr.GetOutputBody())
	}
	return internaldto.NewExecutorOutput(nil, nil, nil, nil, nil)
}

func (pr *asyncHTTPMonitorPrimitive) ID() int64 {
	return pr.id
}

func (pr *asyncHTTPMonitorPrimitive) GetInputFromAlias(string) (internaldto.ExecutorOutput, bool) {
	var rv internaldto.ExecutorOutput
	return rv, false
}

func (pr *asyncHTTPMonitorPrimitive) SetExecutor(_ func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput) error {
	return fmt.Errorf("asyncHTTPMonitorPrimitive does not support SetExecutor()")
}

func getOpDescriptor(body map[string]interface{}) string {
	operationDescriptor := "operation"
	if body == nil {
		return operationDescriptor
	}
	//nolint:nestif,govet // TODO: refactor
	if descriptor, ok := body["kind"]; ok {
		if descriptorStr, ok := descriptor.(string); ok {
			operationDescriptor = descriptorStr
			if typeElem, ok := body["operationType"]; ok {
				if typeStr, ok := typeElem.(string); ok {
					operationDescriptor = fmt.Sprintf("%s: %s", descriptorStr, typeStr)
				}
			}
		}
	}
	return operationDescriptor
}

func prepareResultSet(
	prim *asyncHTTPMonitorPrimitive,
	pc primitive.IPrimitiveCtx,
	target map[string]interface{},
	operationDescriptor string,
) internaldto.ExecutorOutput {
	payload := internaldto.PrepareResultSetDTO{
		OutputBody:  target,
		Msg:         nil,
		RowMap:      nil,
		ColumnOrder: nil,
		RowSort:     nil,
		Err:         nil,
	}
	if !prim.noStatus {
		//nolint:errcheck //TODO: handle error
		pc.GetErrWriter().Write([]byte(fmt.Sprintf("%s complete", operationDescriptor) + fmt.Sprintln("")))
	}
	return util.PrepareResultSet(payload)
}

func castItemsArray(iArr interface{}) ([]map[string]interface{}, error) {
	switch iArr := iArr.(type) {
	case []map[string]interface{}:
		return iArr, nil
	case []interface{}:
		var rv []map[string]interface{}
		for i := range iArr {
			item, ok := iArr[i].(map[string]interface{})
			if !ok {
				if iArr[i] != nil {
					item = map[string]interface{}{formulation.AnonymousColumnName: iArr[i]}
				} else {
					item = nil
				}
			}
			rv = append(rv, item)
		}
		return rv, nil
	default:
		return nil, fmt.Errorf("cannot accept items array of type = '%T'", iArr)
	}
}

func shallowGenerateSuccessMessagesFromHeirarchy(isAwait bool) []string {
	baseSuccessString := "The operation completed successfully"
	if !isAwait {
		baseSuccessString = "The operation was despatched successfully"
	}
	successMsgs := []string{
		baseSuccessString,
	}
	return successMsgs
}
