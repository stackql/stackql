package primitivebuilder

import (
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"
	"github.com/stackql/go-openapistackql/pkg/graphql"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/httpmiddleware"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/streaming"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
)

// GraphQLSingleSelectAcquire implements the Builder interface
// and represents the action of acquiring data from an endpoint
// and then persisting that data into a table.
// This data would then subsequently be queried by later execution phases.
type GraphQLSingleSelectAcquire struct {
	graph                      *primitivegraph.PrimitiveGraph
	handlerCtx                 *handler.HandlerContext
	tableMeta                  *taxonomy.ExtendedTableMetadata
	drmCfg                     drm.DRMConfig
	insertPreparedStatementCtx *drm.PreparedStatementCtx
	txnCtrlCtr                 *dto.TxnControlCounters
	rowSort                    func(map[string]map[string]interface{}) []string
	root                       primitivegraph.PrimitiveNode
	stream                     streaming.MapStream
}

func newGraphQLSingleSelectAcquire(
	graph *primitivegraph.PrimitiveGraph,
	handlerCtx *handler.HandlerContext,
	tableMeta *taxonomy.ExtendedTableMetadata,
	insertCtx *drm.PreparedStatementCtx,
	rowSort func(map[string]map[string]interface{}) []string,
	stream streaming.MapStream,
) Builder {
	var tcc *dto.TxnControlCounters
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
		drmCfg:                     handlerCtx.DrmConfig,
		insertPreparedStatementCtx: insertCtx,
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

func (ss *GraphQLSingleSelectAcquire) Build() error {
	prov, err := ss.tableMeta.GetProvider()
	if err != nil {
		return err
	}
	httpArmoury, err := ss.tableMeta.GetHttpArmoury()
	if err != nil {
		return err
	}
	gql, ok := ss.tableMeta.GetGraphQL()
	if !ok {
		return fmt.Errorf("could not build graphql exection for table")
	}
	ex := func(pc primitive.IPrimitiveCtx) dto.ExecutorOutput {
		ss.graph.AddTxnControlCounters(*ss.insertPreparedStatementCtx.GetGCCtrlCtrs())

		for _, reqCtx := range httpArmoury.GetRequestParams() {
			req := reqCtx.Request
			housekeepingDone := false
			client, err := httpmiddleware.GetAuthenticatedClient(*ss.handlerCtx, prov)
			if err != nil {
				return dto.NewErroneousExecutorOutput(err)
			}
			paramMap, err := reqCtx.Parameters.ToFlatMap()
			if err != nil {
				return dto.NewErroneousExecutorOutput(err)
			}
			cursorJsonPath, ok := gql.GetCursorJSONPath()
			if !ok {
				return dto.NewErroneousExecutorOutput(fmt.Errorf("cannot perform graphql action without cursor json path"))
			}
			responseJsonPath, ok := gql.GetResponseJSONPath()
			if !ok {
				return dto.NewErroneousExecutorOutput(fmt.Errorf("cannot perform graphql action without response json path"))
			}
			graphQLReader, err := graphql.NewStandardGQLReader(
				client,
				req,
				ss.handlerCtx.RuntimeContext.HTTPPageLimit,
				gql.Query,
				paramMap,
				"",
				responseJsonPath,
				cursorJsonPath,
			)
			if err != nil {
				return dto.NewErroneousExecutorOutput(err)
			}
			for {
				response, err := graphQLReader.Read()
				if len(response) > 0 {
					if !housekeepingDone && ss.insertPreparedStatementCtx != nil {
						_, err = ss.handlerCtx.SQLEngine.Exec(ss.insertPreparedStatementCtx.GetGCHousekeepingQueries())
						housekeepingDone = true
					}
					if err != nil {
						return dto.NewErroneousExecutorOutput(err)
					}
					err = ss.stream.Write(response)
					if err != nil {
						return dto.NewErroneousExecutorOutput(err)
					}
					for _, item := range response {
						r, err := ss.drmCfg.ExecuteInsertDML(ss.handlerCtx.SQLEngine, ss.insertPreparedStatementCtx, item)
						log.Infoln(fmt.Sprintf("insert result = %v, error = %v", r, err))
						if err != nil {
							return dto.NewErroneousExecutorOutput(err)
						}
					}
				}
				if err == io.EOF {
					break
				}
				if err != nil {
					return dto.NewErroneousExecutorOutput(err)
				}
			}
		}
		return dto.ExecutorOutput{}
	}

	prep := func() *drm.PreparedStatementCtx {
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
