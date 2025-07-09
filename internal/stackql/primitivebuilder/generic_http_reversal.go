package primitivebuilder

import (
	"fmt"

	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/acid/binlog"
	"github.com/stackql/stackql/internal/stackql/asynccompose"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/execution"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/builder_input"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/primitive_context"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/provider"
)

type genericHTTPReversal struct {
	graphHolder       primitivegraph.PrimitiveGraphHolder
	handlerCtx        handler.HandlerContext
	drmCfg            drm.Config
	insertCtx         drm.PreparedStatementCtx
	root              primitivegraph.PrimitiveNode
	op                anysdk.OperationStore
	commentDirectives sqlparser.CommentDirectives
	isAwait           bool
	isReturning       bool
	verb              string // may be "insert" or "update"
	inputAlias        string
	isUndo            bool
	reversalStream    anysdk.HttpPreparatorStream
	prov              provider.IProvider
}

func newGenericHTTPReversal(
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
	op, opExists := builderInput.GetOperationStore()
	if !opExists {
		return nil, fmt.Errorf("operation store is required")
	}
	prepStream, prepStreamExists := builderInput.GetHTTPPreparatorStream()
	if !prepStreamExists {
		return nil, fmt.Errorf("preparator stream is required")
	}
	prov, provExists := builderInput.GetProvider()
	if !provExists {
		return nil, fmt.Errorf("provider is required")
	}

	rv := &genericHTTPReversal{
		prov:           prov,
		graphHolder:    graphHolder,
		handlerCtx:     handlerCtx,
		drmCfg:         handlerCtx.GetDrmConfig(),
		op:             op,
		reversalStream: prepStream,
		isAwait:        builderInput.IsAwait(),
		verb:           builderInput.GetVerb(),
		inputAlias:     builderInput.GetInputAlias(),
		isUndo:         builderInput.IsUndo(),
	}
	insertCtx, insertCtxExists := builderInput.GetInsertCtx()
	if insertCtxExists {
		rv.insertCtx = insertCtx
	}
	return rv, nil
}

func (gh *genericHTTPReversal) GetRoot() primitivegraph.PrimitiveNode {
	return gh.root
}

func (gh *genericHTTPReversal) GetTail() primitivegraph.PrimitiveNode {
	return gh.root
}

func (gh *genericHTTPReversal) decorateOutput(
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

//nolint:funlen,gocognit // TODO: fix this
func (gh *genericHTTPReversal) Build() error {
	m := gh.op
	prov := gh.prov
	provider, providerErr := prov.GetProvider()
	if providerErr != nil {
		return providerErr
	}
	handlerCtx := gh.handlerCtx
	rtCtx := handlerCtx.GetRuntimeContext()
	authCtx, authCtxErr := handlerCtx.GetAuthContext(provider.GetName())
	if authCtxErr != nil {
		return authCtxErr
	}
	outErrFile := handlerCtx.GetOutErrFile()
	commentDirectives := gh.commentDirectives
	isAwait := gh.isAwait
	_, _, responseAnalysisErr := m.GetResponseBodySchemaAndMediaType()
	actionPrimitive := primitive.NewGenericPrimitive(
		nil,
		nil,
		nil,
		primitive_context.NewPrimitiveContext(),
	)
	tableName := m.GetName()
	// target := make(map[string]interface{})
	ex := func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
		httpPreparator, httpPreparatorExists := gh.reversalStream.Next()
		resultSet := internaldto.NewErroneousExecutorOutput(fmt.Errorf("no executions detected"))
		var err error
		for {
			if !httpPreparatorExists {
				break
			}
			httpArmoury, httpErr := httpPreparator.BuildHTTPRequestCtx()
			if httpErr != nil {
				return internaldto.NewErroneousExecutorOutput(httpErr)
			}

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
						true,
						"undo",
					)
					processor := execution.NewProcessor(pp)
					processorResponse := processor.Process()
					processorErr := processorResponse.GetError()
					singletonBody := processorResponse.GetSingletonBody()
					// if processorResponse.IsFailed() && !gh.isAwait {
					// 	processorErr = fmt.Errorf(processorResponse.GetFailedMessage())
					// }
					msgs := internaldto.NewBackendMessages(processorResponse.GetSuccessMessages())
					return gh.decorateOutput(
						internaldto.NewExecutorOutput(
							nil,
							singletonBody,
							nil,
							msgs,
							processorErr,
						),
						tableName,
					)
				}

				nullaryExecutors = append(nullaryExecutors, nullaryEx)
			}
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
				execPrim, execErr := asynccompose.ComposeAsyncMonitor(
					handlerCtx, dependentInsertPrimitive, prov, m, commentDirectives, gh.isReturning, gh.insertCtx, gh.drmCfg)
				if execErr != nil {
					return internaldto.NewErroneousExecutorOutput(execErr)
				}
				resultSet = execPrim.Execute(pc)
				if resultSet.GetError() != nil {
					return resultSet
				}
			}
			httpPreparator, httpPreparatorExists = gh.reversalStream.Next()
		}
		return gh.decorateOutput(
			resultSet,
			tableName,
		)
	}
	err := actionPrimitive.SetExecutor(ex)
	if err != nil {
		return err
	}

	graphHolder := gh.graphHolder

	// actionNode := graphHolder.CreatePrimitiveNode(actionPrimitive)

	// actionPrimitive.SetInputAlias(gh.inputAlias, gh.dependencyNode.ID())

	actionNode := graphHolder.CreateInversePrimitiveNode(actionPrimitive)
	gh.root = actionNode
	return nil
}
