package primitivebuilder

import (
	"errors"
	"fmt"
	"io"

	"github.com/stackql/go-openapistackql/pkg/graphql"
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
)

// GraphQLSingleSelectAcquire implements the Builder interface
// and represents the action of acquiring data from an endpoint
// and then persisting that data into a table.
// This data would then subsequently be queried by later execution phases.
type GraphQLSingleSelectAcquire struct {
	graph                      primitivegraph.PrimitiveGraph
	handlerCtx                 handler.HandlerContext
	tableMeta                  tablemetadata.ExtendedTableMetadata
	drmCfg                     drm.Config
	insertPreparedStatementCtx drm.PreparedStatementCtx
	insertionContainer         tableinsertioncontainer.TableInsertionContainer
	txnCtrlCtr                 internaldto.TxnControlCounters
	rowSort                    func(map[string]map[string]interface{}) []string
	root                       primitivegraph.PrimitiveNode
	stream                     streaming.MapStream
}

func newGraphQLSingleSelectAcquire(
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
	return &GraphQLSingleSelectAcquire{
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

func (ss *GraphQLSingleSelectAcquire) GetRoot() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *GraphQLSingleSelectAcquire) GetTail() primitivegraph.PrimitiveNode {
	return ss.root
}

//nolint:govet,funlen,gocognit,revive,errcheck,stylecheck // TODO: fix
func (ss *GraphQLSingleSelectAcquire) Build() error {
	prov, err := ss.tableMeta.GetProvider()
	if err != nil {
		return err
	}
	httpArmoury, err := ss.tableMeta.GetHTTPArmoury()
	if err != nil {
		return err
	}
	gql, ok := ss.tableMeta.GetGraphQL()
	if !ok {
		return fmt.Errorf("could not build graphql exection for table")
	}
	ex := func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
		currentTcc := ss.insertPreparedStatementCtx.GetGCCtrlCtrs().Clone()
		ss.graph.AddTxnControlCounters(currentTcc)

		for _, reqCtx := range httpArmoury.GetRequestParams() {
			req := reqCtx.GetRequest()
			housekeepingDone := false
			client, err := httpmiddleware.GetAuthenticatedClient(ss.handlerCtx.Clone(), prov)
			if err != nil {
				return internaldto.NewErroneousExecutorOutput(err)
			}
			paramMap, err := reqCtx.GetParameters().ToFlatMap()
			if err != nil {
				return internaldto.NewErroneousExecutorOutput(err)
			}
			cursorJsonPath, ok := gql.GetCursorJSONPath()
			if !ok {
				return internaldto.NewErroneousExecutorOutput(fmt.Errorf("cannot perform graphql action without cursor json path"))
			}
			responseJsonPath, ok := gql.GetResponseJSONPath()
			if !ok {
				return internaldto.NewErroneousExecutorOutput(
					fmt.Errorf("cannot perform graphql action without response json path"),
				)
			}
			tableName, err := ss.tableMeta.GetTableName()
			if err != nil {
				return internaldto.NewErroneousExecutorOutput(err)
			}
			reqEncoding := reqCtx.Encode()
			//nolint:lll // chained
			olderTcc, isMatch := ss.handlerCtx.GetNamespaceCollection().GetAnalyticsCacheTableNamespaceConfigurator().Match(tableName, reqEncoding, ss.drmCfg.GetControlAttributes().GetControlLatestUpdateColumnName(), ss.drmCfg.GetControlAttributes().GetControlInsertEncodedIDColumnName())
			if isMatch {
				nonControlColumns := ss.insertPreparedStatementCtx.GetNonControlColumns()
				var nonControlColumnNames []string
				for _, c := range nonControlColumns {
					nonControlColumnNames = append(nonControlColumnNames, c.GetName())
				}
				ss.handlerCtx.GetGarbageCollector().Update(tableName, olderTcc, currentTcc)
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
			graphQLReader, err := graphql.NewStandardGQLReader(
				client,
				req,
				ss.handlerCtx.GetRuntimeContext().HTTPPageLimit,
				gql.GetQuery(),
				paramMap,
				"",
				responseJsonPath,
				cursorJsonPath,
			)
			if err != nil {
				return internaldto.NewErroneousExecutorOutput(err)
			}
			for {
				response, err := graphQLReader.Read()
				if len(response) > 0 {
					if !housekeepingDone && ss.insertPreparedStatementCtx != nil {
						_, err = ss.handlerCtx.GetSQLEngine().Exec(ss.insertPreparedStatementCtx.GetGCHousekeepingQueries())
						ss.insertionContainer.SetTableTxnCounters(tableName, ss.insertPreparedStatementCtx.GetGCCtrlCtrs())
						housekeepingDone = true
					}
					if err != nil {
						return internaldto.NewErroneousExecutorOutput(err)
					}
					err = ss.stream.Write(response)
					if err != nil {
						return internaldto.NewErroneousExecutorOutput(err)
					}
					for _, item := range response {
						// TODO: handle request encoding
						r, err := ss.drmCfg.ExecuteInsertDML(ss.handlerCtx.GetSQLEngine(), ss.insertPreparedStatementCtx, item, "")
						logging.GetLogger().Infoln(fmt.Sprintf("insert result = %v, error = %v", r, err))
						if err != nil {
							return internaldto.NewErroneousExecutorOutput(err)
						}
					}
				}
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
			}
		}
		return internaldto.NewEmptyExecutorOutput()
	}

	prep := func() drm.PreparedStatementCtx {
		return ss.insertPreparedStatementCtx
	}
	insertPrim := primitive.NewHTTPRestPrimitive(
		prov,
		ex,
		prep,
		ss.txnCtrlCtr,
		primitive_context.NewPrimitiveContext(),
	)
	graph := ss.graph
	insertNode := graph.CreatePrimitiveNode(insertPrim)
	ss.root = insertNode

	return nil
}

func (ss *GraphQLSingleSelectAcquire) SetWriteOnly(_ bool) {
}

func (ss *GraphQLSingleSelectAcquire) IsWriteOnly() bool {
	return false
}
