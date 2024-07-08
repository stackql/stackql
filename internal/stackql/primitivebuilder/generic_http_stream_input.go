package primitivebuilder

import (
	"fmt"
	"strconv"

	"github.com/stackql/any-sdk/anysdk"
	pkg_response "github.com/stackql/any-sdk/pkg/response"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/acid/binlog"
	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/httpmiddleware"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/builder_input"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/primitive_context"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/streaming/http_preparator_stream.go"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"github.com/stackql/stackql/internal/stackql/util"
)

type genericHTTPStreamInput struct {
	graphHolder       primitivegraph.PrimitiveGraphHolder
	handlerCtx        handler.HandlerContext
	drmCfg            drm.Config
	root              primitivegraph.PrimitiveNode
	tbl               tablemetadata.ExtendedTableMetadata
	commentDirectives sqlparser.CommentDirectives
	dependencyNode    primitivegraph.PrimitiveNode
	parserNode        sqlparser.SQLNode
	isAwait           bool
	verb              string // may be "insert" or "update"
	inputAlias        string
	isUndo            bool
	reversalStream    http_preparator_stream.HttpPreparatorStream
	reversalBuilder   Builder
	rollbackType      constants.RollbackType
}

func newGenericHTTPStreamInput(
	builderInput builder_input.BuilderInput,
) (Builder, error) {
	handlerCtx, handlerCtxExists := builderInput.GetHandlerContext()
	if !handlerCtxExists {
		return nil, fmt.Errorf("handler context is required")
	}
	graphHolder, graphHolderExists := builderInput.GetGraphHolder()
	if !graphHolderExists {
		return nil, fmt.Errorf("graph holder is required")
	}
	tbl, tblExists := builderInput.GetTableMetadata()
	if !tblExists {
		return nil, fmt.Errorf("table metadata is required")
	}
	commentDirectives, _ := builderInput.GetCommentDirectives()
	dependencyNode, dependencyNodeExists := builderInput.GetDependencyNode()
	if !dependencyNodeExists {
		return nil, fmt.Errorf("dependency node is required")
	}
	parserNode, _ := builderInput.GetParserNode()
	return &genericHTTPStreamInput{
		graphHolder:       graphHolder,
		handlerCtx:        handlerCtx,
		drmCfg:            handlerCtx.GetDrmConfig(),
		tbl:               tbl,
		commentDirectives: commentDirectives,
		dependencyNode:    dependencyNode,
		isAwait:           builderInput.IsAwait(),
		verb:              builderInput.GetVerb(),
		inputAlias:        builderInput.GetInputAlias(),
		isUndo:            builderInput.IsUndo(),
		parserNode:        parserNode,
		reversalStream:    http_preparator_stream.NewHttpPreparatorStream(),
		rollbackType:      handlerCtx.GetRollbackType(),
	}, nil
}

func (gh *genericHTTPStreamInput) isReverseRequired() bool {
	return gh.rollbackType != constants.NopRollback
}

func (gh *genericHTTPStreamInput) GetRoot() primitivegraph.PrimitiveNode {
	return gh.root
}

func (gh *genericHTTPStreamInput) GetTail() primitivegraph.PrimitiveNode {
	return gh.root
}

func (gh *genericHTTPStreamInput) appendReversalData(prep anysdk.HTTPPreparator) error {
	return gh.reversalStream.Write(prep)
}

func (gh *genericHTTPStreamInput) decorateOutput(
	op internaldto.ExecutorOutput, tableName string) internaldto.ExecutorOutput {
	op.SetUndoLog(
		binlog.NewSimpleLogEntry(
			nil,
			[]string{
				fmt.Sprintf("Undo the %s on %s", gh.verb, tableName),
			},
		),
	)
	return op
}

func (gh *genericHTTPStreamInput) getInterestingMaps(actionPrimitive primitive.IPrimitive) (mapsAggregatorDTO, error) {
	input, inputExists := actionPrimitive.GetInputFromAlias(gh.inputAlias)
	if !inputExists {
		return nil, fmt.Errorf("input does not exist")
	}
	inputStream, inputErr := input.ResultToMap()
	if inputErr != nil {
		return nil, inputErr
	}
	rr, rrErr := inputStream.Read()
	if rrErr != nil {
		return nil, rrErr
	}
	inputMap, inputErr := rr.GetMap()
	if inputErr != nil {
		return nil, inputErr
	}
	paramMap, paramErr := util.ExtractSQLNodeParams(gh.parserNode, inputMap)
	if paramErr != nil {
		return nil, paramErr
	}
	return newMapsAggregatorDTO(paramMap, inputMap), nil
}

