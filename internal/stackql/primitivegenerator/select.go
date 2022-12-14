package primitivegenerator

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/astvisit"
	"github.com/stackql/stackql/internal/stackql/dependencyplanner"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/planbuilderinput"
	"github.com/stackql/stackql/internal/stackql/primitivebuilder"
	"github.com/stackql/stackql/internal/stackql/router"
	"github.com/stackql/stackql/internal/stackql/tableinsertioncontainer"
	"vitess.io/vitess/go/vt/sqlparser"
)

func (p *standardPrimitiveGenerator) analyzeSelect(pbi planbuilderinput.PlanBuilderInput) error {

	annotatedAST := pbi.GetAnnotatedAST()

	allIndirects := annotatedAST.GetIndirects()

	for k, v := range allIndirects {
		// planbuilderinput.NewPlanBuilderInput(
		// 	annotatedAST,
		// 	pbi.GetHandlerCtx(),
		// 	v.GetSelectAST(),
		// )
		// p.analyzeSelect()
		logging.GetLogger().Debugf("indirect k = '%s', v = '%v'\n", k, v)
	}

	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetSelect()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Select", pbi.GetStatement())
	}

	// TODO: get rid of this and dependent tests.
	// We need not emulate postgres for other backends at this stage.
	if sel, ok := planbuilderinput.IsPGSetupQuery(pbi); ok {
		if sel != nil {
			bldr := primitivebuilder.NewNativeSelect(p.PrimitiveComposer.GetGraph(), handlerCtx, sel)
			p.PrimitiveComposer.SetBuilder(bldr)
			return nil
		}
		return p.AnalyzeNop(pbi)
	}

	var pChild PrimitiveGenerator
	var err error

	// BLOCK  ParameterHierarchy
	// The AST analysis passes extract parameters
	// prior to the assembly of hierarchies.
	// This is a chicken and egg scenario:
	//   - we need hierarchies a priori for temporal
	//     dependencies between tables.
	//   - we need parameters to determine hierarchy (for now).
	//   - parameters may refer to tables and we want to reference
	//     this for semantic analysis and later temporal sequencing,
	//     data flow semantics.
	//   - TODO: so... will need to split this up into multiple passes;
	//     parameters will need to have Hierarchies attached after they are inferred.
	//     Then semantic anlaysis and data flow can be instrumented.
	//   - TODO: add support for views and subqueries.
	whereParamMap := astvisit.ExtractParamsFromWhereClause(annotatedAST, node.Where)
	onParamMap := astvisit.ExtractParamsFromFromClause(annotatedAST, node.From)

	// TODO: There is god awful object <-> namespacing inside here: abstract it.
	paramRouter := router.NewParameterRouter(
		annotatedAST,
		pbi.GetAliasedTables(),
		pbi.GetAssignedAliasedColumns(),
		whereParamMap,
		onParamMap,
		pbi.GetColRefs(),
		handlerCtx.GetNamespaceCollection(),
		handlerCtx.GetASTFormatter(),
	)

	// TODO: Do the proper SOLID treatment on router, etc.
	// Might need to split into multiple passes.
	v := router.NewTableRouteAstVisitor(pbi.GetHandlerCtx(), paramRouter)

	err = v.Visit(pbi.GetStatement())

	if err != nil {
		return err
	}

	tblz := v.GetTableMap()
	annotations := v.GetAnnotations()
	annotations.AssignParams()
	existingParams := annotations.GetStringParams()
	colRefs := pbi.GetColRefs()
	// END_BLOCK  ParameterHierarchy

	// BLOCK  SequencingAccrual
	dataFlows, err := paramRouter.GetOnConditionDataFlows()
	logging.GetLogger().Debugf("%v\n", dataFlows)
	// END_BLOCK  SequencingAccrual

	onConditionsToRewrite := paramRouter.GetOnConditionsToRewrite()

	parserutil.NaiveRewriteComparisonExprs(onConditionsToRewrite)

	if err != nil {
		return err
	}

	for k, v := range tblz {
		p.PrimitiveComposer.SetTable(k, v)
	}

	for i, fromExpr := range node.From {
		var leafKey interface{} = i
		switch from := fromExpr.(type) {
		case *sqlparser.AliasedTableExpr:
			if from.As.GetRawVal() != "" {
				leafKey = from.As.GetRawVal()
			}
		}

		leaf, err := p.PrimitiveComposer.GetSymTab().NewLeaf(leafKey)
		if err != nil {
			return err
		}
		pChild = p.AddChildPrimitiveGenerator(fromExpr, leaf)

		for _, tbl := range tblz {
			err := p.expandTable(tbl)
			if err != nil {
				return err
			}
		}
	}

	// BLOCK REWRITE_WHERE
	// TODO: fix this hack
	// might make sense to implement an "all in one"
	// query rewrite as an AST visitor.
	var rewrittenWhere *sqlparser.Where
	var paramsPresent []string
	if len(node.From) == 1 {
		switch ft := node.From[0].(type) {
		case *sqlparser.ExecSubquery:
			logging.GetLogger().Infoln(fmt.Sprintf("%v", ft))
		default:
			rewrittenWhere, paramsPresent, err = p.analyzeWhere(node.Where, existingParams)
			if err != nil {
				return err
			}
			p.PrimitiveComposer.SetWhere(rewrittenWhere)
		}
	}
	logging.GetLogger().Debugf("len(paramsPresent) = %d\n", len(paramsPresent))
	// END_BLOCK REWRITE_WHERE

	if len(node.From) == 1 {
		switch ft := node.From[0].(type) {
		case *sqlparser.JoinTableExpr, *sqlparser.AliasedTableExpr:
			tcc := pbi.GetTxnCtrlCtrs()
			dp, err := dependencyplanner.NewStandardDependencyPlanner(
				annotatedAST,
				handlerCtx,
				dataFlows,
				colRefs,
				rewrittenWhere,
				pbi.GetStatement(),
				tblz,
				p.PrimitiveComposer,
				tcc,
			)
			if err != nil {
				return err
			}
			err = dp.Plan()
			if err != nil {
				return err
			}
			bld := dp.GetBldr()
			selCtx := dp.GetSelectCtx()
			pChild.GetPrimitiveComposer().SetBuilder(bld)
			p.PrimitiveComposer.SetSelectPreparedStatementCtx(selCtx)
			return nil
		case *sqlparser.ExecSubquery:
			cols, err := parserutil.ExtractSelectColumnNames(node, handlerCtx.GetASTFormatter())
			if err != nil {
				return err
			}
			tbl, err := pChild.AnalyzeUnaryExec(pbi, handlerCtx, ft.Exec, node, cols)
			if err != nil {
				return err
			}
			insertionContainer, err := tableinsertioncontainer.NewTableInsertionContainer(tbl, handlerCtx.GetSQLEngine())
			if err != nil {
				return err
			}
			pChild.GetPrimitiveComposer().SetBuilder(primitivebuilder.NewSingleAcquireAndSelect(pChild.GetPrimitiveComposer().GetGraph(), pChild.GetPrimitiveComposer().GetTxnCtrlCtrs(), handlerCtx, insertionContainer, pChild.GetPrimitiveComposer().GetInsertPreparedStatementCtx(), pChild.GetPrimitiveComposer().GetSelectPreparedStatementCtx(), nil))
			return nil
		}

	}
	return fmt.Errorf("cannot process cartesian join select just yet")
}
