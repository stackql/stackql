package primitivebuilder

import (
	"fmt"
	"strconv"

	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/httpbuild"
	"github.com/stackql/stackql/internal/stackql/httpmiddleware"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"vitess.io/vitess/go/vt/sqlparser"
)

type Insert struct {
	graph               primitivegraph.PrimitiveGraph
	handlerCtx          handler.HandlerContext
	drmCfg              drm.DRMConfig
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
	_, _, err = tbl.GetResponseSchemaAndMediaType()
	if err != nil {
		return err
	}
	insertPrimitive := primitive.NewHTTPRestPrimitive(
		prov,
		nil,
		nil,
		nil,
	)
	var target map[string]interface{}
	ex := func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
		input, inputExists := insertPrimitive.GetInputFromAlias("")
		if !inputExists {
			return internaldto.NewErroneousExecutorOutput(fmt.Errorf("input does not exist"))
		}
		inputStream, err := input.ResultToMap()
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		rr, err := inputStream.Read()
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		inputMap, err := rr.GetMap()
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		httpArmoury, err := httpbuild.BuildHTTPRequestCtx(node, prov, m, svc, inputMap, nil)
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}

		var zeroArityExecutors []func() internaldto.ExecutorOutput
		for _, r := range httpArmoury.GetRequestParams() {
			req := r
			zeroArityEx := func() internaldto.ExecutorOutput {
				// logging.GetLogger().Infoln(fmt.Sprintf("req.BodyBytes = %s", string(req.BodyBytes)))
				// req.Context.SetBody(bytes.NewReader(req.BodyBytes))
				// logging.GetLogger().Infoln(fmt.Sprintf("req.Context = %v", req.Context))
				response, apiErr := httpmiddleware.HttpApiCallFromRequest(handlerCtx.Clone(), prov, m, req.GetRequest())
				if apiErr != nil {
					return internaldto.NewErroneousExecutorOutput(apiErr)
				}

				target, err = m.DeprecatedProcessResponse(response)
				handlerCtx.LogHTTPResponseMap(target)
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
				composeAsyncMonitor(handlerCtx, insertPrimitive, tbl, commentDirectives)
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
				logging.GetLogger().Infoln(fmt.Sprintf("target = %v", target))
				items, ok := target[tbl.LookupSelectItemsKey()]
				keys := make(map[string]map[string]interface{})
				if ok {
					iArr, ok := items.([]interface{})
					if ok && len(iArr) > 0 {
						for i := range iArr {
							item, ok := iArr[i].(map[string]interface{})
							if ok {
								keys[strconv.Itoa(i)] = item
							}
						}
					}
				}
				msgs := internaldto.BackendMessages{}
				if err == nil {
					msgs.WorkingMessages = generateSuccessMessagesFromHeirarchy(tbl, isAwait)
				} else {
					msgs.WorkingMessages = []string{err.Error()}
				}
				return internaldto.NewExecutorOutput(nil, target, nil, &msgs, err)
			}
			zeroArityExecutors = append(zeroArityExecutors, zeroArityEx)
		}
		resultSet := internaldto.NewErroneousExecutorOutput(fmt.Errorf("no executions detected"))
		msgs := internaldto.BackendMessages{}
		if !isAwait {
			for _, ei := range zeroArityExecutors {
				execInstance := ei
				resultSet = execInstance()
				if resultSet.Msg != nil && resultSet.Msg.WorkingMessages != nil && len(resultSet.Msg.WorkingMessages) > 0 {
					for _, m := range resultSet.Msg.WorkingMessages {
						msgs.WorkingMessages = append(msgs.WorkingMessages, m)
					}
				}
				if resultSet.Err != nil {
					resultSet.Msg = &msgs
					return resultSet
				}
			}
			resultSet.Msg = &msgs
			return resultSet
		}
		for _, eI := range zeroArityExecutors {
			execInstance := eI
			dependentInsertPrimitive := primitive.NewHTTPRestPrimitive(
				prov,
				nil,
				nil,
				nil,
			)
			err = dependentInsertPrimitive.SetExecutor(func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
				return execInstance()
			})
			if err != nil {
				return internaldto.NewErroneousExecutorOutput(err)
			}
			execPrim, err := composeAsyncMonitor(handlerCtx, dependentInsertPrimitive, tbl, commentDirectives)
			if err != nil {
				return internaldto.NewErroneousExecutorOutput(err)
			}
			resultSet = execPrim.Execute(pc)
			if resultSet.Err != nil {
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
