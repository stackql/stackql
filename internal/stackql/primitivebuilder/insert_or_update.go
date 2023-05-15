package primitivebuilder

import (
	"fmt"
	"strconv"

	"github.com/stackql/go-openapistackql/openapistackql"
	pkg_response "github.com/stackql/go-openapistackql/pkg/response"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/acid/binlog"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/httpmiddleware"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/primitive_context"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"github.com/stackql/stackql/internal/stackql/util"
)

type InsertOrUpdate struct {
	graph               primitivegraph.PrimitiveGraph
	handlerCtx          handler.HandlerContext
	drmCfg              drm.Config
	root                primitivegraph.PrimitiveNode
	tbl                 tablemetadata.ExtendedTableMetadata
	node                sqlparser.SQLNode
	commentDirectives   sqlparser.CommentDirectives
	selectPrimitiveNode primitivegraph.PrimitiveNode
	isAwait             bool
	verb                string // may be "insert" or "update"
}

func NewInsertOrUpdate(
	graph primitivegraph.PrimitiveGraph,
	handlerCtx handler.HandlerContext,
	node sqlparser.SQLNode,
	tbl tablemetadata.ExtendedTableMetadata,
	selectPrimitiveNode primitivegraph.PrimitiveNode,
	commentDirectives sqlparser.CommentDirectives,
	isAwait bool,
	verb string,
) Builder {
	return &InsertOrUpdate{
		graph:               graph,
		handlerCtx:          handlerCtx,
		drmCfg:              handlerCtx.GetDrmConfig(),
		tbl:                 tbl,
		node:                node,
		commentDirectives:   commentDirectives,
		selectPrimitiveNode: selectPrimitiveNode,
		isAwait:             isAwait,
		verb:                verb,
	}
}

func (ss *InsertOrUpdate) GetRoot() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *InsertOrUpdate) GetTail() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *InsertOrUpdate) decorateOutput(op internaldto.ExecutorOutput, tableName string) internaldto.ExecutorOutput {
	op.SetUndoLog(
		binlog.NewSimpleLogEntry(
			nil,
			[]string{
				fmt.Sprintf("Undo the %s on %s", ss.verb, tableName),
			},
		),
	)
	return op
}

