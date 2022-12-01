package planbuilder

import (
	"github.com/stackql/stackql/internal/stackql/astvisit"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/parse"
	"github.com/stackql/stackql/internal/stackql/plan"
	"vitess.io/vitess/go/vt/sqlparser"
)

func BuildPlanFromContext(handlerCtx *handler.HandlerContext) (*plan.Plan, error) {
	defer handlerCtx.GarbageCollector.Close()
	tcc, err := dto.NewTxnControlCounters(handlerCtx.TxnCounterMgr)
	handlerCtx.TxnStore.Put(tcc.GetTxnID())
	defer handlerCtx.TxnStore.Del(tcc.GetTxnID())
	logging.GetLogger().Debugf("tcc = %v\n", tcc)
	if err != nil {
		return nil, err
	}
	planKey := handlerCtx.Query
	if qp, ok := handlerCtx.LRUCache.Get(planKey); ok && isPlanCacheEnabled() {
		logging.GetLogger().Infoln("retrieving query plan from cache")
		pl, ok := qp.(*plan.Plan)
		if ok {
			txnId, err := handlerCtx.TxnCounterMgr.GetNextTxnId()
			if err != nil {
				return nil, err
			}
			pl.Instructions.SetTxnId(txnId)
			return pl, nil
		}
		return qp.(*plan.Plan), nil
	}
	qPlan := plan.NewPlan(
		handlerCtx.RawQuery,
	)
	var rowSort func(map[string]map[string]interface{}) []string
	var statement sqlparser.Statement
	statement, err = parse.ParseQuery(handlerCtx.Query)
	if err != nil {
		return createErroneousPlan(handlerCtx, qPlan, rowSort, err)
	}
	result, err := sqlparser.RewriteAST(statement)
	if err != nil {
		return createErroneousPlan(handlerCtx, qPlan, rowSort, err)
	}
	statementType := sqlparser.ASTToStatementType(result.AST)
	if err != nil {
		return createErroneousPlan(handlerCtx, qPlan, rowSort, err)
	}
	qPlan.Type = statementType

	pGBuilder := newPlanGraphBuilder(handlerCtx.RuntimeContext.ExecutionConcurrencyLimit)

	// Before analysing AST, see if we can pass stright to SQL backend
	opType, ok := handlerCtx.GetDBMSInternalRouter().CanRoute(result.AST)
	if ok {
		logging.GetLogger().Debugf("%v", opType)
		pbi, err := NewPlanBuilderInput(handlerCtx, result.AST, nil, nil, nil, nil, nil, tcc.Clone())
		if err != nil {
			return nil, err
		}
		createInstructionError := pGBuilder.pgInternal(pbi)
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

	}

	// First pass AST analysis; extract provider strings for auth.
	provStrSlice, cacheExemptMaterialDetected := astvisit.ExtractProviderStringsAndDetectCacheExceptMaterial(result.AST, handlerCtx.SQLDialect, handlerCtx.GetASTFormatter(), handlerCtx.GetNamespaceCollection())
	if cacheExemptMaterialDetected {
		qPlan.SetCacheable(false)
	}
	for _, p := range provStrSlice {
		_, err := handlerCtx.GetProvider(p)
		if err != nil {
			return nil, err
		}
	}
	if err != nil {
		return createErroneousPlan(handlerCtx, qPlan, rowSort, err)
	}

	ast := result.AST

	// Second pass AST analysis; extract provider strings for auth.
	// Extracts:
	//   - parser objects representing tables.
	//   - mapping of string aliases to tables.
	tVis := astvisit.NewTableExtractAstVisitor()
	tVis.Visit(ast)

	// Third pass AST analysis.
	// Accepts slice of parser table objects
	// extracted from previous analysis.
	// Extracts:
	//   - Col Refs; mapping columnar objects to tables.
	//   - Alias Map; mapping the "TableName" objects
	//     defining aliases to table objects.
	aVis := astvisit.NewTableAliasAstVisitor(tVis.GetTables())
	aVis.Visit(ast)

	// Fourth pass AST analysis.
	// Extracts:
	//   - Columnar parameters with null values.
	//     Useful for method matching.
	//     Especially for "Insert" queries.
	tpv := astvisit.NewPlaceholderParamAstVisitor("", false)
	tpv.Visit(ast)

	pbi, err := NewPlanBuilderInput(handlerCtx, ast, tVis.GetTables(), aVis.GetAliasedColumns(), tVis.GetAliasMap(), aVis.GetColRefs(), tpv.GetParameters(), tcc.Clone())
	if err != nil {
		return nil, err
	}

	if sel, ok := isPGSetupQuery(pbi); ok {
		if sel != nil {
			pbi, err := NewPlanBuilderInput(handlerCtx, result.AST, nil, nil, nil, nil, nil, tcc.Clone())
			if err != nil {
				return nil, err
			}
			createInstructionError := pGBuilder.createInstructionFor(pbi)
			if createInstructionError != nil {
				return nil, createInstructionError
			}
		} else {
			pbi, err := NewPlanBuilderInput(handlerCtx, nil, nil, nil, nil, nil, nil, tcc.Clone())
			if err != nil {
				return nil, err
			}
			createInstructionError := pGBuilder.nop(pbi)
			if createInstructionError != nil {
				return nil, createInstructionError
			}
		}
	}

	createInstructionError := pGBuilder.createInstructionFor(pbi)
	if createInstructionError != nil {
		return nil, createInstructionError
	}

	qPlan.Instructions = pGBuilder.planGraph

	if qPlan.Instructions != nil {
		err = qPlan.Instructions.Optimise()
		if err != nil {
			return createErroneousPlan(handlerCtx, qPlan, rowSort, err)
		}
		if qPlan.IsCacheable() {
			handlerCtx.LRUCache.Set(planKey, qPlan)
		}
	}

	return qPlan, err
}
