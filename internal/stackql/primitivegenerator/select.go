package primitivegenerator

import (
	"fmt"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/dependencyplanner"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/planbuilderinput"
	"github.com/stackql/stackql/internal/stackql/primitivebuilder"
	"github.com/stackql/stackql/internal/stackql/tableinsertioncontainer"
)

func (p *standardPrimitiveGenerator) analyzeSelect(pbi planbuilderinput.PlanBuilderInput) error {

	annotatedAST := pbi.GetAnnotatedAST()

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

	selectMetadata, ok := annotatedAST.GetSelectMetadata(node)
	if !ok {
		return fmt.Errorf("could not obtain select metadata for select AST node")
	}

	var pChild PrimitiveGenerator
	var err error

	if err != nil {
		return err
	}

	tblz, hasTblz := selectMetadata.GetTableMap()
	if !hasTblz {
		return fmt.Errorf("select analysis: no table map present")
	}
	annotations, hasAnnotations := selectMetadata.GetAnnotations()
	if !hasAnnotations {
		return fmt.Errorf("select analysis not viable: no annotations present")
	}
	annotations.AssignParams()
	existingParams := annotations.GetStringParams()
	colRefs := pbi.GetColRefs()
	// END_BLOCK  ParameterHierarchy

	// BLOCK  SequencingAccrual
	dataFlows, ok := selectMetadata.GetOnConditionDataFlows()
	if !ok {
		return fmt.Errorf("could not obtain ON condition data flows for select AST node")
	}
	logging.GetLogger().Debugf("%v\n", dataFlows)
	// END_BLOCK  SequencingAccrual

	onConditionsToRewrite := selectMetadata.GetOnConditionsToRewrite()

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
			tccAheadOfTime := pbi.IsTccSetAheadOfTime()
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
				tccAheadOfTime,
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