//nolint:funlen,gocognit,gocyclo,cyclop,nestif // TODO: fix this
func (gh *genericHTTPStreamInput) Build() error {
	tbl := gh.tbl
	handlerCtx := gh.handlerCtx
	commentDirectives := gh.commentDirectives
	isAwait := gh.isAwait
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
	inverse, inverseExists := m.GetInverse()
	if !inverseExists && gh.isReverseRequired() {
		return fmt.Errorf("inverse is required")
	}
	if inverseExists {
		inverseOpStore, inverseOpStoreExists := inverse.GetOperationStore()
		if !inverseOpStoreExists {
			return fmt.Errorf("inverse operation store is required")
		}
		logging.GetLogger().Debugf("inverse = %v", inverse)
		var reversalBuildInitErr error
		reverseInput := builder_input.NewBuilderInput(
			gh.graphHolder,
			handlerCtx,
			nil, // tbl,
		)
		reverseInput.SetHTTPPreparatorStream(gh.reversalStream)
		reverseInput.SetOperationStore(inverseOpStore)
		reverseInput.SetProvider(prov)
		gh.reversalBuilder, reversalBuildInitErr = newGenericHTTPReversal(reverseInput)
		if reversalBuildInitErr != nil {
			return reversalBuildInitErr
		}
		buildErr := gh.reversalBuilder.Build()
		if buildErr != nil {
			return buildErr
		}
		// inverseInput :=
		// inverseBuilder :=
	}
	_, _, responseAnalysisErr := tbl.GetResponseSchemaAndMediaType()
	actionPrimitive := primitive.NewHTTPRestPrimitive(
		prov,
		nil,
		nil,
		nil,
		primitive_context.NewPrimitiveContext(),
	)
	// reversalStream := streaming.NewStandardMapStream()
	target := make(map[string]interface{})
	ex := func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
		pr, prErr := prov.GetProvider()
		if prErr != nil {
			return internaldto.NewErroneousExecutorOutput(prErr)
		}
		interestingMaps, mapsErr := gh.getInterestingMaps(actionPrimitive)
		if mapsErr != nil {
			return internaldto.NewErroneousExecutorOutput(mapsErr)
		}
		httpPreparator := anysdk.NewHTTPPreparator(
			pr,
			svc,
			m,
			interestingMaps.getParameterMap(),
			nil,
			nil,
			logging.GetLogger(),
		)
		httpArmoury, httpErr := httpPreparator.BuildHTTPRequestCtx()
		if httpErr != nil {
			return internaldto.NewErroneousExecutorOutput(httpErr)
		}

		tableName, _ := tbl.GetTableName()

		// var reversalParameters []map[string]interface{}
		// TODO: Implement reversal operations:
		//           - depending upon reversal operation, collect sequence of HTTP operations.
		//           - for each HTTP operation, collect context and store in *some object*
		//                then attach to reversal graph for later retrieval and execution
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
					processed, procErr := m.ProcessResponse(response)
					if err != nil {
						return internaldto.NewErroneousExecutorOutput(procErr)
					}
					reversal, reversalExists := processed.GetReversal()
					if reversalExists {
						reversalAppendErr := gh.appendReversalData(reversal)
						if reversalAppendErr != nil {
							return internaldto.NewErroneousExecutorOutput(reversalAppendErr)
						}
					}
					if !reversalExists && gh.isReverseRequired() {
						return internaldto.NewErroneousExecutorOutput(fmt.Errorf("reversal is required but not provided"))
					}
					resp, respOk := processed.GetResponse()
					if !respOk {
						return internaldto.NewErroneousExecutorOutput(fmt.Errorf("response is not a valid response"))
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
					if response.StatusCode < 300 { //nolint:mnd // TODO: fix this
						msgs := internaldto.NewBackendMessages(
							generateSuccessMessagesFromHeirarchy(tbl, isAwait),
						)
						return gh.decorateOutput(
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
			//nolint:revive // no big deal
			err = dependentInsertPrimitive.SetExecutor(func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
				return execInstance()
			})
			if err != nil {
				return internaldto.NewErroneousExecutorOutput(err)
			}
			execPrim, execErr := composeAsyncMonitor(handlerCtx, dependentInsertPrimitive, prov, m, commentDirectives)
			if execErr != nil {
				return internaldto.NewErroneousExecutorOutput(execErr)
			}
			resultSet = execPrim.Execute(pc)
			if resultSet.GetError() != nil {
				return resultSet
			}
		}
		return gh.decorateOutput(
			resultSet,
			tableName,
		)
	}
	err = actionPrimitive.SetExecutor(ex)
	if err != nil {
		return err
	}

	graphHolder := gh.graphHolder

	//nolint:errcheck // TODO: fix this
	actionPrimitive.SetInputAlias(gh.inputAlias, gh.dependencyNode.ID())
	actionNode := graphHolder.CreatePrimitiveNode(actionPrimitive)
	graphHolder.NewDependency(gh.dependencyNode, actionNode, 1.0)
	gh.root = gh.dependencyNode

	return nil
}
