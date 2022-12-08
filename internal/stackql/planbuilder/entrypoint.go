package planbuilder

import (
	"github.com/stackql/stackql/internal/stackql/astanalysis/earlyanalysis"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/plan"
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

	earlyPassScreenerAnalyzer, err := earlyanalysis.NewEarlyScreenerAnalyzer()
	if err != nil {
		return createErroneousPlan(handlerCtx, qPlan, rowSort, err)
	}
	err = earlyPassScreenerAnalyzer.Analyze(handlerCtx, tcc)
	if err != nil {
		return createErroneousPlan(handlerCtx, qPlan, rowSort, err)
	}
	statementType := earlyPassScreenerAnalyzer.GetStatementType()
	qPlan.Type = statementType

	pGBuilder := newPlanGraphBuilder(handlerCtx.GetRuntimeContext().ExecutionConcurrencyLimit)

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
