package primitivebuilder

import (
	"fmt"
	"strconv"

	"github.com/stackql/any-sdk/pkg/httpelement"
	"github.com/stackql/any-sdk/pkg/response"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/httpmiddleware"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/primitive_context"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/streaming"
	"github.com/stackql/stackql/internal/stackql/tableinsertioncontainer"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"github.com/stackql/stackql/internal/stackql/util"

	sdk_internal_dto "github.com/stackql/any-sdk/pkg/internaldto"
)

// SingleSelectAcquire implements the Builder interface
// and represents the action of acquiring data from an endpoint
// and then persisting that data into a table.
// This data would then subsequently be queried by later execution phases.
type SingleSelectAcquire struct {
	graphHolder                primitivegraph.PrimitiveGraphHolder
	handlerCtx                 handler.HandlerContext
	tableMeta                  tablemetadata.ExtendedTableMetadata
	drmCfg                     drm.Config
	insertPreparedStatementCtx drm.PreparedStatementCtx
	insertionContainer         tableinsertioncontainer.TableInsertionContainer
	txnCtrlCtr                 internaldto.TxnControlCounters
	rowSort                    func(map[string]map[string]interface{}) []string
	root                       primitivegraph.PrimitiveNode
	stream                     streaming.MapStream
	isReadOnly                 bool //nolint:unused // TODO: build out
}

func NewSingleSelectAcquire(
	graphHolder primitivegraph.PrimitiveGraphHolder,
	handlerCtx handler.HandlerContext,
	insertionContainer tableinsertioncontainer.TableInsertionContainer,
	insertCtx drm.PreparedStatementCtx,
	rowSort func(map[string]map[string]interface{}) []string,
	stream streaming.MapStream,
) Builder {
	tableMeta := insertionContainer.GetTableMetadata()
	_, isGraphQL := tableMeta.GetGraphQL()
	if isGraphQL {
		return newGraphQLSingleSelectAcquire(
			graphHolder,
			handlerCtx,
			tableMeta,
			insertCtx,
			insertionContainer,
			rowSort,
			stream,
		)
	}
	return newSingleSelectAcquire(
		graphHolder,
		handlerCtx,
		tableMeta,
		insertCtx,
		insertionContainer,
		rowSort,
		stream,
	)
}

func newSingleSelectAcquire(
	graphHolder primitivegraph.PrimitiveGraphHolder,
	handlerCtx handler.HandlerContext,
	tableMeta tablemetadata.ExtendedTableMetadata,
	insertCtx drm.PreparedStatementCtx,
	insertionContainer tableinsertioncontainer.TableInsertionContainer,
	rowSort func(map[string]map[string]interface{}) []string,
	stream streaming.MapStream,
) Builder {
	var tcc internaldto.TxnControlCounters
	if insertCtx != nil {
		tcc = insertCtx.GetGCCtrlCtrs()
	}
	if stream == nil {
		stream = streaming.NewNopMapStream()
	}
	return &SingleSelectAcquire{
		graphHolder:                graphHolder,
		handlerCtx:                 handlerCtx,
		tableMeta:                  tableMeta,
		rowSort:                    rowSort,
		drmCfg:                     handlerCtx.GetDrmConfig(),
		insertPreparedStatementCtx: insertCtx,
		insertionContainer:         insertionContainer,
		txnCtrlCtr:                 tcc,
		stream:                     stream,
	}
}

func (ss *SingleSelectAcquire) GetRoot() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *SingleSelectAcquire) GetTail() primitivegraph.PrimitiveNode {
	return ss.root
}

