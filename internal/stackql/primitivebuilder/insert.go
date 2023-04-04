package primitivebuilder

import (
	"fmt"
	"strconv"

	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
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

type Insert struct {
	graph               primitivegraph.PrimitiveGraph
	handlerCtx          handler.HandlerContext
	drmCfg              drm.Config
	root                primitivegraph.PrimitiveNode
	tbl                 tablemetadata.ExtendedTableMetadata
	node                sqlparser.SQLNode
	commentDirectives   sqlparser.CommentDirectives
	selectPrimitiveNode primitivegraph.PrimitiveNode
	isAwait             bool
}

func NewInsert(
	graph primitivegraph.PrimitiveGraph,
	handlerCtx handler.HandlerContext,
	node sqlparser.SQLNode,
	tbl tablemetadata.ExtendedTableMetadata,
	selectPrimitiveNode primitivegraph.PrimitiveNode,
	commentDirectives sqlparser.CommentDirectives,
	isAwait bool,
) Builder {
	return &Insert{
		graph:               graph,
		handlerCtx:          handlerCtx,
		drmCfg:              handlerCtx.GetDrmConfig(),
		tbl:                 tbl,
		node:                node,
		commentDirectives:   commentDirectives,
		selectPrimitiveNode: selectPrimitiveNode,
		isAwait:             isAwait,
	}
}

func (ss *Insert) GetRoot() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *Insert) GetTail() primitivegraph.PrimitiveNode {
	return ss.root
}

//nolint:funlen,errcheck,gocognit,cyclop,gocyclo // TODO: fix this
func (ss *Insert) Build() error {
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

		var nullaryExecutors []func() internaldto.ExecutorOutput
		for _, r := range httpArmoury.GetRequestParams() {
			req := r
			nullaryEx := func() internaldto.ExecutorOutput {
				// logging.GetLogger().Infoln(fmt.Sprintf("req.BodyBytes = %s", string(req.BodyBytes)))
				// req.Context.SetBody(bytes.NewReader(req.BodyBytes))
				// logging.GetLogger().Infoln(fmt.Sprintf("req.Context = %v", req.Context))
				response, apiErr := httpmiddleware.HTTPApiCallFromRequest(handlerCtx.Clone(), prov, m, req.GetRequest())
				if apiErr != nil {
					return internaldto.NewErroneousExecutorOutput(apiErr)
				}

				if responseAnalysisErr == nil {
					target, err = m.DeprecatedProcessResponse(response)
					handlerCtx.LogHTTPResponseMap(target)
					if err != nil {
						return internaldto.NewErroneousExecutorOutput(err)
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
						return internaldto.NewExecutorOutput(
							nil,
							target,
							nil,
							msgs,
							nil,
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
		return resultSet
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
