package primitivebuilder

import (
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"

	"github.com/stackql/any-sdk/public/formulation"

	"github.com/stackql/any-sdk/pkg/graphql"
	"github.com/stackql/any-sdk/pkg/logging"
	"github.com/stackql/any-sdk/pkg/streaming"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/builder_input"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/primitive_context"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/tableinsertioncontainer"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
)

// GraphQLSingleSelectAcquire implements the Builder interface
// and represents the action of acquiring data from an endpoint
// and then persisting that data into a table.
// This data would then subsequently be queried by later execution phases.
type GraphQLSingleSelectAcquire struct {
	graph                      primitivegraph.PrimitiveGraphHolder
	handlerCtx                 handler.HandlerContext
	tableMeta                  tablemetadata.ExtendedTableMetadata
	drmCfg                     drm.Config
	insertPreparedStatementCtx drm.PreparedStatementCtx
	insertionContainer         tableinsertioncontainer.TableInsertionContainer
	txnCtrlCtr                 internaldto.TxnControlCounters
	rowSort                    func(map[string]map[string]interface{}) []string
	root                       primitivegraph.PrimitiveNode
	stream                     streaming.MapStream
	bldrInput                  builder_input.BuilderInput
}

func newGraphQLSingleSelectAcquire(
	graph primitivegraph.PrimitiveGraphHolder,
	handlerCtx handler.HandlerContext,
	tableMeta tablemetadata.ExtendedTableMetadata,
	insertCtx drm.PreparedStatementCtx,
	insertionContainer tableinsertioncontainer.TableInsertionContainer,
	rowSort func(map[string]map[string]interface{}) []string,
	stream streaming.MapStream,
	bldrInput builder_input.BuilderInput,
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
		bldrInput:                  bldrInput,
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
	provider, err := prov.GetProvider()
	if err != nil {
		return err
	}
	authCtx, err := ss.handlerCtx.GetAuthContext(provider.GetName())
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
			// Emit the GraphQL wire request + raw response when --http.log.enabled is set
			// (alpha08 ContextWithHTTPLogger), mirroring the REST acquire path.
			if ss.handlerCtx.GetRuntimeContext().HTTPLogEnabled {
				req = req.WithContext(
					graphql.ContextWithHTTPLogger(req.Context(), ss.handlerCtx.GetOutErrFile()))
			}
			housekeepingDone := false
			cc := formulation.NewAnySdkClientConfigurator(
				ss.handlerCtx.GetRuntimeContext(), prov.GetProviderString(),
				prov.GetDefaultHTTPClient())
			client, err := cc.Auth(authCtx, authCtx.Type, false)
			if err != nil {
				return internaldto.NewErroneousExecutorOutput(err)
			}
			paramMap, err := reqCtx.GetParameters().ToFlatMap()
			if err != nil {
				return internaldto.NewErroneousExecutorOutput(err)
			}
			// Push a SQL LIMIT N into the GraphQL query variables as `limit`, so a
			// provider query template referencing {{ .limit }} can bound the page size.
			if node, nodeOk := ss.bldrInput.GetParserNode(); nodeOk {
				if limit, hasLimit := graphQLSelectLimit(node); hasLimit {
					paramMap["limit"] = limit
				}
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
			transformType, transformBody := extractGraphQLResponseTransform(ss.tableMeta)
			cursorCfg := buildGraphQLCursorConfig(gql, cursorJsonPath)
			graphQLReader, err := graphql.NewStandardGQLReaderFull(
				client,
				req,
				ss.handlerCtx.GetRuntimeContext().HTTPPageLimit,
				gql.GetQuery(),
				paramMap,
				"",
				responseJsonPath,
				cursorCfg,
				transformType,
				transformBody,
			)
			if err != nil {
				return internaldto.NewErroneousExecutorOutput(err)
			}
			for {
				response, readErr := graphQLReader.Read()
				ss.handlerCtx.LogHTTPResponseMap(response)
				if len(response) > 0 {
					if processErr := ss.processGraphQLPage(response, tableName, &housekeepingDone); processErr != nil {
						return internaldto.NewErroneousExecutorOutput(processErr)
					}
				}
				if errors.Is(readErr, io.EOF) {
					break
				}
				if readErr != nil {
					return internaldto.NewErroneousExecutorOutput(readErr)
				}
			}
		}
		return internaldto.NewEmptyExecutorOutput()
	}

	prep := func() drm.PreparedStatementCtx {
		return ss.insertPreparedStatementCtx
	}
	insertPrim := primitive.NewGenericPrimitive(
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

func (ss *GraphQLSingleSelectAcquire) processGraphQLPage(
	response []map[string]interface{},
	tableName string,
	housekeepingDone *bool,
) error {
	if !*housekeepingDone && ss.insertPreparedStatementCtx != nil {
		_, hkErr := ss.handlerCtx.GetSQLEngine().Exec(ss.insertPreparedStatementCtx.GetGCHousekeepingQueries())
		//nolint:errcheck // pre-existing behaviour: housekeeping counter set is best-effort
		ss.insertionContainer.SetTableTxnCounters(tableName, ss.insertPreparedStatementCtx.GetGCCtrlCtrs())
		*housekeepingDone = true
		if hkErr != nil {
			return hkErr
		}
	}
	if writeErr := ss.stream.Write(response); writeErr != nil {
		return writeErr
	}
	for _, item := range response {
		// TODO: handle request encoding
		r, insertErr := ss.drmCfg.ExecuteInsertDML(ss.handlerCtx.GetSQLEngine(), ss.insertPreparedStatementCtx, item, "")
		logging.GetLogger().Infoln(fmt.Sprintf("insert result = %v, error = %v", r, insertErr))
		if insertErr != nil {
			return insertErr
		}
	}
	return nil
}

func extractGraphQLResponseTransform(tableMeta tablemetadata.ExtendedTableMetadata) (string, string) {
	op, err := tableMeta.GetMethod()
	if err != nil || op == nil {
		return "", ""
	}
	er, ok := op.GetResponse()
	if !ok || er == nil {
		return "", ""
	}
	t, ok := er.GetTransform()
	if !ok || t == nil {
		return "", ""
	}
	return t.GetType(), t.GetBody()
}

// graphQLSelectLimit returns the integer LIMIT of a SELECT statement, if present.
func graphQLSelectLimit(node sqlparser.SQLNode) (int, bool) {
	sel, ok := node.(*sqlparser.Select)
	if !ok || sel.Limit == nil {
		return 0, false
	}
	v, ok := sel.Limit.Rowcount.(*sqlparser.SQLVal)
	if !ok || v.Type != sqlparser.IntVal {
		return 0, false
	}
	n, err := strconv.Atoi(string(v.Val))
	if err != nil {
		return 0, false
	}
	return n, true
}

func buildGraphQLCursorConfig(gql formulation.GraphQL, cursorJSONPath string) graphql.CursorConfig {
	cfg := graphql.CursorConfig{JSONPath: cursorJSONPath}
	if strategy, ok := gql.GetCursorStrategy(); ok && strategy != "" {
		cfg.Strategy = graphql.CursorStrategy(strategy)
	}
	if format, ok := gql.GetCursorFormat(); ok && format != "" {
		cfg.FormatTemplate = format
	}
	if terminator, ok := gql.GetCursorTerminateOnJSONPath(); ok && terminator != "" {
		cfg.TerminateOnJSONPath = terminator
	}
	if pageSize, ok := gql.GetCursorPageSize(); ok && pageSize > 0 {
		cfg.PageSize = pageSize
	}
	return cfg
}
