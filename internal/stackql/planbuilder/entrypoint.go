package planbuilder

import (
	"github.com/stackql/stackql/internal/stackql/acid/txn_context"
	"github.com/stackql/stackql/internal/stackql/astanalysis/earlyanalysis"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/parser"
	"github.com/stackql/stackql/internal/stackql/plan"
	"github.com/stackql/stackql/internal/stackql/primitivegenerator"
)

var (
	_ PlanBuilder = &standardPlanBuilder{}
)

type PlanBuilder interface {
	BuildPlanFromContext(handlerCtx handler.HandlerContext) (plan.Plan, error)
}

func NewPlanBuilder(transactionContext txn_context.ITransactionContext) PlanBuilder {
	return &standardPlanBuilder{
		transactionContext: transactionContext,
	}
}

type standardPlanBuilder struct {
	transactionContext txn_context.ITransactionContext
}

func (pb *standardPlanBuilder) BuildUndoPlanFromContext(_ handler.HandlerContext) (plan.Plan, error) {
	return nil, nil
}

//nolint:funlen,gocognit // no big deal
func (pb *standardPlanBuilder) BuildPlanFromContext(handlerCtx handler.HandlerContext) (plan.Plan, error) {
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
		pl, plOk := qp.(plan.Plan)
		if plOk {
			txnID, tErr := handlerCtx.GetTxnCounterMgr().GetNextTxnID()
			if tErr != nil {
				return nil, tErr
			}
			pl.SetTxnID(txnID)
			return pl, nil
		}
		return qp.(plan.Plan), nil
	}
	qPlan := plan.NewPlan(
		handlerCtx.GetRawQuery(),
	)
	var rowSort func(map[string]map[string]interface{}) []string

	sqlParser, err := parser.NewParser()
	if err != nil {
		return nil, err
	}
	statement, err := sqlParser.ParseQuery(handlerCtx.GetQuery())
	if err != nil {
		return createErroneousPlan(handlerCtx, qPlan, rowSort, err)
	}

	pGBuilder := newPlanGraphBuilder(handlerCtx.GetRuntimeContext().ExecutionConcurrencyLimit, pb.transactionContext)

	primitiveGenerator := primitivegenerator.NewRootPrimitiveGenerator(
		statement, handlerCtx, pGBuilder.getPlanGraphHolder())

	pGBuilder.setRootPrimitiveGenerator(primitiveGenerator)

	earlyPassScreenerAnalyzer, err := earlyanalysis.NewEarlyScreenerAnalyzer(primitiveGenerator, nil, nil, 0)
	if err != nil {
		return createErroneousPlan(handlerCtx, qPlan, rowSort, err)
	}
	err = earlyPassScreenerAnalyzer.Analyze(statement, handlerCtx, tcc)
	if err != nil {
		return createErroneousPlan(handlerCtx, qPlan, rowSort, err)
	}
	// TODO: full analysis of view, which will become child of top level query
	statementType := earlyPassScreenerAnalyzer.GetStatementType()
	qPlan.SetType(statementType)

	isReadOnlyFromEarlyPasses := earlyPassScreenerAnalyzer.IsReadOnly()
	qPlan.SetReadOnly(isReadOnlyFromEarlyPasses)

	qPlan.SetStatement(earlyPassScreenerAnalyzer.GetStatement())

	switch earlyPassScreenerAnalyzer.GetInstructionType() { //nolint:exhaustive // acceptable
	case earlyanalysis.InternallyRoutableInstruction:
		qPlan.SetReadOnly(true)
		createInstructionError := pGBuilder.pgInternal(earlyPassScreenerAnalyzer.GetPlanBuilderInput())
		if createInstructionError != nil {
			return nil, createInstructionError
		}
		qPlan.SetInstructions(pGBuilder.getPlanGraphHolder())

		if qPlan.GetInstructions() != nil {
			err = qPlan.GetInstructions().GetPrimitiveGraph().Optimise()
			if err != nil {
				return createErroneousPlan(handlerCtx, qPlan, rowSort, err)
			}
		}
		return qPlan, err
	case earlyanalysis.StandardInstruction:
		createInstructionError := pGBuilder.createInstructionFor(earlyPassScreenerAnalyzer.GetPlanBuilderInput())
		if createInstructionError != nil {
			return nil, createInstructionError
		}
	case earlyanalysis.DummiedPGInstruction:
		qPlan.SetReadOnly(true)
		createInstructionError := pGBuilder.createInstructionFor(earlyPassScreenerAnalyzer.GetPlanBuilderInput())
		if createInstructionError != nil {
			return nil, createInstructionError
		}
	case earlyanalysis.NopInstruction:
		qPlan.SetReadOnly(true)
		createInstructionError := pGBuilder.nop(earlyPassScreenerAnalyzer.GetPlanBuilderInput())
		if createInstructionError != nil {
			return nil, createInstructionError
		}
	}

	if pGBuilder.getPlanGraphHolder().ContainsIndirect() {
		qPlan.SetCacheable(false)
	}

	qPlan.SetInstructions(pGBuilder.getPlanGraphHolder())

	if qPlan.GetInstructions() != nil {
		err = qPlan.GetInstructions().GetPrimitiveGraph().Optimise()
		if err != nil {
			return createErroneousPlan(handlerCtx, qPlan, rowSort, err)
		}
		if qPlan.IsCacheable() {
			handlerCtx.GetLRUCache().Set(planKey, qPlan)
		}
	}

	return qPlan, err
}
