package primitivebuilder

import (
	"fmt"
	"strconv"

	"github.com/stackql/go-openapistackql/pkg/httpelement"
	"github.com/stackql/go-openapistackql/pkg/response"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/httpmiddleware"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/streaming"
	"github.com/stackql/stackql/internal/stackql/tableinsertioncontainer"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"github.com/stackql/stackql/internal/stackql/util"
)

// SingleSelectAcquire implements the Builder interface
// and represents the action of acquiring data from an endpoint
// and then persisting that data into a table.
// This data would then subsequently be queried by later execution phases.
type SingleSelectAcquire struct {
	graph                      primitivegraph.PrimitiveGraph
	handlerCtx                 handler.HandlerContext
	tableMeta                  tablemetadata.ExtendedTableMetadata
	drmCfg                     drm.DRMConfig
	insertPreparedStatementCtx drm.PreparedStatementCtx
	insertionContainer         tableinsertioncontainer.TableInsertionContainer
	txnCtrlCtr                 internaldto.TxnControlCounters
	rowSort                    func(map[string]map[string]interface{}) []string
	root                       primitivegraph.PrimitiveNode
	stream                     streaming.MapStream
}

func NewSingleSelectAcquire(
	graph primitivegraph.PrimitiveGraph,
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
			graph,
			handlerCtx,
			tableMeta,
			insertCtx,
			insertionContainer,
			rowSort,
			stream,
		)
	}
	return newSingleSelectAcquire(
		graph,
		handlerCtx,
		tableMeta,
		insertCtx,
		insertionContainer,
		rowSort,
		stream,
	)
}

