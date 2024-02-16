package primitivebuilder

import (
	"fmt"
	"strconv"

	"github.com/stackql/any-sdk/anysdk"
	pkg_response "github.com/stackql/any-sdk/pkg/response"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/acid/binlog"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/httpmiddleware"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/builder_input"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/primitive_context"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/provider"
	"github.com/stackql/stackql/internal/stackql/streaming/http_preparator_stream.go"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
)

type genericHTTPReversal struct {
	graphHolder       primitivegraph.PrimitiveGraphHolder
	handlerCtx        handler.HandlerContext
	drmCfg            drm.Config
	root              primitivegraph.PrimitiveNode
	op                anysdk.OperationStore
	commentDirectives sqlparser.CommentDirectives
	isAwait           bool
	verb              string // may be "insert" or "update"
	inputAlias        string
	isUndo            bool
	reversalStream    http_preparator_stream.HttpPreparatorStream
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

	return &genericHTTPReversal{
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
	}, nil
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
	handlerCtx := gh.handlerCtx
	commentDirectives := gh.commentDirectives
	isAwait := gh.isAwait
	_, _, responseAnalysisErr := m.GetResponseBodySchemaAndMediaType()
	actionPrimitive := primitive.NewHTTPRestPrimitive(
		prov,
		nil,
		nil,
		nil,
		primitive_context.NewPrimitiveContext(),
	)
	tableName := m.GetName()
	target := make(map[string]interface{})
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
				nullaryEx := func() internaldto.ExecutorOutput {
					response, apiErr := httpmiddleware.HTTPApiCallFromRequest(handlerCtx.Clone(), prov, m, req.GetRequest())
					if apiErr != nil {
						return internaldto.NewErroneousExecutorOutput(apiErr)
					}

					if responseAnalysisErr == nil {
						var resp pkg_response.Response
						processed, processErr := m.ProcessResponse(response)
						if processErr != nil {
							return internaldto.NewErroneousExecutorOutput(processErr)
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
					items, ok := target[tablemetadata.LookupSelectItemsKey(m)]
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
								[]string{"undo over HTTP successful"},
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
						generatedErr := fmt.Errorf("undo over HTTP error: %s", response.Status)
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
				err = dependentInsertPrimitive.SetExecutor(func(_ primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
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
