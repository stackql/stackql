package primitivegenerator

import (
	"fmt"
	"strings"

	"github.com/stackql/any-sdk/pkg/logging"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/astformat"
	"github.com/stackql/stackql/internal/stackql/astindirect"
	"github.com/stackql/stackql/internal/stackql/astvisit"
	"github.com/stackql/stackql/internal/stackql/dependencyplanner"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/planbuilderinput"
	"github.com/stackql/stackql/internal/stackql/primitivebuilder"
	"github.com/stackql/stackql/internal/stackql/symtab"
	"github.com/stackql/stackql/internal/stackql/tableinsertioncontainer"
)

//nolint:errcheck,funlen,gocognit,govet,gocyclo,cyclop // TODO: review
func (pb *standardPrimitiveGenerator) analyzeSelect(pbi planbuilderinput.PlanBuilderInput) error {
	annotatedAST := pbi.GetAnnotatedAST()

	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetSelect()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Select", pbi.GetStatement())
	}

	// Before analysing AST, see if we can pass straight to SQL backend
	_, isInternallyRoutable := handlerCtx.GetDBMSInternalRouter().CanRoute(annotatedAST.GetAST())
	if isInternallyRoutable { //nolint:nestif // acceptable
		selQuery := strings.ReplaceAll(astformat.String(node, handlerCtx.GetASTFormatter()), "from \"dual\"", "")
		v := astvisit.NewInternallyRoutableTypingAstVisitor(
			selQuery,
			annotatedAST,
			handlerCtx,
			nil,
			handlerCtx.GetDrmConfig(),
			handlerCtx.GetNamespaceCollection(),
		)
		visitErr := v.Visit(annotatedAST.GetAST())
		if visitErr != nil {
			return visitErr
		}
		selCtx, selCtxExists := v.GetSelectContext()
		if !selCtxExists {
			return fmt.Errorf("internal routing error: could not obtain select context")
		}
		pb.PrimitiveComposer.SetSelectPreparedStatementCtx(selCtx)
		selectIndirect, indirectError := astindirect.NewParserSelectIndirect(node, selCtx)
		if indirectError != nil {
			return indirectError
		}
		annotatedAST.SetSelectIndirect(node, selectIndirect)
		primitiveGenerator := pb
		clonedPbi := pbi.Clone()
		clonedPbi.SetRawQuery(selQuery)
		err := primitiveGenerator.AnalyzePGInternal(clonedPbi)
		if err != nil {
			return err
		}
		builder := primitiveGenerator.GetPrimitiveComposer().GetBuilder()
		if builder == nil {
			return fmt.Errorf("nil pg internal builder")
		}
		if pb.PrimitiveComposer.IsIndirect() {
			pb.SetIndirectCreateTailBuilder([]primitivebuilder.Builder{builder})
		}
		return nil
	}

	// TODO: get rid of this and dependent tests.
	// We need not emulate postgres for other backends at this stage.
	if sel, ok := planbuilderinput.IsPGSetupQuery(pbi); ok {
		if sel != nil {
			bldr := primitivebuilder.NewNativeSelect(pb.PrimitiveComposer.GetGraphHolder(), handlerCtx, sel)
			pb.PrimitiveComposer.SetBuilder(bldr)
			return nil
		}
		return pb.AnalyzeNop(pbi)
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
		pb.PrimitiveComposer.SetTable(k, v)
	}

	for i, fromExpr := range node.From {
		var leafKey interface{} = i
		switch from := fromExpr.(type) { //nolint:gocritic // TODO: review
		case *sqlparser.AliasedTableExpr:
			if from.As.GetRawVal() != "" {
				leafKey = from.As.GetRawVal()
			}
		}

		leaf, err := pb.PrimitiveComposer.GetSymTab().NewLeaf(leafKey)
		if err != nil {
			return err
		}
		pChild = pb.AddChildPrimitiveGenerator(fromExpr, leaf)

		for _, selectExpr := range node.SelectExprs {
			//nolint:gocritic // experimental
			switch expr := selectExpr.(type) {
			case *sqlparser.AliasedExpr:
				alias := expr.As.GetRawVal()
				if alias == "" {
					continue
				}
				entry := symtab.NewSymTabEntry(
					pb.PrimitiveComposer.GetDRMConfig().GetRelationalType("string"),
					"",
					alias,
				)
				pb.PrimitiveComposer.SetSymbol(alias, entry)
			}
		}

		for _, tbl := range tblz {
			err := pb.expandTable(tbl)
			_, isIndirect := tbl.GetIndirect()
			if err != nil && !isIndirect {
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
			rewrittenWhere, paramsPresent, err = pb.analyzeWhere(node.Where, existingParams)
			if err != nil {
				return err
			}
			pb.PrimitiveComposer.SetWhere(rewrittenWhere)
		}
	}
	logging.GetLogger().Debugf("len(paramsPresent) = %d\n", len(paramsPresent))
	// END_BLOCK REWRITE_WHERE

	isSimpleFrom := parserutil.IsFromExprSimple(node.From)

	if len(node.From) >= 1 && isSimpleFrom {
		switch node.From[0].(type) {
		case *sqlparser.JoinTableExpr, *sqlparser.AliasedTableExpr:
			tcc := pbi.GetTxnCtrlCtrs()
			tccAheadOfTime := pbi.IsTccSetAheadOfTime()
			dp, err := dependencyplanner.NewStandardDependencyPlanner(
				annotatedAST,
				handlerCtx,
				dataFlows,
				colRefs,
				rewrittenWhere,
				node,
				tblz,
				pb.PrimitiveComposer,
				tcc,
				tccAheadOfTime,
			)
			if err != nil {
				return err
			}
			dp = dp.WithPrepStmtOffset(pb.prepStmtOffset)
			dp = dp.WithElideRead(pb.IsElideRead())
			err = dp.Plan()
			if err != nil {
				return err
			}
			bld := dp.GetBldr()
			if pb.PrimitiveComposer.IsIndirect() {
				pb.SetIndirectCreateTailBuilder([]primitivebuilder.Builder{bld})
			}
			selCtx := dp.GetSelectCtx()
			pChild.GetPrimitiveComposer().SetBuilder(bld)
			pb.PrimitiveComposer.SetSelectPreparedStatementCtx(selCtx)
			return nil
		}
	}
	//nolint:gocritic // tactical
	if len(node.From) == 1 {
		switch ft := node.From[0].(type) {
		case *sqlparser.ExecSubquery:
			cols, err := parserutil.ExtractSelectColumnNames(node, handlerCtx.GetASTFormatter())
			if err != nil {
				return err
			}
			tbl, err := pChild.AnalyzeUnaryExec(pbi, handlerCtx, ft.Exec, node, cols)
			if err != nil {
				return err
			}
			insertionContainer, err := tableinsertioncontainer.NewTableInsertionContainer(
				tbl,
				handlerCtx.GetSQLEngine(),
				handlerCtx.GetTxnCounterMgr())
			if err != nil {
				return err
			}
			selIndirect, indirectErr := astindirect.NewParserSelectIndirect(
				node, pChild.GetPrimitiveComposer().GetSelectPreparedStatementCtx())
			if indirectErr != nil {
				return indirectErr
			}
			annotatedAST.SetSelectIndirect(node, selIndirect)
			pChild.GetPrimitiveComposer().SetBuilder(
				primitivebuilder.NewSingleAcquireAndSelect(
					pChild.GetPrimitiveComposer().GetGraphHolder(),
					pChild.GetPrimitiveComposer().GetTxnCtrlCtrs(),
					handlerCtx,
					insertionContainer,
					pChild.GetPrimitiveComposer().GetInsertPreparedStatementCtx(),
					pChild.GetPrimitiveComposer().GetSelectPreparedStatementCtx(),
					nil))
			return nil
		}
	}
	return fmt.Errorf("cannot process cartesian join select just yet")
}