//nolint:funlen,gocognit,gocyclo,cyclop,nestif // TODO: investigate
func (ss *SingleSelectAcquire) Build() error {
	prov, err := ss.tableMeta.GetProvider()
	if err != nil {
		return err
	}
	m, err := ss.tableMeta.GetMethod()
	if err != nil {
		return err
	}
	tableName, err := ss.tableMeta.GetTableName()
	if err != nil {
		return err
	}
	ex := func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
		logging.GetLogger().Infof("SingleSelectAcquire.Execute() beginning execution for table %s", tableName)
		currentTcc := ss.insertPreparedStatementCtx.GetGCCtrlCtrs().Clone()
		ss.graphHolder.AddTxnControlCounters(currentTcc)
		mr := prov.InferMaxResultsElement(m)
		// TODO: instrument for split source vertices !!!important!!!
		httpArmoury, armouryErr := ss.tableMeta.GetHTTPArmoury()
		if armouryErr != nil {
			return internaldto.NewErroneousExecutorOutput(armouryErr)
		}
		if mr != nil {
			// TODO: infer param position and act accordingly
			ok := true
			if ok && ss.handlerCtx.GetRuntimeContext().HTTPMaxResults > 0 {
				passOverParams := httpArmoury.GetRequestParams()
				for i, p := range passOverParams {
					param := p
					// param.Context.SetQueryParam("maxResults", strconv.Itoa(ss.handlerCtx.GetRuntimeContext().HTTPMaxResults))
					q := param.GetQuery()
					q.Set("maxResults", strconv.Itoa(ss.handlerCtx.GetRuntimeContext().HTTPMaxResults))
					param.SetRawQuery(q.Encode())
					passOverParams[i] = param
				}
				httpArmoury.SetRequestParams(passOverParams)
			}
		}
		reqParams := httpArmoury.GetRequestParams()
		logging.GetLogger().Infof("SingleSelectAcquire.Execute() req param count = %d", len(reqParams))
		for _, rc := range reqParams {
			reqCtx := rc
			paramsUsed, paramErr := reqCtx.ToFlatMap()
			if paramErr != nil {
				return internaldto.NewErroneousExecutorOutput(paramErr)
			}
			reqEncoding := reqCtx.Encode()
			//nolint:lll // chaining
			olderTcc, isMatch := ss.handlerCtx.GetNamespaceCollection().GetAnalyticsCacheTableNamespaceConfigurator().Match(tableName, reqEncoding, ss.drmCfg.GetControlAttributes().GetControlLatestUpdateColumnName(), ss.drmCfg.GetControlAttributes().GetControlInsertEncodedIDColumnName())
			if isMatch {
				nonControlColumns := ss.insertPreparedStatementCtx.GetNonControlColumns()
				var nonControlColumnNames []string
				for _, c := range nonControlColumns {
					nonControlColumnNames = append(nonControlColumnNames, c.GetName())
				}
				//nolint:errcheck // TODO: fix
				ss.handlerCtx.GetGarbageCollector().Update(
					tableName,
					olderTcc.Clone(),
					currentTcc,
				)
				//nolint:errcheck // TODO: fix
				ss.insertionContainer.SetTableTxnCounters(tableName, olderTcc)
				ss.insertPreparedStatementCtx.SetGCCtrlCtrs(olderTcc)
				//nolint:rowserrcheck // TODO: fix this
				r, sqlErr := ss.handlerCtx.GetNamespaceCollection().GetAnalyticsCacheTableNamespaceConfigurator().Read(
					tableName, reqEncoding,
					ss.drmCfg.GetControlAttributes().GetControlInsertEncodedIDColumnName(),
					nonControlColumnNames)
				if sqlErr != nil {
					internaldto.NewErroneousExecutorOutput(sqlErr)
				}
				ss.drmCfg.ExtractObjectFromSQLRows(r, nonControlColumns, ss.stream)
				return internaldto.NewEmptyExecutorOutput()
			}
			// TODO: fix cloning ops
			response, apiErr := httpmiddleware.HTTPApiCallFromRequest(
				ss.handlerCtx.Clone(),
				prov,
				m,
				reqCtx.GetRequest().Clone(
					reqCtx.GetRequest().Context(),
				),
			)
			housekeepingDone := false
			npt := prov.InferNextPageResponseElement(ss.tableMeta.GetHeirarchyObjects())
			nptRequest := prov.InferNextPageRequestElement(ss.tableMeta.GetHeirarchyObjects())
			pageCount := 1
			for {
				if apiErr != nil {
					return util.PrepareResultSet(internaldto.NewPrepareResultSetDTO(nil, nil, nil, ss.rowSort, apiErr, nil,
						ss.handlerCtx.GetTypingConfig(),
					))
				}
				processed, resErr := m.ProcessResponse(response)
				if resErr != nil {
					//nolint:errcheck // TODO: fix
					ss.handlerCtx.GetOutErrFile().Write(
						[]byte(fmt.Sprintf("error processing response: %s\n", resErr.Error())),
					)
					if processed == nil {
						return internaldto.NewErroneousExecutorOutput(resErr)
					}
				}
				res, respOk := processed.GetResponse()
				if !respOk {
					return internaldto.NewErroneousExecutorOutput(fmt.Errorf("response is not a valid response"))
				}
				if res.HasError() {
					return internaldto.NewNopEmptyExecutorOutput([]string{res.Error()})
				}
				ss.handlerCtx.LogHTTPResponseMap(res.GetProcessedBody())
				logging.GetLogger().Infoln(fmt.Sprintf("SingleSelectAcquire.Execute() response = %v", res))
				var items interface{}
				var ok bool
				target := res.GetProcessedBody()
				logging.GetLogger().Infoln(fmt.Sprintf("SingleSelectAcquire.Execute() target = %v", target))
				switch pl := target.(type) {
				// add case for xml object,
				case map[string]interface{}:
					if ss.tableMeta.GetSelectItemsKey() != "" && ss.tableMeta.GetSelectItemsKey() != "/*" {
						items, ok = pl[ss.tableMeta.GetSelectItemsKey()]
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
					return internaldto.NewEmptyExecutorOutput()
				}
				keys := make(map[string]map[string]interface{})

				//nolint:nestif // TODO: fix
				if ok {
					iArr, iErr := castItemsArray(items)
					if iErr != nil {
						return internaldto.NewErroneousExecutorOutput(iErr)
					}
					err = ss.stream.Write(iArr)
					if err != nil {
						return internaldto.NewErroneousExecutorOutput(err)
					}
					if ok && len(iArr) > 0 {
						if !housekeepingDone && ss.insertPreparedStatementCtx != nil {
							_, err = ss.handlerCtx.GetSQLEngine().Exec(ss.insertPreparedStatementCtx.GetGCHousekeepingQueries())
							tcc := ss.insertPreparedStatementCtx.GetGCCtrlCtrs()
							tcc.SetTableName(tableName)
							//nolint:errcheck // TODO: fix
							ss.insertionContainer.SetTableTxnCounters(tableName, tcc)
							housekeepingDone = true
						}
						if err != nil {
							return internaldto.NewErroneousExecutorOutput(err)
						}

						for i, item := range iArr {
							if item != nil {
								if err == nil {
									for k, v := range paramsUsed {
										if _, itemOk := item[k]; !itemOk {
											item[k] = v
										}
									}
								}

								logging.GetLogger().Infoln(
									fmt.Sprintf(
										"running insert with query = '''%s''', control parameters: %v",
										ss.insertPreparedStatementCtx.GetQuery(),
										ss.insertPreparedStatementCtx.GetGCCtrlCtrs(),
									),
								)
								r, rErr := ss.drmCfg.ExecuteInsertDML(
									ss.handlerCtx.GetSQLEngine(),
									ss.insertPreparedStatementCtx,
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
									return internaldto.NewErroneousExecutorOutput(
										fmt.Errorf(
											"sql insert error: '%w' from query: %s",
											rErr,
											ss.insertPreparedStatementCtx.GetQuery(),
										),
									)
								}
								keys[strconv.Itoa(i)] = item
							}
						}
					}
				}
				if npt == nil || nptRequest == nil {
					break
				}
				tk := extractNextPageToken(res, npt)
				//nolint:lll // long conditional
				if tk == "" || tk == "<nil>" || tk == "[]" || (ss.handlerCtx.GetRuntimeContext().HTTPPageLimit > 0 && pageCount >= ss.handlerCtx.GetRuntimeContext().HTTPPageLimit) {
					break
				}
				pageCount++
				req, reqErr := reqCtx.SetNextPage(m, tk, nptRequest)
				if reqErr != nil {
					return internaldto.NewErroneousExecutorOutput(reqErr)
				}
				response, apiErr = httpmiddleware.HTTPApiCallFromRequest(ss.handlerCtx.Clone(), prov, m, req)
			}
			if reqCtx.GetRequest() != nil {
				q := reqCtx.GetRequest().URL.Query()
				q.Del(nptRequest.GetName())
				reqCtx.SetRawQuery(q.Encode())
			}
		}
		logging.GetLogger().Infof("SingleSelectAcquire.Execute() returning empty for table %s", tableName)
		return internaldto.NewEmptyExecutorOutput()
	}

	prep := func() drm.PreparedStatementCtx {
		return ss.insertPreparedStatementCtx
	}
	primitiveCtx := primitive_context.NewPrimitiveContext()
	primitiveCtx.SetIsReadOnly(true)
	insertPrim := primitive.NewHTTPRestPrimitive(
		prov,
		ex,
		prep,
		ss.txnCtrlCtr,
		primitiveCtx,
	).WithDebugName(fmt.Sprintf("insert_%s_%s", tableName, ss.tableMeta.GetAlias()))
	graphHolder := ss.graphHolder
	insertNode := graphHolder.CreatePrimitiveNode(insertPrim)
	ss.root = insertNode

	return nil
}

func extractNextPageToken(res response.Response, tokenKey sdk_internal_dto.HTTPElement) string {
	//nolint:exhaustive // TODO: review
	switch tokenKey.GetType() {
	case sdk_internal_dto.BodyAttribute:
		return extractNextPageTokenFromBody(res, tokenKey)
	case sdk_internal_dto.Header:
		return extractNextPageTokenFromHeader(res, tokenKey)
	}
	return ""
}

func extractNextPageTokenFromHeader(res response.Response, tokenKey sdk_internal_dto.HTTPElement) string {
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

func extractNextPageTokenFromBody(res response.Response, tokenKey sdk_internal_dto.HTTPElement) string {
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
