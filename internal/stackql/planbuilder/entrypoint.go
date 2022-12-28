package planbuilder

import (
	"github.com/stackql/stackql/internal/stackql/astanalysis/earlyanalysis"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/parse"
	"github.com/stackql/stackql/internal/stackql/plan"
	"github.com/stackql/stackql/internal/stackql/primitivegenerator"
)

func BuildPlanFromContext(handlerCtx handler.HandlerContext) (*plan.Plan, error) {
	defer handlerCtx.GetGarbageCollector().Close()
	tcc, err := internaldto.NewTxnControlCounters(handlerCtx.GetTxnCounterMgr())
	handlerCtx.GetTxnStore().Put(tcc.GetTxnID())
	defer handlerCtx.GetTxnStore().Del(tcc.GetTxnID())
	logging.GetLogger().Debugf("tcc = %v\n", tcc)
	if err != nil {
		return nil, err
	}
	planKey := handlerCtx.GetQuery()
	if qp, ok := handlerCtx.GetLRUCache().Get(planKey); ok && isPlanCacheEnabled() {
		logging.GetLogger().Infoln("retrieving query plan from cache")
		pl, ok := qp.(*plan.Plan)
		if ok {
			txnId, err := handlerCtx.GetTxnCounterMgr().GetNextTxnId()
			if err != nil {
				return nil, err
			}
			pl.Instructions.SetTxnId(txnId)
			return pl, nil
		}
		return qp.(*plan.Plan), nil
	}
	qPlan := plan.NewPlan(
		handlerCtx.GetRawQuery(),
	)
	var rowSort func(map[string]map[string]interface{}) []string

	statement, err := parse.ParseQuery(handlerCtx.GetQuery())
	if err != nil {
		return createErroneousPlan(handlerCtx, qPlan, rowSort, err)
	}

	pGBuilder := newPlanGraphBuilder(handlerCtx.GetRuntimeContext().ExecutionConcurrencyLimit)

	primitiveGenerator := primitivegenerator.NewRootPrimitiveGenerator(statement, handlerCtx, pGBuilder.planGraph)

	pGBuilder.rootPrimitiveGenerator = primitiveGenerator

	earlyPassScreenerAnalyzer, err := earlyanalysis.NewEarlyScreenerAnalyzer(primitiveGenerator, nil, nil)
	if err != nil {
		return createErroneousPlan(handlerCtx, qPlan, rowSort, err)
	}
	err = earlyPassScreenerAnalyzer.Analyze(statement, handlerCtx, tcc)
	if err != nil {
		return createErroneousPlan(handlerCtx, qPlan, rowSort, err)
	}
	// TODO: full analysis of view, which will become child of top level query
	statementType := earlyPassScreenerAnalyzer.GetStatementType()
	qPlan.Type = statementType

	switch earlyPassScreenerAnalyzer.GetInstructionType() {
	case earlyanalysis.InternallyRoutableInstruction:
		createInstructionError := pGBuilder.pgInternal(earlyPassScreenerAnalyzer.GetPlanBuilderInput())
		if createInstructionError != nil {
			return nil, createInstructionError
		}
		qPlan.Instructions = pGBuilder.planGraph

		if qPlan.Instructions != nil {
			err = qPlan.Instructions.Optimise()
			if err != nil {
				return createErroneousPlan(handlerCtx, qPlan, rowSort, err)
			}
		}
		return qPlan, err
	case earlyanalysis.StandardInstruction, earlyanalysis.DummiedPGInstruction:
		createInstructionError := pGBuilder.createInstructionFor(earlyPassScreenerAnalyzer.GetPlanBuilderInput())
		if createInstructionError != nil {
			return nil, createInstructionError
		}
	case earlyanalysis.NopInstruction:
		createInstructionError := pGBuilder.nop(earlyPassScreenerAnalyzer.GetPlanBuilderInput())
		if createInstructionError != nil {
			return nil, createInstructionError
		}
	}

	if pGBuilder.planGraph.ContainsIndirect() {
		qPlan.SetCacheable(false)
	}

	qPlan.Instructions = pGBuilder.planGraph

	if qPlan.Instructions != nil {
		err = qPlan.Instructions.Optimise()
		if err != nil {
			return createErroneousPlan(handlerCtx, qPlan, rowSort, err)
		}
		if qPlan.IsCacheable() {
			handlerCtx.GetLRUCache().Set(planKey, qPlan)
		}
	}

	return qPlan, err
}
