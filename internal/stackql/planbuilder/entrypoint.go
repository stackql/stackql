package planbuilder

import (
	"fmt"

	"github.com/stackql/any-sdk/pkg/logging"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/acid/txn_context"
	"github.com/stackql/stackql/internal/stackql/astanalysis/earlyanalysis"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/parser"
	"github.com/stackql/stackql/internal/stackql/parserutil"
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

//nolint:nilnil // TODO: sweep through tech debt
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
	//nolint:gocritic // acceptable
	switch stmt := statement.(type) {
	case *sqlparser.RefreshMaterializedView:
		relationName := stmt.ViewName.GetRawVal()
		catalogueEntry, catalogueEntryExists := handlerCtx.GetSQLSystem().GetMaterializedViewByName(relationName)
		if !catalogueEntryExists {
			return createErroneousPlan(
				handlerCtx, qPlan, rowSort,
				fmt.Errorf("could not find materialized view '%s' to refresh", relationName))
		}
		rawQuery := catalogueEntry.GetRawQuery()
		implicitStatement, stmtErr := sqlParser.ParseQuery(rawQuery)
		if stmtErr != nil {
			return createErroneousPlan(handlerCtx, qPlan, rowSort, stmtErr)
		}
		implicitSelectStatement, isSelect := parserutil.ExtractSelectStatmentFromDDL(implicitStatement)
		if !isSelect {
			return createErroneousPlan(
				handlerCtx, qPlan, rowSort,
				fmt.Errorf("could not find implicit select statement for materialized view '%s' to refresh", relationName))
		}
		stmt.ImplicitSelect = implicitSelectStatement
		statement = stmt
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
	prebuiltIndirect, prebuiltIndirectExists := earlyPassScreenerAnalyzer.GetIndirectCreateTail()
	if prebuiltIndirectExists {
		pGBuilder.setPrebuiltIndirect(prebuiltIndirect)
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

	if pGBuilder.getPlanGraphHolder().ContainsUserManagedRelation() {
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
