package routeanalysis

import (
	"fmt"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/astanalysis/selectmetadata"
	"github.com/stackql/stackql/internal/stackql/astvisit"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/planbuilderinput"
	"github.com/stackql/stackql/internal/stackql/router"
)

var (
	_ RoutePass = &standardSelectRoutePass{}
)

// TODO: must accomodate parent (and indeed ancestor) clause parameter passing.
type RoutePass interface {
	RoutePass() error
	GetPlanBuilderInput() planbuilderinput.PlanBuilderInput
	IsPGInternalOnly() bool
}

func NewSelectRoutePass(
	node sqlparser.SelectStatement,
	pbi planbuilderinput.PlanBuilderInput,
	parentWhereParams parserutil.ParameterMap,
) RoutePass {
	return &standardSelectRoutePass{
		inputPbi:          pbi,
		handlerCtx:        pbi.GetHandlerCtx(),
		node:              node,
		parentWhereParams: parentWhereParams,
	}
}

type standardSelectRoutePass struct {
	inputPbi          planbuilderinput.PlanBuilderInput
	outputPbi         planbuilderinput.PlanBuilderInput
	handlerCtx        handler.HandlerContext
	node              sqlparser.SelectStatement
	parentWhereParams parserutil.ParameterMap
	isPGInternalOnly  bool
}

func (sp *standardSelectRoutePass) IsPGInternalOnly() bool {
	return sp.isPGInternalOnly
}

func (sp *standardSelectRoutePass) GetPlanBuilderInput() planbuilderinput.PlanBuilderInput {
	return sp.outputPbi
}

//nolint:funlen,gocognit // defer uplifts on analysers
func (sp *standardSelectRoutePass) RoutePass() error {
	var node *sqlparser.Select

	pbi := sp.inputPbi.Clone()

	// counters := pbi.GetTxnCtrlCtrs()

	switch n := sp.node.(type) {
	case *sqlparser.Select:
		node = n
	case *sqlparser.ParenSelect:
		routePass := NewSelectRoutePass(n.Select, sp.inputPbi, sp.parentWhereParams)
		err := routePass.RoutePass()
		sp.isPGInternalOnly = routePass.IsPGInternalOnly()
		sp.outputPbi = pbi
		return err
	case *sqlparser.Union:
		routePass := NewSelectRoutePass(n.FirstStatement, pbi, sp.parentWhereParams)
		err := routePass.RoutePass()
		if err != nil {
			return err
		}
		lhsPGInternalOnly := routePass.IsPGInternalOnly()
		// TODO: eventualy accomodate sharing pg native stuff to
		//       mix and match with stackql stuff.
		var rhsNonPGInternalDetected bool
		rhsPbi := pbi
		var hasPbi bool
		for _, s := range n.UnionSelects {
			rhsPbi, hasPbi = rhsPbi.Next()
			if !hasPbi {
				return fmt.Errorf("no more PBIs for union selects")
			}
			// ctrClone := counters.CloneAndIncrementInsertID()
			// rhsPbi.SetTxnCtrlCtrs(ctrClone)
			routePass := NewSelectRoutePass(s.Statement, rhsPbi, sp.parentWhereParams) //nolint:govet // intentional shadow
			err = routePass.RoutePass()
			if err != nil {
				return err
			}
			if !routePass.IsPGInternalOnly() {
				rhsNonPGInternalDetected = true
			}
		}

		sp.isPGInternalOnly = lhsPGInternalOnly && !rhsNonPGInternalDetected && len(n.UnionSelects) > 0
		sp.outputPbi = pbi
		return nil
	}

	handlerCtx := sp.handlerCtx

	annotatedAST := pbi.GetAnnotatedAST()

	// TODO: get rid of this and dependent tests.
	// We need not emulate postgres for other backends at this stage.
	if sel, ok := planbuilderinput.IsPGSetupQuery(pbi); ok {
		if sel != nil {
			return nil
		}
		return nil
	}

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
	whereParamMap, ok := pbi.GetAnnotatedAST().GetWhereParamMapsEntry(node.Where)
	if !ok {
		whereParamMap = astvisit.ExtractParamsFromWhereClause(annotatedAST, node.Where)
	}
	whereParamMap.Merge(sp.parentWhereParams)
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
		handlerCtx.GetDataFlowCfg(),
	)

	// TODO: Do the proper SOLID treatment on router, etc.
	// Might need to split into multiple passes.
	v := router.NewTableRouteAstVisitor(pbi.GetHandlerCtx(), paramRouter)

	err := v.Visit(node)

	if err != nil {
		return err
	}

	pbi = pbi.WithParameterRouter(paramRouter)

	pbi = pbi.WithTableRouteVisitor(v)

	onConditionsToRewrite := v.GetParameterRouter().GetOnConditionsToRewrite()
	// TODO: ensure this only contains actual data flows, not normal on conitions
	onConditionDataFlows, err := v.GetParameterRouter().GetOnConditionDataFlows()
	if err != nil {
		return err
	}

	selectMetadata := selectmetadata.NewSelectMetadata(
		onConditionDataFlows,
		onConditionsToRewrite,
		v.GetTableMap(),
		v.GetAnnotations(),
	)

	annotatedAST.SetSelectMetadata(node, selectMetadata)

	sp.outputPbi = pbi

	return nil
}