func newSingleSelectAcquire(
	graph primitivegraph.PrimitiveGraph,
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
		graph:                      graph,
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
		currentTcc := ss.insertPreparedStatementCtx.GetGCCtrlCtrs().Clone()
		ss.graph.AddTxnControlCounters(currentTcc)
		mr := prov.InferMaxResultsElement(m)
		// TODO: instrument for view
		httpArmoury, err := ss.tableMeta.GetHttpArmoury()
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		if mr != nil {
			// TODO: infer param position and act accordingly
			ok := true
			if ok && ss.handlerCtx.GetRuntimeContext().HTTPMaxResults > 0 {
				passOverParams := httpArmoury.GetRequestParams()
				for i, param := range passOverParams {
					// param.Context.SetQueryParam("maxResults", strconv.Itoa(ss.handlerCtx.GetRuntimeContext().HTTPMaxResults))
					q := param.GetQuery()
					q.Set("maxResults", strconv.Itoa(ss.handlerCtx.GetRuntimeContext().HTTPMaxResults))
					param.SetRawQuery(q.Encode())
					passOverParams[i] = param
				}
				httpArmoury.SetRequestParams(passOverParams)
			}
		}
		for _, reqCtx := range httpArmoury.GetRequestParams() {
			paramsUsed, err := reqCtx.ToFlatMap()
			if err != nil {
				return internaldto.NewErroneousExecutorOutput(err)
			}
			reqEncoding := reqCtx.Encode()
			olderTcc, isMatch := ss.handlerCtx.GetNamespaceCollection().GetAnalyticsCacheTableNamespaceConfigurator().Match(tableName, reqEncoding, ss.drmCfg.GetControlAttributes().GetControlLatestUpdateColumnName(), ss.drmCfg.GetControlAttributes().GetControlInsertEncodedIdColumnName())
			if isMatch {
				nonControlColumns := ss.insertPreparedStatementCtx.GetNonControlColumns()
				var nonControlColumnNames []string
				for _, c := range nonControlColumns {
					nonControlColumnNames = append(nonControlColumnNames, c.GetName())
				}
				ss.handlerCtx.GetGarbageCollector().Update(tableName, olderTcc.Clone(), currentTcc)
				ss.insertionContainer.SetTableTxnCounters(tableName, olderTcc)
				ss.insertPreparedStatementCtx.SetGCCtrlCtrs(olderTcc)
				r, sqlErr := ss.handlerCtx.GetNamespaceCollection().GetAnalyticsCacheTableNamespaceConfigurator().Read(tableName, reqEncoding, ss.drmCfg.GetControlAttributes().GetControlInsertEncodedIdColumnName(), nonControlColumnNames)
				if sqlErr != nil {
					internaldto.NewErroneousExecutorOutput(sqlErr)
				}
				ss.drmCfg.ExtractObjectFromSQLRows(r, nonControlColumns, ss.stream)
				return internaldto.ExecutorOutput{}
			}
			// TODO: fix cloning ops
			response, apiErr := httpmiddleware.HttpApiCallFromRequest(ss.handlerCtx.Clone(), prov, m, reqCtx.GetRequest().Clone(reqCtx.GetRequest().Context()))
			housekeepingDone := false
			npt := prov.InferNextPageResponseElement(ss.tableMeta.GetHeirarchyObjects())
			nptRequest := prov.InferNextPageRequestElement(ss.tableMeta.GetHeirarchyObjects())
			pageCount := 1
			for {
				if apiErr != nil {
					return util.PrepareResultSet(internaldto.NewPrepareResultSetDTO(nil, nil, nil, ss.rowSort, apiErr, nil))
				}
				res, err := m.ProcessResponse(response)
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
				ss.handlerCtx.LogHTTPResponseMap(res.GetProcessedBody())
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
				logging.GetLogger().Infoln(fmt.Sprintf("target = %v", res))
				var items interface{}
				var ok bool
				target := res.GetProcessedBody()
				switch pl := target.(type) {
				// add case for xml object,
				case map[string]interface{}:
					if ss.tableMeta.GetSelectItemsKey() != "" && ss.tableMeta.GetSelectItemsKey() != "/*" {
						items, ok = pl[ss.tableMeta.GetSelectItemsKey()]
						if !ok {
							items = []interface{}{
								pl,
							}
							ok = true
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
					return internaldto.ExecutorOutput{}
				}
				keys := make(map[string]map[string]interface{})

				if ok {
					iArr, err := castItemsArray(items)
					if err != nil {
						return internaldto.NewErroneousExecutorOutput(err)
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
										if _, ok := item[k]; !ok {
											item[k] = v
										}
									}
								}

								logging.GetLogger().Infoln(fmt.Sprintf("running insert with control parameters: %v", ss.insertPreparedStatementCtx.GetGCCtrlCtrs()))
								r, err := ss.drmCfg.ExecuteInsertDML(ss.handlerCtx.GetSQLEngine(), ss.insertPreparedStatementCtx, item, reqEncoding)
								logging.GetLogger().Infoln(fmt.Sprintf("insert result = %v, error = %v", r, err))
								if err != nil {
									return internaldto.NewErroneousExecutorOutput(fmt.Errorf("sql insert error: '%s' from query: %s", err.Error(), ss.insertPreparedStatementCtx.GetQuery()))
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
				if tk == "" || tk == "<nil>" || tk == "[]" || (ss.handlerCtx.GetRuntimeContext().HTTPPageLimit > 0 && pageCount >= ss.handlerCtx.GetRuntimeContext().HTTPPageLimit) {
					break
				}
				pageCount++
				req, err := reqCtx.SetNextPage(m, tk, nptRequest)
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
				response, apiErr = httpmiddleware.HttpApiCallFromRequest(ss.handlerCtx.Clone(), prov, m, req)
			}
			if reqCtx.GetRequest() != nil {
				q := reqCtx.GetRequest().URL.Query()
				q.Del(nptRequest.GetName())
				reqCtx.SetRawQuery(q.Encode())
			}
		}
		return internaldto.ExecutorOutput{}
	}

	prep := func() drm.PreparedStatementCtx {
		return ss.insertPreparedStatementCtx
	}
	insertPrim := primitive.NewHTTPRestPrimitive(
		prov,
		ex,
		prep,
		ss.txnCtrlCtr,
	)
	graph := ss.graph
	insertNode := graph.CreatePrimitiveNode(insertPrim)
	ss.root = insertNode

	return nil
}

func extractNextPageToken(res *response.Response, tokenKey internaldto.HTTPElement) string {
	switch tokenKey.GetType() {
	case internaldto.BodyAttribute:
		return extractNextPageTokenFromBody(res, tokenKey)
	case internaldto.Header:
		return extractNextPageTokenFromHeader(res, tokenKey)
	}
	return ""
}

func extractNextPageTokenFromHeader(res *response.Response, tokenKey internaldto.HTTPElement) string {
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

func extractNextPageTokenFromBody(res *response.Response, tokenKey internaldto.HTTPElement) string {
	elem, err := httpelement.NewHTTPElement(tokenKey.GetName(), "body")
	if err == nil {
		rawVal, err := res.ExtractElement(elem)
		if err == nil {
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
	switch target := body.(type) {
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