//nolint:funlen,errcheck,gocognit,cyclop,gocyclo // TODO: fix this
func (ss *InsertOrUpdate) Build() error {
	node := ss.node
	tbl := ss.tbl
	handlerCtx := ss.handlerCtx
	commentDirectives := ss.commentDirectives
	isAwait := ss.isAwait
	switch node := node.(type) {
	case *sqlparser.Insert, *sqlparser.Update:
	default:
		return fmt.Errorf("mutation executor: cannnot accomodate node of type '%T'", node)
	}
	prov, err := tbl.GetProvider()
	if err != nil {
		return err
	}
	svc, err := tbl.GetService()
	if err != nil {
		return err
	}
	m, err := tbl.GetMethod()
	if err != nil {
		return err
	}
	_, _, responseAnalysisErr := tbl.GetResponseSchemaAndMediaType()
	insertPrimitive := primitive.NewHTTPRestPrimitive(
		prov,
		nil,
		nil,
		nil,
		primitive_context.NewPrimitiveContext(),
	)
	target := make(map[string]interface{})
	ex := func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
		input, inputExists := insertPrimitive.GetInputFromAlias("")
		if !inputExists {
			return internaldto.NewErroneousExecutorOutput(fmt.Errorf("input does not exist"))
		}
		inputStream, inputErr := input.ResultToMap()
		if inputErr != nil {
			return internaldto.NewErroneousExecutorOutput(inputErr)
		}
		rr, rrErr := inputStream.Read()
		if rrErr != nil {
			return internaldto.NewErroneousExecutorOutput(rrErr)
		}
		inputMap, inputErr := rr.GetMap()
		if inputErr != nil {
			return internaldto.NewErroneousExecutorOutput(inputErr)
		}
		pr, prErr := prov.GetProvider()
		if prErr != nil {
			return internaldto.NewErroneousExecutorOutput(prErr)
		}
		paramMap, paramErr := util.ExtractSQLNodeParams(node, inputMap)
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(paramErr)
		}
		httpPreparator := openapistackql.NewHTTPPreparator(
			pr,
			svc,
			m,
			inputMap,
			paramMap,
			nil,
			nil,
			logging.GetLogger(),
		)
		httpArmoury, httpErr := httpPreparator.BuildHTTPRequestCtx()
		if httpErr != nil {
			return internaldto.NewErroneousExecutorOutput(httpErr)
		}

		tableName, _ := tbl.GetTableName()

		var nullaryExecutors []func() internaldto.ExecutorOutput
		for _, r := range httpArmoury.GetRequestParams() {
			req := r
			nullaryEx := func() internaldto.ExecutorOutput {
				response, apiErr := httpmiddleware.HTTPApiCallFromRequest(handlerCtx.Clone(), prov, m, req.GetRequest())
				if apiErr != nil {
					return internaldto.NewErroneousExecutorOutput(apiErr)
				}

				if responseAnalysisErr == nil {
					var resp pkg_response.Response
					resp, err = m.ProcessResponse(response)
					if err != nil {
						return internaldto.NewErroneousExecutorOutput(err)
					}
					processedBody := resp.GetProcessedBody()
					switch processedBody := processedBody.(type) { //nolint:gocritic // TODO: fix this
					case map[string]interface{}:
						target = processedBody
					}
				}
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
				logging.GetLogger().Infoln(fmt.Sprintf("target = %v", target))
				items, ok := target[tbl.LookupSelectItemsKey()]
				keys := make(map[string]map[string]interface{})
				if ok {
					iArr, iOk := items.([]interface{})
					if iOk && len(iArr) > 0 {
						for i := range iArr {
							item, itemOk := iArr[i].(map[string]interface{})
							if itemOk {
								keys[strconv.Itoa(i)] = item
							}
						}
					}
				}
				if err == nil {
					if response.StatusCode < 300 { //nolint:gomnd // TODO: fix this
						msgs := internaldto.NewBackendMessages(
							generateSuccessMessagesFromHeirarchy(tbl, isAwait),
						)
						return ss.decorateOutput(
							internaldto.NewExecutorOutput(
								nil,
								target,
								nil,
								msgs,
								nil,
							),
							tableName,
						)
					}
					generatedErr := fmt.Errorf("insert over HTTP error: %s", response.Status)
					return internaldto.NewExecutorOutput(
						nil,
						target,
						nil,
						nil,
						generatedErr,
					)
				}
				return internaldto.NewExecutorOutput(
					nil,
					target,
					nil,
					nil,
					err,
				)
			}

			nullaryExecutors = append(nullaryExecutors, nullaryEx)
		}
		resultSet := internaldto.NewErroneousExecutorOutput(fmt.Errorf("no executions detected"))
		if !isAwait {
			for _, ei := range nullaryExecutors {
				execInstance := ei
				aPrioriMessages := resultSet.GetMessages()
				resultSet = execInstance()
				resultSet.AppendMessages(aPrioriMessages)
				if resultSet.GetError() != nil {
					return resultSet
				}
			}
			return resultSet
		}
		for _, eI := range nullaryExecutors {
			execInstance := eI
			dependentInsertPrimitive := primitive.NewHTTPRestPrimitive(
				prov,
				nil,
				nil,
				nil,
				primitive_context.NewPrimitiveContext(),
			)
			err = dependentInsertPrimitive.SetExecutor(func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
				return execInstance()
			})
			if err != nil {
				return internaldto.NewErroneousExecutorOutput(err)
			}
			execPrim, execErr := composeAsyncMonitor(handlerCtx, dependentInsertPrimitive, tbl, commentDirectives)
			if execErr != nil {
				return internaldto.NewErroneousExecutorOutput(execErr)
			}
			resultSet = execPrim.Execute(pc)
			if resultSet.GetError() != nil {
				return resultSet
			}
		}
		return ss.decorateOutput(
			resultSet,
			tableName,
		)
	}
	err = insertPrimitive.SetExecutor(ex)
	if err != nil {
		return err
	}

	graph := ss.graph

	insertPrimitive.SetInputAlias("", ss.selectPrimitiveNode.ID())
	insertNode := graph.CreatePrimitiveNode(insertPrimitive)
	graph.NewDependency(ss.selectPrimitiveNode, insertNode, 1.0)
	ss.root = ss.selectPrimitiveNode

	return nil
}

func (ss *InsertOrUpdate) SetWriteOnly(_ bool) {
}

func (ss *InsertOrUpdate) IsWriteOnly() bool {
	return false
}
