package primitivebuilder

import (
	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/asynccompose"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/execution"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/builder_input"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/primitive_context"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
)

type Exec struct {
	graph         primitivegraph.PrimitiveGraphHolder
	handlerCtx    handler.HandlerContext
	drmCfg        drm.Config
	root          primitivegraph.PrimitiveNode
	tbl           tablemetadata.ExtendedTableMetadata
	isAwait       bool
	isShowResults bool
	tcc           internaldto.TxnControlCounters
}

func NewExec(
	graph primitivegraph.PrimitiveGraphHolder,
	handlerCtx handler.HandlerContext,
	node sqlparser.SQLNode, //nolint:revive // future proofing
	tbl tablemetadata.ExtendedTableMetadata,
	isAwait bool,
	isShowResults bool,
	tcc internaldto.TxnControlCounters,
) Builder {
	return &Exec{
		graph:         graph,
		handlerCtx:    handlerCtx,
		drmCfg:        handlerCtx.GetDrmConfig(),
		tbl:           tbl,
		isAwait:       isAwait,
		isShowResults: isShowResults,
		tcc:           tcc,
	}
}

func (ss *Exec) GetRoot() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *Exec) GetTail() primitivegraph.PrimitiveNode {
	return ss.root
}

//nolint:gocognit,funlen // probably a headache no matter which way you slice it
func (ss *Exec) Build() error {
	handlerCtx := ss.handlerCtx
	tbl := ss.tbl
	prov, err := tbl.GetProvider()
	if err != nil {
		return err
	}
	provider, err := prov.GetProvider()
	if err != nil {
		return err
	}
	rtCtx := handlerCtx.GetRuntimeContext()
	authCtx, authCtxErr := handlerCtx.GetAuthContext(provider.GetName())
	if authCtxErr != nil {
		return authCtxErr
	}
	outErrFile := handlerCtx.GetOutErrFile()

	m, err := tbl.GetMethod()
	if err != nil {
		return err
	}
	svc, err := tbl.GetService()
	if err != nil {
		return err
	}
	if ss.isShowResults {
		analysisInput := anysdk.NewMethodAnalysisInput(
			m,
			svc,
			true,
			[]anysdk.ColumnDescriptor{},
			false,
		)
		analyser := anysdk.NewMethodAnalyzer()
		methodAnalysisOutput, analysisErr := analyser.AnalyzeUnaryAction(analysisInput)
		if analysisErr != nil {
			return analysisErr
		}
		methodAnalysisOutput.GetInsertTabulation()
		bldrInput := builder_input.NewBuilderInput(
			ss.graph,
			handlerCtx.Clone(),
			tbl,
		)
		bldrInput.SetTxnCtrlCtrs(ss.tcc)
		deFactoSelectBuilder := NewSingleAcquireAndSelect(
			bldrInput,
			nil,
			nil,
			nil,
		)
		buildErr := deFactoSelectBuilder.Build()
		return buildErr
	}
	// isNullary := m.IsNullary()
	// var target map[string]interface{}
	//nolint:revive // no big deal
	ex := func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
		// var columnOrder []string
		// keys := make(map[string]map[string]interface{})
		httpArmoury, httpArmouryErr := tbl.GetHTTPArmoury()
		if httpArmouryErr != nil {
			return internaldto.NewErroneousExecutorOutput(httpArmouryErr)
		}
		polyHandler := execution.NewStandardPolyHandler(
			handlerCtx,
		)
		tableName, tableNameErr := tbl.GetTableName()
		if tableNameErr != nil {
			return internaldto.NewErroneousExecutorOutput(tableNameErr)
		}
		var singletonBody map[string]interface{}
		var rawMessages []string
		var readyMessages internaldto.BackendMessages
		for _, req := range httpArmoury.GetRequestParams() {
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
				!ss.isShowResults,
				true,
				ss.isAwait,
				false,
				!ss.isShowResults,
				"",
				prov.GetDefaultHTTPClient(), // for testing purposes only
			)
			processor := execution.NewProcessor(pp)
			processorResponse := processor.Process()
			processorErr := processorResponse.GetError()
			if processorErr != nil {
				return internaldto.NewErroneousExecutorOutput(processorErr)
			}
			singletonBody = processorResponse.GetSingletonBody()
			if len(processorResponse.GetSuccessMessages()) > 0 {
				rawMessages = append(rawMessages, processorResponse.GetSuccessMessages()...)
			}
		}
		if len(rawMessages) > 0 {
			readyMessages = internaldto.NewBackendMessages(rawMessages)
		}
		return internaldto.NewExecutorOutput(nil, singletonBody, nil, readyMessages, nil)
	}
	execPrimitive := primitive.NewGenericPrimitive(
		ex,
		nil,
		nil,
		primitive_context.NewPrimitiveContext(),
	)
	if !ss.isAwait {
		ss.graph.CreatePrimitiveNode(execPrimitive)
		return nil
	}
	pr, err := asynccompose.ComposeAsyncMonitor(
		handlerCtx, execPrimitive, prov, m,
		nil, false, nil, nil) // returning hardcoded to false for now
	if err != nil {
		return err
	}
	ss.graph.CreatePrimitiveNode(pr)
	return nil
}
