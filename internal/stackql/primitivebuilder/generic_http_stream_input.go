package primitivebuilder

import (
	"fmt"

	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/any-sdk/pkg/client"
	"github.com/stackql/any-sdk/pkg/constants"
	"github.com/stackql/any-sdk/pkg/local_template_executor"
	"github.com/stackql/any-sdk/pkg/logging"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/acid/binlog"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/execution"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/builder_input"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/primitive_context"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
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
	isMutation        bool
	reversalStream    anysdk.HttpPreparatorStream
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
		reversalStream:    anysdk.NewHttpPreparatorStream(),
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

//nolint:funlen,gocognit,gocyclo,cyclop // TODO: fix this
func (gh *genericHTTPStreamInput) Build() error {
	tbl := gh.tbl
	handlerCtx := gh.handlerCtx
	commentDirectives := gh.commentDirectives
	isAwait := gh.isAwait
	prov, err := tbl.GetProvider()
	if err != nil {
		return err
	}
	provider, providerErr := prov.GetProvider()
	if providerErr != nil {
		return providerErr
	}
	rtCtx := handlerCtx.GetRuntimeContext()
	authCtx, authCtxErr := handlerCtx.GetAuthContext(provider.GetName())
	if authCtxErr != nil {
		return authCtxErr
	}
	outErrFile := handlerCtx.GetOutErrFile()
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
	actionPrimitive := primitive.NewGenericPrimitive(
		nil,
		nil,
		nil,
		primitive_context.NewPrimitiveContext(),
	)
	// reversalStream := streaming.NewStandardMapStream()
	ex := func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
		pr, prErr := prov.GetProvider()
		if prErr != nil {
			return internaldto.NewErroneousExecutorOutput(prErr)
		}
		protocolType, protocolTypeErr := pr.GetProtocolType()
		if protocolTypeErr != nil {
			return internaldto.NewErroneousExecutorOutput(protocolTypeErr)
		}
		interestingMaps, mapsErr := gh.getInterestingMaps(actionPrimitive)
		if mapsErr != nil {
			return internaldto.NewErroneousExecutorOutput(mapsErr)
		}
		paramMap := interestingMaps.getParameterMap()
		params := paramMap[0]
		//nolint:exhaustive // no big deal
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
			resp, exErr := executor.Execute(
				map[string]any{"parameters": params},
			)
			if exErr != nil {
				return internaldto.NewErroneousExecutorOutput(exErr)
			}
			var backendMessages []string
			stdOut, stdOutExists := resp.GetStdOut()
			if stdOutExists {
				backendMessages = append(backendMessages, stdOut.String())
			}
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
			httpPreparator := anysdk.NewHTTPPreparator(
				pr,
				svc,
				m,
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

			// var reversalParameters []map[string]interface{}
			// TODO: Implement reversal operations:
			//           - depending upon reversal operation, collect sequence of HTTP operations.
			//           - for each HTTP operation, collect context and store in *some object*
			//                then attach to reversal graph for later retrieval and execution
			var nullaryExecutors []func() internaldto.ExecutorOutput
			for _, r := range httpArmoury.GetRequestParams() {
				req := r
				isSkipResponse := responseAnalysisErr != nil
				polyHandler := execution.NewStandardPolyHandler(
					handlerCtx,
				)
				nullaryEx := func() internaldto.ExecutorOutput {
					pp := execution.NewProcessorPayload(
						req,
						execution.NewNilMethodElider(),
						provider,
						m,
						tableName,
						rtCtx,
						authCtx,
						outErrFile,
						polyHandler,
						"",
						nil,
						isSkipResponse,
						true,
						isAwait,
						gh.isUndo,
						gh.isMutation,
						"",
					)
					processor := execution.NewProcessor(pp)
					processorResponse := processor.Process()
					processorErr := processorResponse.GetError()
					singletonBody := processorResponse.GetSingletonBody()
					reversalStrem := processorResponse.GetReversalStream()
					for {
						rev, isRevExistent := reversalStrem.Next()
						if !isRevExistent {
							break
						}
						revErr := gh.appendReversalData(rev)
						if revErr != nil {
							return internaldto.NewErroneousExecutorOutput(revErr)
						}
					}
					// if processorResponse.IsFailed() && !gh.isAwait {
					// 	processorErr = fmt.Errorf(processorResponse.GetFailedMessage())
					// }
					return gh.decorateOutput(
						internaldto.NewExecutorOutput(
							nil,
							singletonBody,
							nil,
							internaldto.NewBackendMessages(processorResponse.GetSuccessMessages()),
							processorErr,
						),
						tableName,
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
				dependentInsertPrimitive := primitive.NewGenericPrimitive(
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
		default:
			return internaldto.NewErroneousExecutorOutput(fmt.Errorf("unsupported protocol type: %v", protocolType))
		}
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
